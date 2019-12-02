package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	badger "github.com/dgraph-io/badger/v2"
	proto "github.com/gogo/protobuf/proto"
	ipfs "github.com/ipfs/go-ipfs-api"
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

/*
Okay
there's one lockfile and it's package.jsonld
It has an explicit content URI for a subject though
*/

type Response interface {
	NQuads() (io.Reader, string)
	JSONLD() (io.Reader, string)
	HTML() (io.Reader, string)
}

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

	return

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == "GET" {
			accept := req.Header.Get("Accept")
			ifNoneMatch := req.Header.Get("If-None-Match")
			if req.URL.Path == "/" {
				if ifNoneMatch == pkg.Id {
					res.WriteHeader(304)
					return
				}

				res.Header().Add("ETag", pkg.Id)
				if accept == "application/ld+json" {
					res.Header().Add("Content-Type", accept)
					res.WriteHeader(200)
					encoder := json.NewEncoder(res)
					_ = db.View(func(txn *badger.Txn) error {
						doc, err := pkg.JSON("/", txn)
						if err != nil {
							return err
						}
						return encoder.Encode(doc)
					})
				} else if accept == "application/n-quads" {
					reader, err := sh.Cat(pkg.Id)
					if err != nil {
						res.WriteHeader(502)
						return
					}

					res.Header().Add("Content-Type", accept)
					res.WriteHeader(200)
					_, _ = io.Copy(res, reader)
				} else {
					res.WriteHeader(406)
				}
				return
			}

			match := pathRegex.FindStringSubmatch(req.URL.Path)
			if match == nil {
				res.WriteHeader(404)
				return
			}

			path = match[0]

			last := strings.LastIndex(path, "/")

			// terms := strings.Split(path[1:], "/")

			// container := strings.Join(terms[:len(terms)-1], "/")

			stats, err := sh.ObjectStat(pkg.Value + path[:last])
			if err != nil {
				log.Println(err)
				res.WriteHeader(502)
				return
			}

			if stats.NumLinks > 0 || stats.Hash == emptyDirectory {
				// Directory!

			} else {
				// File!

			}

			// if stats.NumLinks

			// if req.URL.Path[1] == '/' {
			// 	name := fmt.Sprintf("/ipfs%s", req.URL.Path)
			// 	// sh.ObjectStat(key string)
			// 	// sh.List(path string)
			// 	if resource, has := pkg.Members[name]; has {

			// 	} else {
			// 		res.WriteHeader(404)
			// 	}
			// }

		} else if req.Method == "PUT" {
			// contentType := req.Header.Get("Content-Type")

			if req.URL.Path == "/" {

			}
		} else if req.Method == "HEAD" {
			if req.URL.Path == "/" {

			}
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
