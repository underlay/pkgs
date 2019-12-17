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

	// Now that we know there's a resource here, we add the first Link header
	res.Header().Add("Link", linkTypeResource)

	_, etag := resource.ETag()
	if err != nil {
		res.WriteHeader(500)
		return err
	}

	if ifNoneMatch == etag {
		res.WriteHeader(304)
		return nil
	}

	res.Header().Add("ETag", etag)

	switch t := resource.(type) {
	case *types.Package:
		res.Header().Add("Link", linkTypeDirectContainer)
		res.Header().Add("Link", fmt.Sprintf(`<#%s>; rel="self"`, t.Subject))
		res.Header().Add("Content-Type", "application/n-quads")
	case types.Message:
		res.Header().Add("Link", linkTypeRDFSource)
		res.Header().Add("Content-Type", "application/n-quads")
	case *types.File:
		res.Header().Add("Link", linkTypeNonRDFSource)
		extent := strconv.FormatUint(t.Extent, 10)
		res.Header().Add("Content-Type", t.Format)
		res.Header().Add("Content-Length", extent)
	}

	res.Write(nil)
	return nil
}
