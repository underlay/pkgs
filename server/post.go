package server

import (
	"context"
	"net/http"

	badger "github.com/dgraph-io/badger/v2"
	core "github.com/ipfs/interface-go-ipfs-core"

	types "github.com/underlay/pkgs/types"
)

// Post handles HTTP POST requests
func Post(ctx context.Context, res http.ResponseWriter, req *http.Request, db *badger.DB, api core.CoreAPI) error {
	path := req.URL.Path
	if path == "/" {
		res.WriteHeader(403)
		return nil
	} else if !pathRegex.MatchString(path) {
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
