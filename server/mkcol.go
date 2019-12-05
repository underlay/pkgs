package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	badger "github.com/dgraph-io/badger/v2"
	core "github.com/ipfs/interface-go-ipfs-core"
	options "github.com/ipfs/interface-go-ipfs-core/options"
	path "github.com/ipfs/interface-go-ipfs-core/path"

	types "github.com/underlay/pkgs/types"
)

// Mkcol handles HTTP MKCOL requests
func Mkcol(ctx context.Context, res http.ResponseWriter, req *http.Request, db *badger.DB, api core.CoreAPI) error {
	fs, pin := api.Unixfs(), api.Pin()
	pathname := req.URL.Path
	if pathname == "/" {
		res.WriteHeader(403)
		return nil
	} else if !pathRegex.MatchString(pathname) {
		res.WriteHeader(404)
		return nil
	}

	accept := req.Header.Get("Content-Type")
	if accept != "application/ld+json" && accept != "application/n-quads" {
		res.WriteHeader(415)
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

	return db.Update(func(txn *badger.Txn) error {
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
			if member == name {
				res.WriteHeader(409)
				return nil
			}
		}

		parent.Member = append(parent.Member, name)
		c, p, err := types.NewPackage(ctx, pathname, fmt.Sprintf(""), fs)
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

		leaf := path.IpfsPath(c)

		err = percolate(
			ctx,
			parentPath,
			parentID,
			parentValue,
			parent,
			name,
			leaf,
			txn, api,
		)

		if err != nil {
			res.WriteHeader(500)
			return err
		}

		err = pin.Rm(ctx, leaf, options.Pin.RmRecursive(true))
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		return nil
	})
}
