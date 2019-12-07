package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	badger "github.com/dgraph-io/badger/v2"

	types "github.com/underlay/pkgs/types"
)

// Delete handles HTTP DELETE requests
func (server *Server) Delete(ctx context.Context, res http.ResponseWriter, req *http.Request) error {
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

	return server.db.Update(func(txn *badger.Txn) error {
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
		parentID, parentValue, err := parent.Paths()
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		parentValue, err = server.object.RmLink(ctx, parentValue, name)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		txn.Delete([]byte(pathname))

		// Also remove the direct object for packages
		if p := r.GetPackage(); p != nil {
			txn.Delete([]byte(fmt.Sprintf("%s.nt", pathname)))
			parentValue, err = server.object.RmLink(ctx, parentValue, fmt.Sprintf("%s.nt", name))
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
			parentPath,
			parentID,
			parentValue,
			parent,
			name, nil,
			txn,
		)

		if err != nil {
			res.WriteHeader(500)
			return err
		}

		res.Write(nil)
		return nil
	})

}
