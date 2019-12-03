package main

import (
	"net/http"
	"strconv"

	badger "github.com/dgraph-io/badger/v2"
	proto "github.com/gogo/protobuf/proto"
	cid "github.com/ipfs/go-cid"
	ipfs "github.com/ipfs/go-ipfs-api"
	"github.com/multiformats/go-multibase"
)

func Head(res http.ResponseWriter, req *http.Request, root string, pkg *Package, sh *ipfs.Shell, db *badger.DB) {
	accept := req.Header.Get("Accept")
	ifNoneMatch := req.Header.Get("If-None-Match")
	if req.URL.Path == "/" {
		res.Header().Add("Link", linkTypeResource)
		res.Header().Add("Link", linkTypeDirectContainer)
		if ifNoneMatch == root {
			res.WriteHeader(304)
			return
		}

		res.Header().Add("ETag", root)
		if accept == "application/ld+json" || accept == "application/n-quads" {
			res.Header().Add("Content-Type", accept)
		} else {
			res.WriteHeader(406)
		}
		return
	}

	if !pathRegex.MatchString(req.URL.Path) {
		res.WriteHeader(404)
		return
	}

	resource := &Resource{}
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(req.URL.Path))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			return proto.Unmarshal(val, resource)
		})
		if err != nil {
			return err
		}

		return nil
	})

	if err == badger.ErrKeyNotFound {
		res.WriteHeader(404)
		return
	} else if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(err.Error()))
		return
	}

	res.Header().Add("Link", linkTypeResource)

	// Okay now we have a Resource and we get to respond with its representation
	p, m, f := resource.GetPackage(), resource.GetMessage(), resource.GetFile()
	if f != nil {
		c, err := cid.Cast(f.Value)
		if err != nil {
			res.WriteHeader(500)
			res.Write([]byte(err.Error()))
			return
		}
		s, err := c.StringOfBase(multibase.Base32)
		if err != nil {
			res.WriteHeader(500)
			res.Write([]byte(err.Error()))
			return
		}

		res.Header().Add("Link", linkTypeNonRDFSource)

		if ifNoneMatch == s {
			res.WriteHeader(304)
			return
		}

		extent := strconv.FormatUint(f.Extent, 10)
		res.Header().Add("ETag", s)
		res.Header().Add("Content-Type", f.Format)
		res.Header().Add("Content-Length", extent)
	} else if m != nil {
		c, err := cid.Cast(m)
		if err != nil {
			res.WriteHeader(500)
			res.Write([]byte(err.Error()))
			return
		}
		s, err := c.StringOfBase(multibase.Base32)
		if err != nil {
			res.WriteHeader(500)
			res.Write([]byte(err.Error()))
			return
		}

		res.Header().Add("Link", linkTypeRDFSource)

		if ifNoneMatch == s {
			res.WriteHeader(304)
			return
		}

		res.Header().Add("ETag", s)
		if accept == "application/n-quads" || accept == "application/ld+json" {
			res.Header().Add("Content-Type", accept)
		} else {
			res.WriteHeader(406)
			return
		}
	} else if p != nil {
		c, err := cid.Cast(p.Id)
		if err != nil {
			res.WriteHeader(500)
			res.Write([]byte(err.Error()))
			return
		}
		s, err := c.StringOfBase(multibase.Base32)
		if err != nil {
			res.WriteHeader(500)
			res.Write([]byte(err.Error()))
			return
		}
		if ifNoneMatch == s {
			res.WriteHeader(304)
			return
		}

		if accept == "application/n-quads" || accept == "application/ld+json" {
			res.Header().Add("ETag", s)
			res.Header().Add("Content-Type", accept)
			res.Header().Add("Link", "FJkdlsfjkdsljfklsd")
		} else {
			res.WriteHeader(406)
			return
		}
	}
}
