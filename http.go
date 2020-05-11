package main

import (
	"context"
	"net/http"
)

func makeSelfLink(id string) string { return "<" + id + `>; rel="self"` }

// ServeHTTP handles HTTP requests using the database and core API
func (server *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	if req.Method == "GET" {
		server.Get(ctx, res, req)
	} else if req.Method == "HEAD" {
		server.Head(ctx, res, req)
	} else if req.Method == "POST" {
		server.Post(ctx, res, req)
	} else if req.Method == "PUT" {
		server.Put(ctx, res, req)
	} else if req.Method == "DELETE" {
		server.Delete(ctx, res, req)
		// } else if req.Method == "PATCH" {
		//  server.Patch(ctx, res, req)
		// } else if req.Method == "COPY" {
		// } else if req.Method == "LOCK" {
	} else if req.Method == "MKCOL" {
		server.Mkcol(ctx, res, req)
		// } else if req.Method == "MOVE" {
		// } else if req.Method == "UNLOCK" {
	} else {
		res.WriteHeader(405)
	}

	return
}
