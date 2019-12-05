package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	badger "github.com/dgraph-io/badger/v2"
	core "github.com/ipfs/interface-go-ipfs-core"

	types "github.com/underlay/pkgs/types"
)

// Get handles HTTP GET requests
func Get(ctx context.Context, res http.ResponseWriter, req *http.Request, db *badger.DB, api core.CoreAPI) error {
	fs := api.Unixfs()
	accept := req.Header.Get("Accept")
	ifNoneMatch := req.Header.Get("If-None-Match")

	pathname := req.URL.Path

	if pathname != "/" && !pathRegex.MatchString(pathname) {
		res.WriteHeader(404)
		return nil
	}

	resource := &types.Resource{}
	err := db.View(func(txn *badger.Txn) error {
		return resource.Get(pathname, txn)
	})

	if err == badger.ErrKeyNotFound {
		res.WriteHeader(404)
		return nil
	} else if err != nil {
		res.WriteHeader(500)
		return err
	}

	res.Header().Add("Link", linkTypeResource)

	etag := resource.ETag()
	c, s, err := types.GetCid(etag)
	if err != nil {
		res.WriteHeader(500)
		return err
	}

	if ifNoneMatch == s {
		res.WriteHeader(304)
		return nil
	}

	res.Header().Add("ETag", s)

	// Okay now we have a Resource and we get to respond with its representation
	p, m, f := resource.GetPackage(), resource.GetMessage(), resource.GetFile()
	if f != nil {
		res.Header().Add("Link", linkTypeNonRDFSource)
		file, err := types.GetFile(ctx, c, fs)
		if err != nil {
			res.WriteHeader(502)
			return err
		}

		extent := strconv.FormatUint(f.Extent, 10)
		res.Header().Add("Content-Type", f.Format)
		res.Header().Add("Content-Length", extent)
		_, _ = io.Copy(res, file)
	} else if m != nil {
		res.Header().Add("Link", linkTypeNonRDFSource)
		if accept == "application/n-quads" {
			file, err := types.GetFile(ctx, c, fs)
			if err != nil {
				res.WriteHeader(502)
				return err
			}

			res.Header().Add("Content-Type", accept)
			_, _ = io.Copy(res, file)
		} else if accept == "application/ld+json" {
			file, err := types.GetFile(ctx, c, fs)
			if err != nil {
				res.WriteHeader(502)
				return err
			}

			doc, err := types.Proc.FromRDF(file, types.Opts)
			if err != nil {
				res.WriteHeader(500)
				return err
			}

			res.Header().Add("Content-Type", accept)
			_ = json.NewEncoder(res).Encode(doc)
		} else {
			res.WriteHeader(406)
			return nil
		}
	} else if p != nil {
		res.Header().Add("Link", linkTypeDirectContainer)
		if accept != "application/n-quads" && accept != "application/ld+json" {
			res.WriteHeader(406)
			return nil
		}

		res.Header().Add("Link", fmt.Sprintf(`<#%s>; rel="self"`, p.Subject))
		res.Header().Add("Content-Type", accept)

		if accept == "application/n-quads" {
			file, err := types.GetFile(ctx, c, fs)
			if err != nil {
				res.WriteHeader(502)
				return err
			}
			_, _ = io.Copy(res, file)
		} else if accept == "application/ld+json" {
			var doc map[string]interface{}
			err = db.View(func(txn *badger.Txn) (err error) {
				doc, err = p.JSON(pathname, txn)
				return
			})

			if err != nil {
				res.WriteHeader(500)
				return err
			}

			_ = json.NewEncoder(res).Encode(doc)
		}
	}
	return nil
}
