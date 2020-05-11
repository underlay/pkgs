package main

import (
	"context"
	"net/http"
	"strings"

	badger "github.com/dgraph-io/badger/v2"
	types "github.com/underlay/pkgs/types"
)

// Mkcol handles HTTP MKCOL requests
func (server *Server) Mkcol(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	key := types.ParsePath(req.URL.Path)
	if len(key) == 0 {
		res.WriteHeader(409)
		return
	}

	txn := server.db.NewTransaction(true)
	defer txn.Discard()

	_, err := txn.Get(getKey(key))
	if err == badger.ErrKeyNotFound {
	} else if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(err.Error()))
		return
	} else {
		res.WriteHeader(409)
		return
	}

	resource := server.resource + "/" + strings.Join(key, "/")
	name := key[len(key)-1]
	pkg := types.NewPackage(resource, name)

	_, err = server.normalize(ctx, pkg)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(err.Error()))
		return
	}

	err = server.set(ctx, key, pkg, txn)
	if err == badger.ErrKeyNotFound {
		res.WriteHeader(404)
		return
	} else if err == ErrParentNotPackage {
		res.WriteHeader(409)
		return
	} else if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(err.Error()))
		return
	}

	err = server.commit(ctx, pkg.Created, key, pkg, txn)
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

	res.Header().Add("ETag", pkg.ETag())
	res.Header().Add("Link", makeSelfLink(pkg.URI()))
	res.Header().Add("Link", types.MakeLinkType(types.LDPResource))
	res.Header().Add("Link", types.MakeLinkType(types.LDPDirectContainer))

	res.WriteHeader(201)
}
