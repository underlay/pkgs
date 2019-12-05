package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	cid "github.com/ipfs/go-cid"
	core "github.com/ipfs/interface-go-ipfs-core"
	options "github.com/ipfs/interface-go-ipfs-core/options"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	multibase "github.com/multiformats/go-multibase"
	ld "github.com/piprate/json-gold/ld"
	types "github.com/underlay/pkgs/types"
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

var pathRegex = regexp.MustCompile("^(/[a-zA-Z0-9-\\.]+)+$")

var proc = ld.NewJsonLdProcessor()

var debug = true

var index = "/"

func Initialize(ctx context.Context, p, resource string, api core.CoreAPI) (*badger.DB, error) {
	opts := badger.DefaultOptions(p)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return db, db.Update(func(txn *badger.Txn) error {
		r := &types.Resource{}
		err := r.Get(index, txn)
		if err == badger.ErrKeyNotFound {
			_, p, err := types.NewPackage(ctx, index, resource, api.Unixfs())
			if err != nil {
				return err
			}

			r.Resource = &types.Resource_Package{Package: p}
			return r.Set(index, txn)
		} else if err != nil {
			return err
		}

		pkg := r.GetPackage()
		if pkg == nil {
			return fmt.Errorf("Invalid index: %v", r)
		}

		return err
	})
}

// Handler handles HTTP requests using the database and core API
func Handler(res http.ResponseWriter, req *http.Request, db *badger.DB, api core.CoreAPI) {
	var err error
	ctx := context.TODO()
	if req.Method == "GET" {
		err = Get(ctx, res, req, db, api)
	} else if req.Method == "PUT" {
		err = Put(ctx, res, req, db, api)
	} else if req.Method == "POST" {
		err = Post(ctx, res, req, db, api)
	} else if req.Method == "HEAD" {
		err = Head(ctx, res, req, db, api)
	} else if req.Method == "DELETE" {
		err = Delete(ctx, res, req, db, api)
	} else if req.Method == "OPTIONS" {
	} else if req.Method == "COPY" {
	} else if req.Method == "LOCK" {
	} else if req.Method == "MKCOL" {
		err = Mkcol(ctx, res, req, db, api)
	} else if req.Method == "MOVE" {
	} else if req.Method == "PROPFIND" {
	} else if req.Method == "PROPPATCH" {
	} else if req.Method == "UNLOCK" {
	}

	if err != nil {
		res.Write([]byte(err.Error()))
		res.Write([]byte("\n"))
	}

	return
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
	modified := time.Now().Format(time.RFC3339)
	for {
		// First patch the parent's value directory object
		if value != nil {
			value, err = object.AddLink(ctx, parentValue, name, value)
			if err != nil {
				if debug {
					log.Println("PUT: error patching parent value link", name)
				}
				return err
			}
		} else {
			value = parentValue
		}

		stat, err := object.Stat(ctx, value)
		if err != nil {
			return err
		}

		parent.Extent = uint64(stat.CumulativeSize)
		parent.Value = value.Cid().Bytes()
		parent.Modified = modified

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
