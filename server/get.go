package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	badger "github.com/dgraph-io/badger/v2"
	files "github.com/ipfs/go-ipfs-files"
	path "github.com/ipfs/interface-go-ipfs-core/path"

	types "github.com/underlay/pkgs/types"
)

// Get handles HTTP GET requests
func (server *Server) Get(ctx context.Context, res http.ResponseWriter, req *http.Request) error {
	accept := req.Header.Get("Accept")
	ifNoneMatch := req.Header.Get("If-None-Match")

	pathname := req.URL.Path

	if pathname != "/" && !pathRegex.MatchString(pathname) {
		res.WriteHeader(404)
		return nil
	}

	var resource types.Resource
	err := server.db.View(func(txn *badger.Txn) (err error) {
		resource, _, err = types.GetResource(pathname, txn)
		return
	})

	if err == badger.ErrKeyNotFound {
		res.WriteHeader(404)
		return nil
	} else if err != nil {
		res.WriteHeader(500)
		return err
	}

	res.Header().Add("Link", linkTypeResource)

	c, etag := resource.ETag()
	if ifNoneMatch == etag {
		res.WriteHeader(304)
		return nil
	}

	res.Header().Add("ETag", etag)

	node, err := server.fs.Get(ctx, path.IpfsPath(c))
	if err != nil {
		res.WriteHeader(502)
		return err
	}

	file := files.ToFile(node)

	// Okay now we have a Resource and we get to respond with its representation
	switch t := resource.(type) {
	case *types.Package:
		res.Header().Add("Link", linkTypeDirectContainer)
		res.Header().Add("Link", fmt.Sprintf(`<#%s>; rel="self"`, t.Subject))
		if accept == "application/ld+json" {
			res.Header().Add("Content-Type", "application/ld+json")
			doc, err := server.proc.FromRDF(file, server.opts)
			if err != nil {
				res.WriteHeader(500)
				return err
			}

			frame := map[string]interface{}{
				"@context": types.ContextURL,
				"@type":    types.PackageIri.Value,
			}

			framed, err := server.proc.Frame(doc, frame, server.opts)
			if err != nil {
				res.WriteHeader(500)
				return err
			}

			framed["@context"] = types.ContextURL
			_ = json.NewEncoder(res).Encode(framed)
		} else {
			res.Header().Add("Content-Type", "application/n-quads")
			_, _ = io.Copy(res, file)
		}
	case types.Message:
		res.Header().Add("Link", linkTypeNonRDFSource)
		if accept == "application/ld+json" {
			doc, err := server.proc.FromRDF(file, server.opts)
			if err != nil {
				res.WriteHeader(500)
				return err
			}

			res.Header().Add("Content-Type", accept)
			_ = json.NewEncoder(res).Encode(doc)
		} else {
			res.Header().Add("Content-Type", "application/n-quads")
			_, _ = io.Copy(res, file)
		}
	case *types.File:
		res.Header().Add("Link", linkTypeNonRDFSource)

		extent := strconv.FormatUint(t.Extent, 10)
		res.Header().Add("Content-Type", t.Format)
		res.Header().Add("Content-Length", extent)
		_, _ = io.Copy(res, file)
	}
	return nil
}
