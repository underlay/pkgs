package main

import (
	"net/http"
	"strings"

	badger "github.com/dgraph-io/badger/v2"
	proto "github.com/gogo/protobuf/proto"
	ipfs "github.com/ipfs/go-ipfs-api"
)

func Put(res http.ResponseWriter, req *http.Request, root string, pkg *Package, sh *ipfs.Shell, db *badger.DB) {
	contentType := req.Header.Get("Content-Type")
	links := req.Header["Link"]
	var linkType string
	var isTypeResource bool
	for _, link := range links {
		isTypeResource = isTypeResource || link == linkTypeResource
		if _, has := linkTypes[link]; has {
			if linkType == "" {
				linkType = link
			} else {
				// Too many link types found
				res.WriteHeader(400)
				return
			}
		}
	}

	if linkType == "" {
		// No link type found
		res.WriteHeader(400)
		return
	}

	if linkType != linkTypeNonRDFSource {
		if contentType != "application/n-quads" && contentType != "application/ld+json" {
			res.WriteHeader(415)
			return
		}
	}

	if req.URL.Path == "/" {
		if linkType != linkTypeDirectContainer {
			res.WriteHeader(400)
			return
		}
		// Yikes
		// We have to do some smart diffing here :-/
	}

	if !pathRegex.MatchString(req.URL.Path) {
		res.WriteHeader(404)
		return
	}

	_ = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(req.URL.Path))
		if err == badger.ErrKeyNotFound {
			tail := strings.LastIndex(req.URL.Path, "/")
			parentItem, err := txn.Get([]byte(req.URL.Path[:tail]))
			if err != nil {
				return err
			}

			parent := &Resource{}
			err = parentItem.Value(func(val []byte) error {
				return proto.Unmarshal(val, parent)
			})

			if err == badger.ErrKeyNotFound {
				// Parent doesn't exist!
				res.WriteHeader(404)
				return nil
			} else if err != nil {
				res.WriteHeader(500)
				res.Write([]byte(err.Error()))
				return err
			}

			p := parent.GetPackage()
			if p == nil {
				// Parent is not a package!
				res.WriteHeader(404)
				return nil
			}

			// Okay so parent is an existing package and this is a new
			// route beneath it, with link type linkType.

		} else if err != nil {
			return err
		} else {
			resource := &Resource{}
			err = item.Value(func(val []byte) error {
				return proto.Unmarshal(val, resource)
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func create() {

}
