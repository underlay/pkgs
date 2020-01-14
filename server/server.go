package server

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	files "github.com/ipfs/go-ipfs-files"
	core "github.com/ipfs/interface-go-ipfs-core"
	options "github.com/ipfs/interface-go-ipfs-core/options"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	multibase "github.com/multiformats/go-multibase"
	ld "github.com/underlay/json-gold/ld"

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

var PathRegex = regexp.MustCompile("^(/[a-zA-Z0-9-\\.]+)+$")

var etagRegex = regexp.MustCompile("^\"([a-z2-7]{59})\"$")

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
	locks    map[string]*sync.Mutex
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
		locks:    map[string]*sync.Mutex{},
		proc:     ld.NewJsonLdProcessor(),
		opts: &ld.JsonLdOptions{
			OmitGraph:      true,
			CompactArrays:  true,
			UseNativeTypes: true,
			Format:         "application/n-quads",
			Algorithm:      "URDNA2015",
			DocumentLoader: ld.NewDwebDocumentLoader(api),
		},
	}

	fs, object, pin := api.Unixfs(), api.Object(), api.Pin()

	directory := path.IpfsPath(types.EmptyDirectoryCID)
	err = pin.Add(ctx, directory)
	if err != nil {
		return nil, err
	}

	var u types.ResourceType
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(index))
		if err == nil {
			u = types.ResourceType(item.UserMeta())
		}
		return err
	})

	if err == badger.ErrKeyNotFound {
		name := "context.jsonld"
		pkg := types.NewPackage(ctx, time.Now(), index, resource)
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

		pkg.Value = value.Cid().Bytes()

		rootStat, err := object.Stat(ctx, value)
		if err != nil {
			return nil, err
		}

		pkg.Extent = uint64(rootStat.CumulativeSize)

		fileStat, err := object.Stat(ctx, path.Join(value, name))
		if err != nil {
			return nil, err
		}

		file := &types.File{
			Value:  fileStat.Cid.Bytes(),
			Format: "application/ld+json",
			Extent: uint64(fileStat.CumulativeSize),
		}

		err = db.Update(func(txn *badger.Txn) (err error) {
			err = types.SetResource(file, "/"+name, txn)
			if err != nil {
				return
			}

			// This *has* to be done *afer* the fileResource is set,
			// because .Normalize() will use txn to look up its own children.
			_, err = server.Normalize(ctx, index, pkg, true, txn)
			if err != nil {
				return
			}

			return types.SetResource(pkg, index, txn)
		})

		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else if u != types.PackageType {
		return nil, types.ErrNotPackage
	}

	return server, nil
}

// Normalize re-computes the normalized n-quads representation of the package,
// pins it to IPFS, and sets the pkg.Id with the result. It returns the string cid.
// You probably want to be careful about unpinning the resulting CID sometime afterwards
func (server *Server) Normalize(
	ctx context.Context,
	path string,
	pkg *types.Package,
	pin bool,
	txn *badger.Txn,
) (resolved path.Resolved, err error) {
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
	resolved, err = server.fs.Add(
		ctx,
		files.NewReaderFile(reader),
		options.Unixfs.Pin(pin),
		options.Unixfs.RawLeaves(true),
		options.Unixfs.CidVersion(1),
	)

	if err != nil {
		return
	}

	pkg.Id = resolved.Cid().Bytes()
	return
}

// Handle handles HTTP requests using the database and core API
func (server *Server) Handle(res http.ResponseWriter, req *http.Request) {
	var err error
	ctx := context.TODO()
	if req.Method == "GET" {
		err = server.Get(ctx, res, req)
	} else if req.Method == "HEAD" {
		err = server.Head(ctx, res, req)
	} else if req.Method == "POST" {
		err = server.Post(ctx, res, req)
	} else if req.Method == "PUT" {
		err = server.Put(ctx, res, req)
	} else if req.Method == "DELETE" {
		err = server.Delete(ctx, res, req)
	} else if req.Method == "PATCH" {
		err = server.Patch(ctx, res, req)
	} else if req.Method == "COPY" {
	} else if req.Method == "LOCK" {
	} else if req.Method == "MKCOL" {
		err = server.Mkcol(ctx, res, req)
	} else if req.Method == "MOVE" {
	} else if req.Method == "UNLOCK" {
	} else {
		res.WriteHeader(405)
	}

	if err != nil {
		res.Write([]byte(err.Error()))
		res.Write([]byte("\n"))
	}

	return
}

func (server *Server) percolate(
	ctx context.Context,
	modified time.Time,
	pathname string,
	p *types.Package,
	oldID, oldValue path.Resolved,
	value path.Resolved,
	txn *badger.Txn,
) (err error) {
	var stat *core.ObjectStat
	var s string
	var id path.Resolved

	m := modified.Format(time.RFC3339)

	for {

		if value != nil {
			stat, err = server.object.Stat(ctx, value)
			if err != nil {
				return
			}

			p.Extent = uint64(stat.CumulativeSize)
			p.Value = value.Cid().Bytes()
		}

		p.Modified = m
		p.RevisionOf = p.Id
		p.RevisionOfSubject = p.Subject

		// Now that parent.Value has changed, we need to re-normalize
		id, err = server.Normalize(ctx, pathname, p, false, txn)
		if err != nil {
			return
		}

		err = types.SetResource(p, pathname, txn)
		if err != nil {
			return
		}

		if pathname == "/" {
			if value != nil {
				s, err = oldValue.Cid().StringOfBase(multibase.Base32)
				if err != nil {
					return
				}

				unpin := options.Pin.Unpin(s != types.EmptyDirectory)
				err = server.pin.Update(ctx, oldValue, value, unpin)
				if err != nil {
					return
				}
			}

			err = server.pin.Update(ctx, oldID, id)
			if err != nil {
				return
			}

			// ...
			return
		}

		tail := strings.LastIndex(pathname, "/")
		name := pathname[tail+1:]
		if tail > 0 {
			pathname = pathname[:tail]
		} else {
			pathname = "/"
		}

		p, err = types.GetPackage(pathname, txn)
		if err != nil {
			return
		}

		oldID, oldValue, err = p.Paths()
		if err != nil {
			return
		}

		// First patch the parent's value directory object
		value, err = server.object.AddLink(ctx, oldValue, name, value)
		if err != nil {
			return
		}

		value, err = server.object.AddLink(ctx, value, name+".nt", id)
		if err != nil {
			return
		}
	}
}

func (server *Server) parseDataset(body io.Reader, contentType string) (quads []*ld.Quad, err error) {
	var rdf interface{}
	if contentType == "application/n-quads" {
		ns := &ld.NQuadRDFSerializer{}
		rdf, err = ns.Parse(body)
		if err != nil {
			return
		}
	} else if contentType == "application/ld+json" {
		opts := ld.NewJsonLdOptions("")
		opts.DocumentLoader = server.opts.DocumentLoader
		rdf, err = server.proc.ToRDF(body, opts)
		if err != nil {
			return
		}
	} else {
		return nil, nil
	}

	na := ld.NewNormalisationAlgorithm(server.opts.Algorithm)
	na.Normalize(rdf.(*ld.RDFDataset))
	return na.Quads(), nil
}
