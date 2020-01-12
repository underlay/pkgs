package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	multibase "github.com/multiformats/go-multibase"

	types "github.com/underlay/pkgs/types"
)

// Mkcol handles HTTP MKCOL requests
func (server *Server) Mkcol(ctx context.Context, res http.ResponseWriter, req *http.Request) error {
	pathname := req.URL.Path
	if pathname == "/" {
		res.WriteHeader(403)
		return nil
	} else if !PathRegex.MatchString(pathname) {
		res.WriteHeader(404)
		return nil
	}

	tail := strings.LastIndex(pathname, "/")
	name := pathname[tail+1:]
	var parentPath string
	if tail > 0 {
		parentPath = pathname[:tail]
	} else {
		parentPath = "/"
	}

	return server.db.Update(func(txn *badger.Txn) error {
		parent, err := types.GetPackage(parentPath, txn)
		if err == badger.ErrKeyNotFound || err == types.ErrNotPackage {
			// MKCOL actually requires 409 and not 404 here...
			res.WriteHeader(409)
			return nil
		} else if err != nil {
			res.WriteHeader(500)
			return err
		}

		for _, member := range parent.Member {
			if member == name || member == name+".nt" {
				res.WriteHeader(409)
				return nil
			}
		}

		t := time.Now()
		parent.Member = append(parent.Member, name)
		p := types.NewPackage(ctx, t, pathname, server.resource+pathname)
		id, err := server.Normalize(ctx, pathname, p, false, nil)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		err = types.SetResource(p, pathname, txn)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		parentID, parentValue, err := parent.Paths()
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		value, err := server.object.AddLink(ctx, parentValue, name, id)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		err = server.percolate(
			ctx,
			t,
			parentPath,
			parent,
			parentID, parentValue,
			value,
			txn,
		)

		if err != nil {
			res.WriteHeader(500)
			return err
		}

		s, err := id.Cid().StringOfBase(multibase.Base32)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		res.Header().Add("Access-Control-Allow-Origin", "http://localhost:8000")
		res.Header().Add("Access-Control-Allow-Methods", "GET, HEAD, POST, PATCH, DELETE")
		res.Header().Add("Access-Control-Allow-Headers", "Accept, Link, If-Match")
		res.Header().Add("Access-Control-Expose-Headers", "Link, ETag")

		res.Header().Add("ETag", fmt.Sprintf("\"%s\"", s))
		res.Header().Add("Link", linkTypeDirectContainer)
		res.Header().Add("Link", fmt.Sprintf(`<#%s>; rel="self"`, p.Subject))
		res.WriteHeader(201) // ???

		return nil
	})
}
