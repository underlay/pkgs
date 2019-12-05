package server

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	core "github.com/ipfs/interface-go-ipfs-core"
	options "github.com/ipfs/interface-go-ipfs-core/options"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	multibase "github.com/multiformats/go-multibase"
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

var index = "/"

// Initialize opens the Badger database and writes an empty root package if none exists
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

		if r.GetPackage() == nil {
			return types.ErrNotPackage
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
	api core.CoreAPI,
) (err error) {
	fs, object, pin := api.Unixfs(), api.Object(), api.Pin()
	modified := time.Now().Format(time.RFC3339)
	for {
		// First patch the parent's value directory object
		if value != nil {
			value, err = object.AddLink(ctx, parentValue, name, value)
			if err != nil {
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
		parent.RevisionOf = parent.Id

		// Now that parent.Value has changed, we need to re-normalize
		id, err := parent.Normalize(ctx, parentPath, fs, txn)
		if err != nil {
			return err
		}

		r := &types.Resource{}
		r.Resource = &types.Resource_Package{Package: parent}
		err = r.Set(parentPath, txn)
		if err != nil {
			return err
		}

		nextID := path.IpfsPath(id)

		if parentPath == "/" {
			s, err := parentValue.Cid().StringOfBase(multibase.Base32)
			if err != nil {
				return err
			}

			unpin := s != types.EmptyDirectory
			err = pin.Update(ctx, parentValue, value, options.Pin.Unpin(unpin))
			if err != nil {
				return err
			}

			err = pin.Update(ctx, parentID, nextID, options.Pin.Unpin(true))
			if err != nil {
				return err
			}

			return nil
		}

		tail := strings.LastIndex(parentPath, "/")
		name = parentPath[tail+1:]
		parentPath = parentPath[:tail]

		parent, err := types.GetPackage(parentPath, txn)
		if err != nil {
			return err
		}

		parentID, parentValue, err = parent.Paths()
		if err != nil {
			return err
		}

		parentValue, err = object.AddLink(ctx, parentValue, fmt.Sprintf("%s.nt", name), nextID)
		if err != nil {
			return err
		}
	}
}
