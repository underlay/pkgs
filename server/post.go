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

	// Should we require If-Match? I bet we should.
	ifMatch := req.Header.Get("If-Match")
	if !etagRegex.MatchString(ifMatch) {
		res.WriteHeader(412)
		return nil
	}

	match := etagRegex.FindStringSubmatch(ifMatch)[1]

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

	_, etag := resource.ETag()
	if err != nil {
		res.WriteHeader(500)
		return err
	}

	if etag != match {
		res.WriteHeader(412)
		return nil
	}

	switch resource.(type) {
	case *types.Package:
	default:
		res.WriteHeader(405)
		return nil
	}

	res.Header().Add("Access-Control-Allow-Origin", "http://localhost:8000")
	res.Header().Add("Access-Control-Allow-Methods", "GET, HEAD, POST, DELETE")
	res.Header().Add("Access-Control-Allow-Headers", "Accept, Link, If-Match")
	res.WriteHeader(501)

	return nil
}
