package server

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	files "github.com/ipfs/go-ipfs-files"
	options "github.com/ipfs/interface-go-ipfs-core/options"
	content "github.com/joeltg/negotiate/content"
	ld "github.com/underlay/json-gold/ld"

	types "github.com/underlay/pkgs/types"
	vocab "github.com/underlay/pkgs/vocab"
)

var ErrNoContentType = fmt.Errorf("Content-Type is required for files")
var ErrFileExists = fmt.Errorf("A resource with that name already exists")

// https://tools.ietf.org/html/rfc3986#section-3.3
var pathSegment = regexp.MustCompile("^(?:[a-zA-Z0-9\\-\\._~!$'\\(\\)\\*\\+,;=:@%*]|(?:[A-F0-9]{2}))+$")

// Post handles HTTP POST requests
func (server *Server) Post(ctx context.Context, res http.ResponseWriter, req *http.Request) error {
	pathname := req.URL.Path
	if pathname == "/" {
	} else if !PathRegex.MatchString(pathname) {
		res.WriteHeader(404)
		return nil
	}

	defaultOffer := "application/n-quads"
	accept := content.NegotiateContentType(req, []string{defaultOffer, "application/ld+json"}, defaultOffer)

	if accept != defaultOffer && accept != "application/ld+json" {
		res.WriteHeader(406)
		return nil
	}

	// Should we require If-Match? I bet we should.
	ifMatch := req.Header.Get("If-Match")
	if !etagRegex.MatchString(ifMatch) {
		res.WriteHeader(412)
		return nil
	}

	match := etagRegex.FindStringSubmatch(ifMatch)[1]

	err := req.ParseMultipartForm(0)
	if err != nil {
		return err
	}

	err = server.db.Update(func(txn *badger.Txn) error {
		p, err := types.GetPackage(pathname, txn)
		if err != nil {
			return err
		}

		oldURI := p.URI()

		_, etag := p.ETag()
		if etag != match {
			res.WriteHeader(412)
			return nil
		}

		members := make(map[string]bool, len(p.Member))
		for _, member := range p.Member {
			members[member] = true
		}

		id, value, err := p.Paths()
		if err != nil {
			return err
		}

		nextValue := value

		newFiles := map[string]*types.File{}

		for _, headers := range req.MultipartForm.File {
			for _, header := range headers {
				format := header.Header.Get("Content-Type")
				if format == "" {
					res.WriteHeader(400)
					return ErrNoContentType
				}

				file, err := header.Open()
				if err != nil {
					return err
				}

				resolved, err := server.fs.Add(
					ctx,
					files.NewReaderFile(file),
					options.Unixfs.Pin(false),
					options.Unixfs.CidVersion(1),
					options.Unixfs.RawLeaves(true),
				)

				if err != nil {
					return err
				}

				f := &types.File{
					Value:  resolved.Cid().Bytes(),
					Extent: uint64(header.Size),
					Format: format,
				}

				disposition := header.Header.Get("Content-Disposition")
				_, params, err := mime.ParseMediaType(disposition)
				if err != nil {
					return err
				}

				var name string
				if fn, has := params["filename"]; has && pathSegment.MatchString(fn) {
					name = fn
				} else {
					_, name = f.ETag()
				}

				if _, has := members[name]; has {
					res.WriteHeader(409)
					return ErrFileExists
				}

				newFiles[name] = f
				var newPathname string
				if pathname == "/" {
					newPathname = "/" + name
				} else {
					newPathname = fmt.Sprintf("%s/%s", pathname, name)
				}

				err = types.SetResource(f, newPathname, txn)
				if err != nil {
					return err
				}

				p.Member = append(p.Member, name)

				nextValue, err = server.object.AddLink(ctx, nextValue, name, resolved)
				if err != nil {
					return err
				}
			}
		}

		if nextValue == value {
			res.WriteHeader(204)
			return nil
		}

		modified := time.Now()
		// It's important to set the file resources first, before
		// percolating the modified packages
		err = server.percolate(ctx, modified, pathname, p, id, value, nextValue, txn)
		if err != nil {
			return err
		}

		_, newETag := p.ETag()

		res.Header().Add("Content-Type", accept)
		res.Header().Add("Link", fmt.Sprintf(`<#%s>; rel="self"`, p.Subject))
		res.Header().Add("ETag", fmt.Sprintf(`"%s"`, newETag))
		res.Header().Add("Content-Type", accept)

		s := ld.NewBlankNode(p.Subject)
		rdf := ld.NewRDFDataset()
		rdf.Graphs["@default"] = append(
			rdf.Graphs["@default"],
			ld.NewQuad(s, vocab.PROVwasRevisionOf, ld.NewIRI(oldURI), ""),
			ld.NewQuad(s, vocab.DCTERMSmodified, ld.NewLiteral(p.Modified, vocab.XSDdateTime, ""), ""),
		)

		for name, file := range newFiles {
			uri := ld.NewIRI(file.URI())
			extent := strconv.FormatUint(file.Extent, 10)
			rdf.Graphs["@default"] = append(
				rdf.Graphs["@default"],
				ld.NewQuad(s, vocab.PROVhadMember, uri, ""),
				ld.NewQuad(uri, vocab.DCTERMSformat, ld.NewLiteral(file.Format, "", ""), ""),
				ld.NewQuad(uri, vocab.DCTERMSextent, ld.NewLiteral(extent, ld.XSDInteger, ""), ""),
			)

			_, etag := file.ETag()
			if etag != name {
				resource := fmt.Sprintf("%s/%s", p.Resource, name)
				rdf.Graphs["@default"] = append(
					rdf.Graphs["@default"],
					ld.NewQuad(uri, vocab.LDPmembershipResource, ld.NewIRI(resource), ""),
					ld.NewQuad(uri, vocab.DCTERMStitle, ld.NewLiteral(name, "", ""), ""),
				)
			}
		}

		if accept == defaultOffer {
			res.WriteHeader(200)
			ns := &ld.NQuadRDFSerializer{}
			_ = ns.SerializeTo(res, rdf)
		} else if accept == "application/ld+json" {
			proc := ld.NewJsonLdProcessor()
			doc, err := proc.FromRDF(rdf, server.opts)
			if err != nil {
				return err
			}
			res.WriteHeader(200)
			json.NewEncoder(res).Encode(doc)
		}
		return nil
	})

	if err == badger.ErrKeyNotFound {
		res.WriteHeader(404)
		return nil
	} else if err == types.ErrNotPackage {
		res.WriteHeader(405)
		return nil
	} else if err != nil {
		res.WriteHeader(500)
		return err
	}

	return nil
}
