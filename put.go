package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	files "github.com/ipfs/go-ipfs-files"
	ld "github.com/piprate/json-gold/ld"
	rdf "github.com/underlay/go-rdfjs"
	types "github.com/underlay/pkgs/types"
	styx "github.com/underlay/styx"
)

// Put handles HTTP PUT requests
func (server *Server) Put(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	self, t := types.ParseLinks(req.Header["Link"])
	key := types.ParsePath(req.URL.Path)
	var name string
	if len(key) > 0 {
		name = key[len(key)-1]
	}

	resource := types.GetURI(server.resource, key)

	var r types.Resource
	var err error
	format := req.Header.Get("Content-Type")
	timestamp := time.Now().Format(time.RFC3339)
	switch t {
	case types.PackageType:
		if format == "" && self != "" && types.PackageURIPattern.MatchString(self) {
			reference := &types.Reference{ID: self, Resource: resource, Title: name}
			r, err = server.parse(ctx, reference)
			if err != nil {
				res.WriteHeader(400)
				res.Write([]byte(err.Error()))
				return
			}
			break
		} else if format == offers[0] || format == offers[1] || format == offers[2] {
			doc, err := parseContent(format, resource, req.Body)
			if err != nil {
				res.WriteHeader(400)
				res.Write([]byte(err.Error()))
				return
			}

			pkg, err := server.framePackage(resource, doc)
			if err != nil {
				res.WriteHeader(400)
				res.Write([]byte(err.Error()))
				return
			}

			_, err = server.normalize(ctx, pkg)
			if err != nil {
				res.WriteHeader(400)
				res.Write([]byte(err.Error()))
				return
			}
			r = pkg
		} else {
			res.WriteHeader(415)
			return
		}
	case types.AssertionType:
		if format == "" && self != "" && types.AssertionURIPattern.MatchString(self) {
			r = &types.Assertion{ID: self, Resource: resource, Title: name, Created: timestamp, Modified: timestamp}
			break
		} else if format == offers[0] || format == offers[1] || format == offers[2] {
			doc, err := parseContent(format, resource, req.Body)
			if err != nil {
				res.WriteHeader(400)
				res.Write([]byte(err.Error()))
				return
			}

			opts := ld.NewJsonLdOptions(resource)
			opts.DocumentLoader = server.documentLoader
			opts.Algorithm = "URDNA2015"
			opts.Format = "application/n-quads"
			normalized, err := ld.NewJsonLdProcessor().Normalize(doc, opts)
			if err != nil {
				res.WriteHeader(400)
				res.Write([]byte(err.Error()))
				return
			}

			a := &types.Assertion{Resource: resource, Title: name, Created: timestamp, Modified: timestamp}
			node := files.NewBytesFile([]byte(normalized.(string)))
			err = server.setAssertion(ctx, a, node)
			if err != nil {
				res.WriteHeader(502)
				res.Write([]byte(err.Error()))
				return
			}
			r = a
		} else {
			res.WriteHeader(415)
			return
		}
	case types.FileType:
		if format == "" && self != "" && types.FileURIPattern.MatchString(self) {
			f := &types.File{ID: self, Resource: resource, Title: name, Created: timestamp, Modified: timestamp, Format: format}
			stat, err := server.api.Object().Stat(ctx, f.Path())
			if err != nil {
				res.WriteHeader(502)
				res.Write([]byte(err.Error()))
				return
			}
			f.Extent = stat.CumulativeSize
			r = f
		} else if format != "" {
			f := &types.File{Resource: resource, Title: name, Created: timestamp, Modified: timestamp, Format: format}
			err := server.setFile(ctx, f, files.NewReaderFile(req.Body))
			if err != nil {
				res.WriteHeader(502)
				return
			}
			r = f
		} else {
			res.WriteHeader(415)
			return
		}
	default:
		res.WriteHeader(400)
		return
	}

	txn := server.db.NewTransaction(true)
	defer txn.Discard()

	err = server.set(ctx, key, r, txn)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(err.Error()))
		return
	}

	err = server.commit(ctx, timestamp, key, r, txn)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(err.Error()))
		return
	}

	err = txn.Commit()
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(err.Error()))
		return
	}

	res.Header().Add("ETag", r.ETag())
	res.Header().Add("Link", makeSelfLink(r.URI()))
	res.WriteHeader(204)
}

func parseContent(format string, base string, body io.Reader) (doc interface{}, err error) {
	if format == offers[0] {
		opts := ld.NewJsonLdOptions(base)
		opts.Format = format
		doc, err = ld.NewJsonLdProcessor().FromRDF(body, opts)
	} else if format == offers[1] {
		err = json.NewDecoder(body).Decode(&doc)
	} else if format == offers[2] {
		var quads []*rdf.Quad
		err = json.NewDecoder(body).Decode(&quads)
		if err != nil {
			return
		}
		dataset := styx.ToRDFDataset(quads)
		opts := ld.NewJsonLdOptions(base)
		doc, err = ld.NewJsonLdApi().FromRDF(dataset, opts)
	}

	return
}
