package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	badger "github.com/dgraph-io/badger/v2"
	core "github.com/ipfs/interface-go-ipfs-core"

	types "github.com/underlay/pkgs/types"
)

// Head handles HTTP HEAD requests
func Head(ctx context.Context, res http.ResponseWriter, req *http.Request, db *badger.DB, api core.CoreAPI) error {
	path := req.URL.Path
	if path != "/" && !pathRegex.MatchString(path) {
		res.WriteHeader(404)
		return nil
	}

	accept := req.Header.Get("Accept")
	ifNoneMatch := req.Header.Get("If-None-Match")

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

	// Now that we know there's a resource here, we add the first Link header
	res.Header().Add("Link", linkTypeResource)

	etag := resource.ETag()
	_, s, err := types.GetCid(etag)
	if err != nil {
		res.WriteHeader(500)
		return err
	}

	if ifNoneMatch == s {
		res.WriteHeader(304)
		return nil
	}

	res.Header().Add("ETag", s)

	p, m, f := resource.GetPackage(), resource.GetMessage(), resource.GetFile()
	if f != nil {
		res.Header().Add("Link", linkTypeNonRDFSource)
		extent := strconv.FormatUint(f.Extent, 10)
		res.Header().Add("Content-Type", f.Format)
		res.Header().Add("Content-Length", extent)
	} else if m != nil {
		if accept == "application/n-quads" || accept == "application/ld+json" {
			res.Header().Add("Link", linkTypeRDFSource)
			res.Header().Add("Content-Type", accept)
		} else {
			res.WriteHeader(406)
			return nil
		}
	} else if p != nil {
		if accept == "application/n-quads" || accept == "application/ld+json" {
			res.Header().Add("Link", fmt.Sprintf(`<#%s>; rel="self"`, p.Subject))
			res.Header().Add("Content-Type", accept)
		} else {
			res.WriteHeader(406)
		}
	}

	res.Write(nil)
	return nil
}
