package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	cid "github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	core "github.com/ipfs/interface-go-ipfs-core"
	options "github.com/ipfs/interface-go-ipfs-core/options"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	multibase "github.com/multiformats/go-multibase"
	ld "github.com/piprate/json-gold/ld"

	types "github.com/underlay/pkgs/types"
)

func Put(ctx context.Context, res http.ResponseWriter, req *http.Request, db *badger.DB, api core.CoreAPI) error {
	if debug {
		log.Println("PUT:", req.URL.Path)
	}

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

	if debug {
		log.Println("PUT:", linkType)
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

	if debug {
		log.Println("PUT: opening transaction")
	}

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

		parent := &types.Resource{}
		err := parent.Get(parentPath, txn)
		if err == badger.ErrKeyNotFound {
			// Parent doesn't exist!
			res.WriteHeader(404)
			return nil
		} else if err != nil {
			res.WriteHeader(500)
			return err
		}

		p := parent.GetPackage()
		if p == nil {
			// Parent is not a package!
			res.WriteHeader(404)
			return nil
		}

		var leaf cid.Cid
		resource := &types.Resource{}
		err = resource.Get(pathname, txn)
		if err == badger.ErrKeyNotFound {
			// Okay so parent is an existing package and this is a new
			// route beneath it, with link type linkType.

			if debug {
				log.Println("PUT: creating new resource")
			}

			// It's safe to start mutating p because it we encouter
			// errors we'll return before we write it back to the database
			p.Member = append(p.Member, name)

			if linkType == linkTypeNonRDFSource {
				// New file!

				resolved, err := fs.Add(
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

				leaf = resolved.Cid()

				if debug {
					log.Println("PUT: new file with CID", leaf.String())
				}

				stat, err := object.Stat(ctx, resolved)
				if err != nil {
					res.WriteHeader(502)
					return err
				}

				file := &types.File{
					Value:  leaf.Bytes(),
					Format: contentType,
					Extent: uint64(stat.CumulativeSize),
				}

				resource.Resource = &types.Resource_File{File: file}
			} else if linkType == linkTypeRDFSource {
				// New message!
				var doc interface{}
				opts := ld.NewJsonLdOptions("")
				opts.Format = "application/n-quads"
				if contentType == "application/ld+json" {
					doc = req.Body
				} else if contentType == "application/n-quads" {
					doc, err = proc.FromRDF(req.Body, opts)
					if err != nil {
						res.WriteHeader(400)
						return err
					}
				}

				n, err := proc.Normalize(doc, opts)
				if err != nil {
					res.WriteHeader(400)
					return err
				}

				m, is := n.(string)
				if !is {
					res.WriteHeader(400)
					return nil
				}

				resolved, err := fs.Add(
					ctx,
					files.NewBytesFile([]byte(m)),
					options.Unixfs.Pin(true),
					options.Unixfs.RawLeaves(true),
					options.Unixfs.CidVersion(1),
				)

				if err != nil {
					res.WriteHeader(502)
					return err
				}

				leaf = resolved.Cid()
				resource.Resource = &types.Resource_Message{Message: leaf.Bytes()}
			} else if linkType == linkTypeDirectContainer {
				// New subpackage!
				// ... we'll implement this later :-/
				res.WriteHeader(501)
				return nil
			}

			err = resource.Set(pathname, txn)

			if debug {
				log.Println("PUT: set resource", pathname, err)
			}

			if err != nil {
				res.WriteHeader(500)
				return err
			}
		} else if err != nil {
			res.WriteHeader(500)
			return err
		} else {
			// Something about unpinning its dependencies...
			// TODO think about diffing
			if debug {
				log.Println("PUT: updating existing resource")
			}
		}

		if debug {
			log.Println("PUT: setting resource")
		}

		// Leaf has been pinned to IPFS directly, so what we really want is to unpin it afterwards
		from := path.IpfsPath(leaf)
		parentValue, err := cid.Cast(p.Value)
		if err != nil {
			return err
		}

		parentID, err := cid.Cast(p.Id)
		if err != nil {
			return err
		}

		if debug {
			log.Println("PUT: percolating merkle tree", parentPath, name)
		}

		err = percolate(ctx,
			parentPath,
			path.IpfsPath(parentID),
			path.IpfsPath(parentValue),
			p, name, from,
			txn, fs, object, pin,
		)

		if err != nil {
			if debug {
				log.Println("PUT: error percolating", err)
			}

			res.WriteHeader(500)
			return err
		}

		err = pin.Rm(ctx, from, options.Pin.RmRecursive(true))
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		return nil
	})
}

func percolate(
	ctx context.Context,
	parentPath string,
	parentID path.Resolved,
	parentValue path.Resolved,
	parent *types.Package,
	name string,
	value path.Resolved,
	txn *badger.Txn,
	fs core.UnixfsAPI,
	object core.ObjectAPI,
	pin core.PinAPI,
) error {
	var err error
	for {
		// First patch the parent's value directory object
		value, err = object.AddLink(ctx, parentValue, name, value)
		if err != nil {
			if debug {
				log.Println("PUT: error patching parent value link", name)
			}
			return err
		}

		stat, err := object.Stat(ctx, value)
		if err != nil {
			return err
		}

		parent.Extent = uint64(stat.CumulativeSize)
		parent.Value = value.Cid().Bytes()
		parent.Modified = time.Now().Format(time.RFC3339)

		// Now that parent.Value has changed, we need to re-normalize
		id, err := parent.Normalize(ctx, parentPath, fs, txn)
		if err != nil {
			if debug {
				log.Println("PUT: error normalizing parent", parentPath)
			}
			return err
		}

		r := &types.Resource{}
		r.Resource = &types.Resource_Package{Package: parent}
		err = r.Set(parentPath, txn)
		if err != nil {
			if debug {
				log.Println("PUT: error setting resource", parentPath)
			}
			return err
		}

		next := path.IpfsPath(id)

		if parentPath == "/" {
			s, err := parentValue.Cid().StringOfBase(multibase.Base32)
			if err != nil {
				return err
			}

			unpin := s != types.EmptyDirectory
			err = pin.Update(ctx, parentValue, value, options.Pin.Unpin(unpin))
			if err != nil {
				if debug {
					log.Println("PUT: error updating parent value pin", s)
				}
				return err
			}

			err = pin.Update(ctx, parentID, next, options.Pin.Unpin(true))
			if err != nil {
				if debug {
					log.Println("PUT: error updating parent value pin", next.Cid().String())
				}
				return err
			}

			return nil
		}

		parentID = next

		tail := strings.LastIndex(parentPath, "/")
		name = parentPath[tail+1:]
		parentPath = parentPath[:tail]

		resource := &types.Resource{}
		err = resource.Get(parentPath, txn)
		if err != nil {
			if debug {
				log.Println("PUT: error getting resource", parentPath)
			}
			return err
		}

		parent = resource.GetPackage()
		if parent == nil {
			return fmt.Errorf("Invalid parent resource: %v", r)
		}

		// Since there's another directory above this, we also need to patch
		// *that* with the new package *id* under `name.nt` in the grandparent directory
		parentValueCid, err := cid.Cast(parent.Value)
		if err != nil {
			return err
		}

		parentValue = path.IpfsPath(parentValueCid)
		parentValue, err = object.AddLink(ctx, parentValue, fmt.Sprintf("%s.nt", name), parentID)
		if err != nil {
			if debug {
				log.Println("PUT: error patching ID link", name, parentID.Cid().String())
			}
			return err
		}
	}
}
