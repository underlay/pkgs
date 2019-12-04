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
	ld "github.com/piprate/json-gold/ld"

	types "github.com/underlay/pkgs/types"
)

func Get(ctx context.Context, res http.ResponseWriter, req *http.Request, db *badger.DB, api core.CoreAPI) error {
	fs := api.Unixfs()
	accept := req.Header.Get("Accept")
	ifNoneMatch := req.Header.Get("If-None-Match")

	path := req.URL.Path

	if path != "/" && !pathRegex.MatchString(path) {
		res.WriteHeader(404)
		return nil
	}

	resource := &types.Resource{}
	err := db.View(func(txn *badger.Txn) error {
		return resource.Get(path, txn)
	})

	if err == badger.ErrKeyNotFound {
		res.WriteHeader(404)
		return nil
	} else if err != nil {
		res.WriteHeader(500)
		return err
	}

	res.Header().Add("Link", linkTypeResource)

	// Okay now we have a Resource and we get to respond with its representation
	p, m, f := resource.GetPackage(), resource.GetMessage(), resource.GetFile()
	if f != nil {
		c, s, err := types.GetCid(f.Value)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		res.Header().Add("Link", linkTypeNonRDFSource)

		if ifNoneMatch == s {
			res.WriteHeader(304)
			return nil
		}

		file, err := types.GetFile(ctx, c, fs)
		if err != nil {
			res.WriteHeader(502)
			return err
		}

		extent := strconv.FormatUint(f.Extent, 10)
		res.Header().Add("ETag", s)
		res.Header().Add("Content-Type", f.Format)
		res.Header().Add("Content-Length", extent)
		_, _ = io.Copy(res, file)
	} else if m != nil {
		c, s, err := types.GetCid(m)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		res.Header().Add("Link", linkTypeNonRDFSource)
		if ifNoneMatch == s {
			res.WriteHeader(304)
			return nil
		}

		res.Header().Add("ETag", s)

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

			opts := ld.NewJsonLdOptions("")
			opts.Format = "application/n-quads"

			doc, err := proc.FromRDF(file, opts)
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
		c, s, err := types.GetCid(p.Id)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		res.Header().Add("Link", linkTypeDirectContainer)

		if ifNoneMatch == s {
			res.WriteHeader(304)
			return nil
		}

		res.Header().Add("ETag", s)

		if accept != "application/n-quads" && accept != "application/ld+json" {
			res.WriteHeader(406)
			return nil
		}

		res.Header().Add("Link", fmt.Sprintf(`<#%s>; rel="self"`, p.Subject))
		res.Header().Add("Content-Type", accept)

		file, err := types.GetFile(ctx, c, fs)
		if err != nil {
			res.WriteHeader(502)
			return err
		}

		if accept == "application/n-quads" {
			_, _ = io.Copy(res, file)
		} else if accept == "application/ld+json" {
			opts := ld.NewJsonLdOptions("")
			opts.Format = "application/n-quads"

			doc, err := proc.FromRDF(file, opts)
			if err != nil {
				res.WriteHeader(500)
				return err
			}

			_ = json.NewEncoder(res).Encode(doc)
		}
	}
	return nil
}
