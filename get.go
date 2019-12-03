package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	badger "github.com/dgraph-io/badger/v2"
	proto "github.com/gogo/protobuf/proto"
	ipfs "github.com/ipfs/go-ipfs-api"
	ld "github.com/piprate/json-gold/ld"
)

func Get(res http.ResponseWriter, req *http.Request, pkg *Package, sh *ipfs.Shell, db *badger.DB) {
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
				res.Write([]byte(err.Error()))
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

	// Okay now we have a Resource and we get to respond with its representation
	p, m, f := resource.GetPackage(), resource.GetMessage(), resource.GetFile()
	if f != nil {
		if ifNoneMatch == f.Value {
			res.WriteHeader(304)
			return
		}

		reader, err := sh.Cat(f.Value)
		if err != nil {
			res.WriteHeader(502)
			res.Write([]byte(err.Error()))
			return
		}

		extent := strconv.FormatUint(f.Extent, 10)
		res.Header().Add("ETag", f.Value)
		res.Header().Add("Content-Type", f.Format)
		res.Header().Add("Content-Length", extent)
		_, _ = io.Copy(res, reader)
	} else if m != "" {
		if ifNoneMatch == m {
			res.WriteHeader(304)
			return
		}

		if accept == "application/n-quads" {
			reader, err := sh.Cat(m)
			if err != nil {
				res.WriteHeader(502)
				res.Write([]byte(err.Error()))
				return
			}

			res.Header().Add("ETag", m)
			res.Header().Add("Content-Type", accept)
			_, _ = io.Copy(res, reader)
		} else if accept == "application/ld+json" {
			reader, err := sh.Cat(m)
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

			res.Header().Add("ETag", m)
			res.Header().Add("Content-Type", accept)
			_ = json.NewEncoder(res).Encode(doc)
		} else {
			res.WriteHeader(406)
			return
		}
	} else if p != nil {
		if ifNoneMatch == p.Id {
			res.WriteHeader(304)
			return
		}

		if accept == "application/n-quads" {
			reader, err := sh.Cat(p.Id)
			if err != nil {
				res.WriteHeader(502)
				res.Write([]byte(err.Error()))
				return
			}

			res.Header().Add("ETag", p.Id)
			res.Header().Add("Content-Type", accept)
			res.Header().Add("Link", "FJkdlsfjkdsljfklsd")
			_, _ = io.Copy(res, reader)
		} else if accept == "application/ld+json" {
			reader, err := sh.Cat(p.Id)
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

			res.Header().Add("ETag", p.Id)
			res.Header().Add("Content-Type", accept)
			res.Header().Add("Link", "FJkdlsfjkdsljfklsd")
			_ = json.NewEncoder(res).Encode(doc)
		} else {
			res.WriteHeader(406)
			return
		}
	}

}
