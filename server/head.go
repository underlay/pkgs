package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	badger "github.com/dgraph-io/badger/v2"

	types "github.com/underlay/pkgs/types"
)

// Head handles HTTP HEAD requests
func (server *Server) Head(ctx context.Context, res http.ResponseWriter, req *http.Request) error {
	pathname := req.URL.Path
	if pathname != "/" && !pathRegex.MatchString(pathname) {
		res.WriteHeader(404)
		return nil
	}

	ifNoneMatch := req.Header.Get("If-None-Match")

	resource := &types.Resource{}

	err := server.db.View(func(txn *badger.Txn) error {
		return resource.Get(pathname, txn)
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
		res.Header().Add("Link", linkTypeRDFSource)
		res.Header().Add("Content-Type", "application/n-quads")
	} else if p != nil {
		res.Header().Add("Link", linkTypeDirectContainer)
		res.Header().Add("Link", fmt.Sprintf(`<#%s>; rel="self"`, p.Subject))
		res.Header().Add("Content-Type", "application/n-quads")
	}

	res.Write(nil)
	return nil
}
