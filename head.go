package main

import (
	"context"
	"net/http"

	badger "github.com/dgraph-io/badger/v2"
	content "github.com/joeltg/negotiate/content"
	types "github.com/underlay/pkgs/types"
)

// Head handles HTTP HEAD requests
func (server *Server) Head(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	txn := server.db.NewTransaction(false)
	defer txn.Discard()

	key := types.ParsePath(req.URL.Path)
	r, err := getResource(key, txn)
	if err == badger.ErrKeyNotFound {
		res.WriteHeader(404)
		return
	} else if err != nil {
		res.WriteHeader(500)
		return
	}

	res.Header().Add("ETag", r.ETag())
	res.Header().Add("Link", makeSelfLink(r.URI()))
	res.Header().Add("Link", types.LinkTypeResource)
	res.Header().Add("Link", types.MakeLinkType(r.Type()))
	switch r := r.(type) {
	case *types.Package:
		format := content.NegotiateContentType(req, append(offers, "text/html"), offers[0])
		res.Header().Add("Content-Type", format)
	case *types.Assertion:
		format := content.NegotiateContentType(req, offers, offers[0])
		res.Header().Add("Content-Type", format)
	case *types.File:
		res.Header().Add("Content-Type", r.Format)
	}

	res.WriteHeader(200)
}
