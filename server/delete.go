package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	cid "github.com/ipfs/go-cid"
	core "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"

	types "github.com/underlay/pkgs/types"
)

// Delete handles HTTP DELETE requests
func Delete(ctx context.Context, res http.ResponseWriter, req *http.Request, db *badger.DB, api core.CoreAPI) error {
	object := api.Object()

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

		next, err := object.RmLink(ctx, path.IpfsPath(c), name)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		// Also remove the direct object for packages
		if r.GetPackage() != nil {
			next, err = object.RmLink(ctx, next, fmt.Sprintf("%s.nt", name))
			if err != nil {
				res.WriteHeader(500)
				return err
			}
		}

		stat, err := object.Stat(ctx, next)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		parent.Extent = uint64(stat.CumulativeSize)
		parent.Value = next.Cid().Bytes()
		parent.Modified = time.Now().Format(time.RFC3339)

		res.WriteHeader(501)

		res.Write(nil)
		return nil
	})

}
