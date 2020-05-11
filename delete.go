package main

import (
	"context"
	"net/http"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	rpc "github.com/underlay/pkgs/rpc"
	types "github.com/underlay/pkgs/types"
)

// Delete a resource
func (server *Server) Delete(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	key := types.ParsePath(req.URL.Path)
	txn := server.db.NewTransaction(true)
	defer txn.Discard()

	r, err := getResource(key, txn)
	if err == badger.ErrKeyNotFound {
		res.WriteHeader(404)
		return
	} else if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(err.Error()))
		return
	}

	if pkg, is := r.(*types.Package); is {
		err = deleteChildren(key, pkg, txn)
		if err != nil {
			return
		}
	}

	rpc.Delete(key, r)
	err = txn.Delete(getKey(key))
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(err.Error()))
		return
	}

	timestamp := time.Now().Format(time.RFC3339)
	err = server.commit(ctx, timestamp, key, nil, txn)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(err.Error()))
		return
	}

	err = txn.Commit()
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(204)
}

func deleteChildren(
	key []string, pkg *types.Package,
	txn *badger.Txn,
) error {
	for _, p := range pkg.Members.Packages {
		childKey := append(key, p.Name())
		childPkg, err := getPackage(childKey, txn)
		if err != nil {
			return err
		}

		err = deleteChildren(childKey, childPkg, txn)
		if err != nil {
			return err
		}

		rpc.Delete(childKey, childPkg)
		err = txn.Delete(getKey(childKey))
		if err != nil {
			return err
		}
	}

	for _, a := range pkg.Members.Assertions {
		childKey := append(key, a.Name())
		rpc.Delete(childKey, a)
		err := txn.Delete(getKey(childKey))
		if err != nil {
			return err
		}
	}

	for _, f := range pkg.Members.Files {
		childKey := append(key, f.Name())
		rpc.Delete(childKey, f)
		err := txn.Delete(getKey(childKey))
		if err != nil {
			return err
		}
	}

	return nil
}
