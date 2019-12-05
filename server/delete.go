package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	badger "github.com/dgraph-io/badger/v2"
	cid "github.com/ipfs/go-cid"
	core "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"

	types "github.com/underlay/pkgs/types"
)

// Delete handles HTTP DELETE requests
func Delete(ctx context.Context, res http.ResponseWriter, req *http.Request, db *badger.DB, api core.CoreAPI) error {
	fs, pin, object := api.Unixfs(), api.Pin(), api.Object()

	pathname := req.URL.Path
	if pathname == "/" {
		res.WriteHeader(403)
		return nil
	} else if !pathRegex.MatchString(pathname) {
		res.WriteHeader(404)
		return nil
	}

	ifMatch := req.Header.Get("If-Match")
	if ifMatch == "" {
		res.WriteHeader(416)
		return nil
	}

	return db.Update(func(txn *badger.Txn) error {
		r := &types.Resource{}
		err := r.Get(pathname, txn)
		if err == badger.ErrKeyNotFound {
			res.WriteHeader(404)
			return nil
		} else if err != nil {
			res.WriteHeader(500)
			return err
		}

		etag := r.ETag()
		_, s, err := types.GetCid(etag)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		if s != ifMatch {
			res.WriteHeader(416)
			return nil
		}

		tail := strings.LastIndex(pathname, "/")
		var parentPath, name string
		if tail > 0 {
			parentPath = pathname[:tail]
			name = pathname[tail+1:]
		} else {
			parentPath = "/"
			name = pathname[1:]
		}

		parentResource := &types.Resource{}
		err = parentResource.Get(parentPath, txn)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		parent := parentResource.GetPackage()
		if parent != nil {
			res.WriteHeader(500)
			return nil
		}

		// Okay now we have the containing package.
		// Let's remove the thing
		for i, member := range parent.Member {
			if member == name {
				parent.Member = append(parent.Member[:i], parent.Member[i+1:]...)
				break
			}
		}

		// Now update the value by removing the object link
		c, err := cid.Cast(parent.Value)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		id, err := cid.Cast(parent.Id)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		parentValue := path.IpfsPath(c)
		parentValue, err = object.RmLink(ctx, parentValue, name)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		// Also remove the direct object for packages
		if r.GetPackage() != nil {
			parentValue, err = object.RmLink(ctx, parentValue, fmt.Sprintf("%s.nt", name))
			if err != nil {
				res.WriteHeader(500)
				return err
			}
		}

		err = percolate(
			ctx,
			parentPath,
			path.IpfsPath(id),
			parentValue,
			parent,
			name, nil,
			txn, fs, object, pin,
		)

		if err != nil {
			res.WriteHeader(500)
			return err
		}

		res.Write(nil)
		return nil
	})

}
