package server

import (
	"context"
	"net/http"
	"regexp"
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
	loader "github.com/underlay/go-dweb-loader/loader"

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

// Server is the main pkgs struct
type Server struct {
	fs       core.UnixfsAPI
	object   core.ObjectAPI
	pin      core.PinAPI
	db       *badger.DB
	resource string
	api      *ld.JsonLdApi
	proc     *ld.JsonLdProcessor
	opts     *ld.JsonLdOptions
}

// Initialize opens the Badger database and writes an empty root package if none exists
func Initialize(ctx context.Context, badgerPath, resource string, api core.CoreAPI) (*Server, error) {
	opts := badger.DefaultOptions(badgerPath)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	server := &Server{
		fs:       api.Unixfs(),
		object:   api.Object(),
		pin:      api.Pin(),
		db:       db,
		resource: resource,
		proc:     ld.NewJsonLdProcessor(),
		opts: &ld.JsonLdOptions{
			OmitGraph:      true,
			CompactArrays:  true,
			UseNativeTypes: true,
			Format:         "application/n-quads",
			Algorithm:      "URGNA2012",
			DocumentLoader: loader.NewDwebDocumentLoader(api),
		},
	}

	fs, object, pin := api.Unixfs(), api.Object(), api.Pin()

	directory := path.IpfsPath(types.EmptyDirectoryCID)
	err = pin.Add(ctx, directory)
	if err != nil {
		return nil, err
	}

	r := &types.Resource{}
	err = db.View(func(txn *badger.Txn) error {
		return r.Get(index, txn)
	})

	if err == badger.ErrKeyNotFound {
		name := "context.jsonld"
		pkg := types.NewPackage(ctx, index, resource)
		pkg.Member = append(pkg.Member, name)

		value, err := fs.Add(ctx,
			files.NewMapDirectory(
				map[string]files.Node{
					name: files.NewBytesFile(types.RawContext),
				},
			),
			options.Unixfs.CidVersion(1),
			options.Unixfs.RawLeaves(true),
			options.Unixfs.Pin(true),
		)

		if err != nil {
			return nil, err
		}

		stat, err := object.Stat(ctx, path.Join(value, name))
		if err != nil {
			return nil, err
		}

		pkg.Value = value.Cid().Bytes()

		file := &types.File{
			Value:  stat.Cid.Bytes(),
			Format: "application/ld+json",
			Extent: uint64(stat.CumulativeSize),
		}

		fileResource := &types.Resource{}
		fileResource.Resource = &types.Resource_File{File: file}
		err = db.Update(func(txn *badger.Txn) (err error) {
			err = fileResource.Set("/"+name, txn)
			if err != nil {
				return
			}

			// This *has* to be done *afer* the fileResource is set,
			// because .Normalize() will use txn to look up its own children.
			_, err = server.Normalize(ctx, index, pkg, true, txn)
			if err != nil {
				return
			}

			return pkg.Set(index, txn)
		})

		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else if r.GetPackage() == nil {
		return nil, types.ErrNotPackage
	}

	return server, nil
}

// Normalize re-computes the normalized n-quads representation of the package,
// pins it to IPFS, and sets the pkg.Id with the result. It returns the string cid.
// You probably want to be careful about unpinning the resulting CID sometime afterwards
func (server *Server) Normalize(ctx context.Context, path string, pkg *types.Package, pin bool, txn *badger.Txn) (c cid.Cid, err error) {
	ds := ld.NewRDFDataset()
	ds.Graphs["@default"], err = pkg.NQuads(path, txn)
	if err != nil {
		return
	}

	var res interface{}
	res, err = server.api.Normalize(ds, server.opts)
	if err != nil {
		return
	}

	reader := strings.NewReader(res.(string))
	resolved, err := server.fs.Add(
		ctx,
		files.NewReaderFile(reader),
		options.Unixfs.Pin(pin),
		options.Unixfs.RawLeaves(true),
		options.Unixfs.CidVersion(1),
	)

	if err != nil {
		return
	}

	c = resolved.Cid()
	pkg.Id = c.Bytes()
	return
}

// Handle handles HTTP requests using the database and core API
func (server *Server) Handle(res http.ResponseWriter, req *http.Request) {
	var err error
	ctx := context.TODO()
	if req.Method == "GET" {
		err = server.Get(ctx, res, req)
	} else if req.Method == "PUT" {
		err = server.Put(ctx, res, req)
	} else if req.Method == "POST" {
		err = server.Post(ctx, res, req)
	} else if req.Method == "HEAD" {
		err = server.Head(ctx, res, req)
	} else if req.Method == "DELETE" {
		err = server.Delete(ctx, res, req)
	} else if req.Method == "OPTIONS" {
	} else if req.Method == "COPY" {
	} else if req.Method == "LOCK" {
	} else if req.Method == "MKCOL" {
		err = server.Mkcol(ctx, res, req)
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

func (server *Server) percolate(
	ctx context.Context,
	parentPath string,
	parentID path.Resolved,
	parentValue path.Resolved,
	parent *types.Package,
	name string,
	value path.Resolved,
	txn *badger.Txn,
) (err error) {
	modified := time.Now().Format(time.RFC3339)
	for {
		// First patch the parent's value directory object
		if value != nil {
			value, err = server.object.AddLink(ctx, parentValue, name, value)
			if err != nil {
				return err
			}
		} else {
			value = parentValue
		}

		stat, err := server.object.Stat(ctx, value)
		if err != nil {
			return err
		}

		parent.Extent = uint64(stat.CumulativeSize)
		parent.Value = value.Cid().Bytes()
		parent.Modified = modified
		parent.RevisionOf = parent.Id

		// Now that parent.Value has changed, we need to re-normalize
		id, err := server.Normalize(ctx, parentPath, parent, false, txn)
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
			err = server.pin.Update(ctx, parentValue, value, options.Pin.Unpin(unpin))
			if err != nil {
				return err
			}

			err = server.pin.Update(ctx, parentID, nextID, options.Pin.Unpin(true))
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

		parentValue, err = server.object.AddLink(ctx, parentValue, name+".nt", nextID)
		if err != nil {
			return err
		}
	}
}
