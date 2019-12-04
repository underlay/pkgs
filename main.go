package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	badger "github.com/dgraph-io/badger/v2"
	proto "github.com/gogo/protobuf/proto"
	cid "github.com/ipfs/go-cid"
	ipfs "github.com/ipfs/go-ipfs-api"
	multibase "github.com/multiformats/go-multibase"

	server "github.com/underlay/pkgs/server"
	types "github.com/underlay/pkgs/types"
)

const defaultHost = "localhost:5001"
const defaultOrigin = "dweb:/ipns"

var host = os.Getenv("IPFS_HOST")
var port = os.Getenv("PKGS_PORT")
var path = os.Getenv("PKGS_PATH")
var name = os.Getenv("PKGS_NAME")
var origin = os.Getenv("PKGS_ORIGIN")

var shError = "IPFS Daemon not running"

func main() {
	if host == "" {
		host = defaultHost
	}

	sh := ipfs.NewShell(host)

	if !sh.IsUp() {
		log.Fatal(shError)
	}

	peerID, err := sh.ID()
	if err != nil {
		log.Fatal(err)
	}

	if origin == "" {
		origin = defaultOrigin
	}

	if name == "" {
		name = peerID.ID
	}

	resource := fmt.Sprintf("%s/%s", origin, name)

	if path == "" {
		path = "/tmp/pkgs"
	}

	opts := badger.DefaultOptions(path)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal(err)
	}

	index := "/"

	var pkg *types.Package
	var root string
	err = db.Update(func(txn *badger.Txn) error {
		r := &types.Resource{}
		item, err := txn.Get([]byte(index))
		if err == badger.ErrKeyNotFound {
			root, pkg, err = types.NewPackage(index, resource, sh)
			if err != nil {
				return err
			}

			r.Resource = &types.Resource_Package{pkg}
			val, err := proto.Marshal(r)
			if err != nil {
				return err
			}

			return txn.Set([]byte(index), val)
		} else if err != nil {
			return err
		} else {
			err = item.Value(func(val []byte) error {
				return proto.Unmarshal(val, r)
			})
			if err != nil {
				return err
			}

			pkg := r.GetPackage()
			if pkg == nil {
				return fmt.Errorf("Invalid index: %v", r)
			}

			c, err := cid.Parse(pkg.Id)
			if err != nil {
				return err
			}

			root, err = c.StringOfBase(multibase.Base32)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Root:", root)
	log.Println("Package:", pkg)

	http.HandleFunc("/", server.Handler)

	log.Printf("http://localhost:%s\n", port)

	http.ListenAndServe(":"+port, nil)
}
