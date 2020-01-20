package server

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v2"

	query "github.com/underlay/pkgs/query"
	types "github.com/underlay/pkgs/types"
)

// Delete handles HTTP DELETE requests
func (server *Server) Delete(ctx context.Context, res http.ResponseWriter, req *http.Request) error {
	pathname := req.URL.Path
	if pathname == "/" {
		res.WriteHeader(403)
		return nil
	} else if !PathRegex.MatchString(pathname) {
		res.WriteHeader(404)
		return nil
	}

	// Acquire lock
	var lock *sync.Mutex
	var has bool
	if lock, has = server.locks[pathname]; !has {
		lock = &sync.Mutex{}
		server.locks[pathname] = lock
	}
	lock.Lock()
	defer lock.Unlock()
	defer delete(server.locks, pathname)

	ifMatch := req.Header.Get("If-Match")
	if !etagRegex.MatchString(ifMatch) {
		res.WriteHeader(412)
		return nil
	}

	match := etagRegex.FindStringSubmatch(ifMatch)[1]

	return server.db.Update(func(txn *badger.Txn) error {
		r, err := types.GetResource(pathname, txn)
		if err == badger.ErrKeyNotFound {
			res.WriteHeader(404)
			return nil
		} else if err != nil {
			res.WriteHeader(500)
			return err
		}

		_, etag := r.ETag()
		if etag != match {
			res.WriteHeader(412)
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

		parent, err := types.GetPackage(parentPath, txn)
		if err != nil {
			res.WriteHeader(500)
			return err
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
		// c, err := cid.Cast(parent.Value)
		// if err != nil {
		// 	res.WriteHeader(500)
		// 	return err
		// }

		// value := path.IpfsPath(c)

		oldID, oldValue, err := parent.Paths()
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		value, err := server.object.RmLink(ctx, oldValue, name)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		_ = txn.Delete([]byte(pathname))

		// Also remove the direct object for packages
		if r.Type() == query.PackageType {
			_ = txn.Delete([]byte(pathname + ".nt"))
			value, err = server.object.RmLink(ctx, value, name+".nt")
			if err != nil {
				res.WriteHeader(500)
				return err
			}

			// Also for packages we have sub-*keys* that we need to clean up.
			// We could do this with `for _, member := range p.Member`, but
			// that would require unmarshalling each item to see it *it* was
			// a package, and recursing on that. Instead, just create an
			// iterator with the current path as the prefix.
			prefix := []byte(pathname + "/")
			iter := txn.NewIterator(badger.IteratorOptions{
				PrefetchValues: false,
				Prefix:         prefix,
			})

			for iter.Seek(prefix); iter.Valid(); iter.Next() {
				// Don't actually return from here because we
				// want to clean up everything if at all possible.
				err = txn.Delete(iter.Item().Key())
				if err != nil {
					log.Println(pathname, err)
				}
			}
		}

		err = server.percolate(
			ctx,
			time.Now(),
			parentPath,
			parent,
			oldID, oldValue,
			value, // parentValue already has the appropriate link removed
			txn,
		)

		if err != nil {
			res.WriteHeader(500)
			return err
		}

		res.WriteHeader(204)
		return nil
	})

}
