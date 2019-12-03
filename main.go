package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	badger "github.com/dgraph-io/badger/v2"
	proto "github.com/gogo/protobuf/proto"
	cid "github.com/ipfs/go-cid"
	ipfs "github.com/ipfs/go-ipfs-api"
	multibase "github.com/multiformats/go-multibase"
	ld "github.com/piprate/json-gold/ld"
)

const defaultHost = "localhost:5001"
const defaultOrigin = "dweb:/ipns"

var host = os.Getenv("IPFS_HOST")
var port = os.Getenv("PKGS_PORT")
var path = os.Getenv("PKGS_PATH")
var name = os.Getenv("PKGS_NAME")
var origin = os.Getenv("PKGS_ORIGIN")

var shError = "IPFS Daemon not running"

var pathRegex = regexp.MustCompile("^(/[a-zA-Z0-9-\\.]+)+$")

var proc = ld.NewJsonLdProcessor()

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

	r := &Resource{}
	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(index))
		if err == badger.ErrKeyNotFound {
			pkg, err := NewPackage(index, resource, sh)
			if err != nil {
				return err
			}

			r.Resource = &Resource_Package{pkg}
			val, err := proto.Marshal(r)
			if err != nil {
				return err
			}

			return txn.Set([]byte(index), val)
		} else if err != nil {
			return err
		} else {
			return item.Value(func(val []byte) error {
				return proto.Unmarshal(val, r)
			})
		}
	})

	if err != nil {
		log.Fatal(err)
	}

	pkg := r.GetPackage()

	log.Println("pkg", pkg)

	c, err := cid.Parse(pkg.Id)
	if err != nil {
		log.Fatal(err)
	}

	root, err := c.StringOfBase(multibase.Base32)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == "GET" {
			Get(res, req, root, pkg, sh, db)
		} else if req.Method == "PUT" {
			Put(res, req, root, pkg, sh, db)
		} else if req.Method == "HEAD" {
			Head(res, req, root, pkg, sh, db)
		} else if req.Method == "DELETE" {
			if req.URL.Path == "/" {

			}
		} else if req.Method == "OPTIONS" {
			if req.URL.Path == "/" {

			}
		}
		return
	})

	log.Printf("http://localhost:%s\n", port)

	http.ListenAndServe(":"+port, nil)
}
