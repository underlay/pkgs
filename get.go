package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	badger "github.com/dgraph-io/badger/v2"
	proto "github.com/gogo/protobuf/proto"
	cid "github.com/ipfs/go-cid"
	ipfs "github.com/ipfs/go-ipfs-api"
	multibase "github.com/multiformats/go-multibase"
	ld "github.com/piprate/json-gold/ld"
)

const linkTypeResource = `<http://www.w3.org/ns/ldp#Resource>; rel="type"`
const linkTypeDirectContainer = `<http://www.w3.org/ns/ldp#DirectContainer>; rel="type"`
const linkTypeRDFSource = `<http://www.w3.org/ns/ldp#RDFSource>; rel="type"`
const linkTypeNonRDFSource = `<http://www.w3.org/ns/ldp#NonRDFSource>; rel="type"`

var linkTypes = map[string]bool{
	linkTypeDirectContainer: true,
	linkTypeRDFSource:       true,
	linkTypeNonRDFSource:    true,
}

func Get(res http.ResponseWriter, req *http.Request, root string, pkg *Package, sh *ipfs.Shell, db *badger.DB) {
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

		if accept != "application/ld+json" && accept != "application/n-quads" {
			res.WriteHeader(406)
			return
		}

		res.Header().Add("Content-Type", accept)

		if accept == "application/ld+json" {
			encoder := json.NewEncoder(res)
			_ = db.View(func(txn *badger.Txn) error {
				doc, err := pkg.JSON("/", txn)
				if err != nil {
					return err
				}
				return encoder.Encode(doc)
			})
		} else if accept == "application/n-quads" {
			reader, err := sh.Cat(root)
			if err != nil {
				res.WriteHeader(502)
				res.Write([]byte(err.Error()))
				return
			}

			_, _ = io.Copy(res, reader)
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

		reader, err := sh.Cat(s)
		if err != nil {
			res.WriteHeader(502)
			res.Write([]byte(err.Error()))
			return
		}

		extent := strconv.FormatUint(f.Extent, 10)
		res.Header().Add("ETag", s)
		res.Header().Add("Content-Type", f.Format)
		res.Header().Add("Content-Length", extent)
		_, _ = io.Copy(res, reader)
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

		res.Header().Add("Link", linkTypeNonRDFSource)
		if ifNoneMatch == s {
			res.WriteHeader(304)
			return
		}

		res.Header().Add("ETag", s)

		if accept == "application/n-quads" {
			reader, err := sh.Cat(s)
			if err != nil {
				res.WriteHeader(502)
				res.Write([]byte(err.Error()))
				return
			}

			res.Header().Add("Content-Type", accept)
			_, _ = io.Copy(res, reader)
		} else if accept == "application/ld+json" {
			reader, err := sh.Cat(s)
			if err != nil {
				res.WriteHeader(502)
				res.Write([]byte(err.Error()))
				return
			}

			opts := ld.NewJsonLdOptions("")
			opts.Format = "application/n-quads"

			doc, err := proc.FromRDF(reader, opts)
			if err != nil {
				res.WriteHeader(500)
				res.Write([]byte(err.Error()))
				return
			}

			res.Header().Add("Content-Type", accept)
			_ = json.NewEncoder(res).Encode(doc)
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

		res.Header().Add("Link", linkTypeDirectContainer)

		if ifNoneMatch == s {
			res.WriteHeader(304)
			return
		}

		res.Header().Add("ETag", s)

		if accept == "application/n-quads" {
			reader, err := sh.Cat(s)
			if err != nil {
				res.WriteHeader(502)
				res.Write([]byte(err.Error()))
				return
			}

			res.Header().Add("Content-Type", accept)
			res.Header().Add("Link", "FJkdlsfjkdsljfklsd")
			_, _ = io.Copy(res, reader)
		} else if accept == "application/ld+json" {
			reader, err := sh.Cat(s)
			if err != nil {
				res.WriteHeader(502)
				res.Write([]byte(err.Error()))
				return
			}

			opts := ld.NewJsonLdOptions("")
			opts.Format = "application/n-quads"

			doc, err := proc.FromRDF(reader, opts)
			if err != nil {
				res.WriteHeader(500)
				res.Write([]byte(err.Error()))
				return
			}

			res.Header().Add("Content-Type", accept)
			res.Header().Add("Link", "FJkdlsfjkdsljfklsd")
			_ = json.NewEncoder(res).Encode(doc)
		} else {
			res.WriteHeader(406)
			return
		}
	}

}
