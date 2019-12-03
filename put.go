package main

import (
	"net/http"

	badger "github.com/dgraph-io/badger/v2"
	ipfs "github.com/ipfs/go-ipfs-api"
)

func Put(res http.ResponseWriter, req *http.Request, pkg *Package, sh *ipfs.Shell, db *badger.DB) {
	contentType := req.Header.Get("Content-Type")
	if contentType != "application/n-quads" && contentType != "application/ld+json" {
		res.WriteHeader(415)
		return
	}

	if req.URL.Path == "/" {

	}
}
