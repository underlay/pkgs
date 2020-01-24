package http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	badger "github.com/dgraph-io/badger/v2"

	query "github.com/underlay/pkgs/query"
	types "github.com/underlay/pkgs/types"
)

// Head handles HTTP HEAD requests
func (server *Server) Head(ctx context.Context, res http.ResponseWriter, req *http.Request) error {
	pathname := req.URL.Path
	if pathname != "/" && !PathRegex.MatchString(pathname) {
		res.WriteHeader(404)
		return nil
	}

	ifNoneMatch := req.Header.Get("If-None-Match")

	var resource query.Resource
	err := server.db.View(func(txn *badger.Txn) (err error) {
		resource, err = types.GetResource(pathname, txn)
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

	res.Header().Add("ETag", fmt.Sprintf("\"%s\"", etag))
	res.WriteHeader(204)
	return nil
}
