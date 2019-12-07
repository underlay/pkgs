package server

import (
	"context"
	"net/http"

	badger "github.com/dgraph-io/badger/v2"

	types "github.com/underlay/pkgs/types"
)

// Post handles HTTP POST requests
func (server *Server) Post(ctx context.Context, res http.ResponseWriter, req *http.Request) error {
	pathname := req.URL.Path
	if pathname == "/" {
		res.WriteHeader(403)
		return nil
	} else if !pathRegex.MatchString(pathname) {
		res.WriteHeader(404)
		return nil
	}

	ifMatch := req.Header.Get("If-Match")
	// Should we require ifMatch? I bet we should.
	if ifMatch == "" {
		res.WriteHeader(416)
		return nil
	}

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

	etag := resource.ETag()
	_, s, err := types.GetCid(etag)
	if err != nil {
		res.WriteHeader(500)
		return err
	}

	if s != ifMatch {
		res.WriteHeader(416)
		return nil
	}

	return nil
}
