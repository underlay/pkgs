package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	badger "github.com/dgraph-io/badger/v2"
	cid "github.com/ipfs/go-cid"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	multibase "github.com/multiformats/go-multibase"

	types "github.com/underlay/pkgs/types"
)

// Mkcol handles HTTP MKCOL requests
func (server *Server) Mkcol(ctx context.Context, res http.ResponseWriter, req *http.Request) error {
	pathname := req.URL.Path
	if pathname == "/" {
		res.WriteHeader(403)
		return nil
	} else if !pathRegex.MatchString(pathname) {
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
		parentResource := &types.Resource{}
		err := parentResource.Get(parentPath, txn)
		if err == badger.ErrKeyNotFound {
			// MKCOL actually requires 409 and not 404 here...
			res.WriteHeader(409)
			return nil
		} else if err != nil {
			res.WriteHeader(500)
			return err
		}

		parent := parentResource.GetPackage()
		if parent == nil {
			res.WriteHeader(409)
			return nil
		}

		for _, member := range parent.Member {
			if member == name || member == name+".nt" {
				res.WriteHeader(409)
				return nil
			}
		}

		parent.Member = append(parent.Member, name)
		p := types.NewPackage(ctx, pathname, server.resource+pathname)
		id, err := server.Normalize(ctx, pathname, p, false, nil)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		resource := &types.Resource{}
		resource.Resource = &types.Resource_Package{Package: p}
		err = resource.Set(pathname, txn)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		parentID, parentValue, err := parent.Paths()
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		value, err := cid.Cast(p.Value)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		err = server.percolate(
			ctx,
			parentPath,
			parentID,
			parentValue,
			parent,
			name,
			id,
			path.IpfsPath(value),
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
		res.Header().Add("ETag", s)
		res.Header().Add("Link", linkTypeDirectContainer)
		res.Header().Add("Link", fmt.Sprintf(`<#%s>; rel="self"`, p.Subject))

		res.WriteHeader(201)
		return nil
	})
}
