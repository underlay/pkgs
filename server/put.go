package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	badger "github.com/dgraph-io/badger/v2"
	files "github.com/ipfs/go-ipfs-files"
	core "github.com/ipfs/interface-go-ipfs-core"
	options "github.com/ipfs/interface-go-ipfs-core/options"
	path "github.com/ipfs/interface-go-ipfs-core/path"

	types "github.com/underlay/pkgs/types"
)

// Put handles HTTP PUT requests
func Put(ctx context.Context, res http.ResponseWriter, req *http.Request, db *badger.DB, api core.CoreAPI) error {
	fs, object, pin := api.Unixfs(), api.Object(), api.Pin()
	contentType := req.Header.Get("Content-Type")
	if contentType == "" || len(req.Header["Content-Type"]) != 1 {
		// Content-Type is required for all requests.
		res.WriteHeader(400)
		return nil
	}

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
				return nil
			}
		}
	}

	if linkType == "" {
		// No link type found
		res.WriteHeader(400)
		return nil
	}

	if linkType != linkTypeNonRDFSource {
		if contentType != "application/n-quads" && contentType != "application/ld+json" {
			res.WriteHeader(415)
			return nil
		}
	}

	pathname := req.URL.Path

	// We have to do some smart diffing here :-/
	// Forget about it for now
	if pathname == "/" {
		if linkType == linkTypeDirectContainer {
			res.WriteHeader(501)
		} else {
			res.WriteHeader(400)
		}
		return nil
	}

	if !pathRegex.MatchString(pathname) {
		res.WriteHeader(404)
		return nil
	}

	ifMatch := req.Header.Get("If-Match")

	return db.Update(func(txn *badger.Txn) error {
		var parentPath string
		tail := strings.LastIndex(pathname, "/")
		if tail > 0 {
			parentPath = pathname[:tail]
		} else {
			parentPath = "/"
			tail = 0
		}

		name := pathname[tail+1:]

		parentResource := &types.Resource{}
		err := parentResource.Get(parentPath, txn)
		if err == badger.ErrKeyNotFound {
			// Parent doesn't exist!
			res.WriteHeader(404)
			return nil
		} else if err != nil {
			res.WriteHeader(500)
			return err
		}

		parent := parentResource.GetPackage()
		if parent == nil {
			// Parent is not a package!
			res.WriteHeader(409)
			return nil
		}

		parentID, parentValue, err := parent.Paths()
		if err != nil {
			return err
		}

		var leaf path.Resolved
		var created bool
		var etag string
		resource := &types.Resource{}
		err = resource.Get(pathname, txn)
		if err == badger.ErrKeyNotFound {
			// Okay so parent is an existing package and this is a new
			// route beneath it, with link type linkType.
			created = true

			// It's safe to start mutating p because it we encouter
			// errors we'll return before we write it back to the database
			parent.Member = append(parent.Member, name)

			if linkType == linkTypeNonRDFSource {
				// New file!
				leaf, err = fs.Add(
					ctx,
					files.NewReaderFile(req.Body),
					options.Unixfs.Pin(true),
					options.Unixfs.RawLeaves(true),
					options.Unixfs.CidVersion(1),
				)

				if err != nil {
					res.WriteHeader(502)
					return err
				}

				stat, err := object.Stat(ctx, leaf)
				if err != nil {
					res.WriteHeader(502)
					return err
				}

				file := &types.File{
					Value:  leaf.Cid().Bytes(),
					Format: contentType,
					Extent: uint64(stat.CumulativeSize),
				}

				resource.Resource = &types.Resource_File{File: file}
			} else if linkType == linkTypeRDFSource {
				// New message!
				var doc interface{}
				if contentType == "application/ld+json" {
					doc = req.Body
				} else if contentType == "application/n-quads" {
					doc, err = types.Proc.FromRDF(req.Body, types.Opts)
					if err != nil {
						res.WriteHeader(400)
						return err
					}
				}

				n, err := types.Proc.Normalize(doc, types.Opts)
				if err != nil {
					res.WriteHeader(400)
					return err
				}

				m, is := n.(string)
				if !is {
					res.WriteHeader(400)
					return nil
				}

				reader := strings.NewReader(m)

				leaf, err = fs.Add(
					ctx,
					files.NewReaderFile(reader),
					options.Unixfs.Pin(true),
					options.Unixfs.RawLeaves(true),
					options.Unixfs.CidVersion(1),
				)

				if err != nil {
					res.WriteHeader(502)
					return err
				}

				resource.Resource = &types.Resource_Message{Message: leaf.Cid().Bytes()}
			} else if linkType == linkTypeDirectContainer {
				// New subpackage!
				// ... we'll implement this later :-/
				res.WriteHeader(501)
				return nil
			}

			_, etag, err = types.GetCid(resource.ETag())
			if err != nil {
				res.WriteHeader(500)
				return err
			}

			err = resource.Set(pathname, txn)
			if err != nil {
				res.WriteHeader(500)
				return err
			}
		} else if err != nil {
			res.WriteHeader(500)
			return err
		} else {
			// The resource already exists!
			// For now we can at least check the If-Match tag
			_, etag, err = types.GetCid(resource.ETag())
			if err != nil {
				res.WriteHeader(500)
				return err
			}

			if etag != ifMatch {
				res.WriteHeader(416)
				return nil
			}

			if p := resource.GetPackage(); p != nil {
				txn.Delete([]byte(fmt.Sprintf("%s.nt", pathname)))

				prefix := []byte(fmt.Sprintf("%s/", pathname))
				iter := txn.NewIterator(badger.IteratorOptions{
					PrefetchValues: false,
					Prefix:         prefix,
				})

				for iter.Seek(prefix); iter.Valid(); iter.Next() {
					txn.Delete(iter.Item().Key())
				}
			}

			res.WriteHeader(501)
			return nil
		}

		err = percolate(ctx,
			parentPath,
			parentID,
			parentValue,
			parent, name, leaf,
			txn, api,
		)

		if err != nil {
			res.WriteHeader(500)
			return err
		}

		// The leaf was pinned directly when we added it,
		// so we should unpin in here. It'll still be indirectly
		// pin through the directory tree.
		err = pin.Rm(ctx, leaf, options.Pin.RmRecursive(true))
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		res.Header().Add("ETag", etag)
		if created {
			res.WriteHeader(201)
		} else {
			res.WriteHeader(200)
		}

		res.Write(nil)
		return nil
	})
}
