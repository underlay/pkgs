package main

import (
	"context"
	"log"
	"regexp"
	"strings"
	"sync"

	badger "github.com/dgraph-io/badger/v2"
	files "github.com/ipfs/go-ipfs-files"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	multibase "github.com/multiformats/go-multibase"
	ld "github.com/piprate/json-gold/ld"
	loader "github.com/underlay/go-dweb-loader"

	rpc "github.com/underlay/pkgs/rpc"
	types "github.com/underlay/pkgs/types"
)

// Server is the main pkgs struct
type Server struct {
	api            coreiface.CoreAPI
	db             *badger.DB
	resource       string
	documentLoader ld.DocumentLoader
	mutex          sync.Mutex
	id             path.Resolved
	value          path.Resolved
}

// Close the underlying badger database
func (server *Server) Close() { server.db.Close() }

var links = map[string]string{}

// NewServer opens the Badger database and writes an empty root package if none exists
func NewServer(ctx context.Context, resource string, db *badger.DB, api coreiface.CoreAPI) (*Server, error) {
	documentLoader := loader.NewDwebDocumentLoader(api)
	server := &Server{api, db, resource, documentLoader, sync.Mutex{}, nil, nil}

	contents := make([]*types.File, len(initialFiles))
	for i, init := range initialFiles {
		name, format, data := init[0], init[1], init[2]
		f := &types.File{Format: format}
		f.Resource = server.resource + "/" + name
		f.Title = name

		node := files.NewBytesFile([]byte(data))
		err := server.setFile(ctx, f, node)
		if err != nil {
			return nil, err
		}

		links[name] = f.ID
		contents[i] = f
	}

	for name, link := range links {
		log.Println(name+":\t", link)
	}

	txn := db.NewTransaction(true)
	defer txn.Discard()
	pkg, err := getPackage(nil, txn)
	if err == badger.ErrKeyNotFound {
		log.Println("No root package found; creating new initial package")
		server.id, server.value, err = server.createInitialPackage(ctx, contents, txn)
		if err != nil {
			return nil, err
		}
		err = txn.Commit()
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		server.id, server.value = pkg.Path(), pkg.ValuePath()
	}

	err = server.api.Pin().Add(ctx, server.id)
	if err != nil {
		return nil, err
	}

	err = server.api.Pin().Add(ctx, server.value)
	if err != nil {
		return nil, err
	}

	return server, nil
}

func (server *Server) createInitialPackage(
	ctx context.Context,
	files []*types.File,
	txn *badger.Txn,
) (id, value path.Resolved, err error) {
	server.mutex.Lock()
	defer server.mutex.Unlock()
	pkg := types.NewPackage(server.resource, getName(server.resource))
	value = EmptyDirectoryPath
	pkg.Members.Files = files
	for _, f := range files {
		f.Created = pkg.Created
		f.Modified = pkg.Modified
		key := []string{f.Title}
		rpc.Set(key, f)
		err = setResource(key, f, txn)
		if err != nil {
			return
		}

		value, err = server.api.Object().AddLink(ctx, value, f.Title, f.Path())
		if err != nil {
			return
		}
	}

	err = server.setValue(ctx, pkg, value)
	if err != nil {
		return
	}

	id, err = server.normalize(ctx, pkg)
	if err != nil {
		return
	}

	id = pkg.Path()
	rpc.Set(nil, pkg)
	err = setResource(nil, pkg, txn)
	return
}

func (server *Server) update(ctx context.Context, id, value path.Resolved) (err error) {
	err = server.api.Pin().Update(ctx, server.id, id)
	if err != nil {
		return err
	}

	err = server.api.Pin().Update(ctx, server.value, value)
	if err != nil {
		return err
	}

	return nil
}

// commit is called *after* the resource at key has been written.
// r is either a *Assertion, *File, or *Package - i.e. NOT just a *Reference.
// Pass a *Reference with ID == "" if you want to delete a package
func (server *Server) commit(ctx context.Context, timestamp string, key []string, r types.Resource, txn *badger.Txn) (err error) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	var value path.Resolved
	if p, is := r.(*types.Package); is && p != nil {
		r = p.CopyResource()
		value = p.ValuePath()

		if len(key) == 0 {
			return server.update(ctx, p.Path(), value)
		}
	}

	var id path.Resolved
	for i := len(key) - 1; i >= 0; i-- {
		parentKey, name := key[:i], key[i]
		parent, err := getPackage(parentKey, txn)
		if err != nil {
			return err
		}

		id, value, err = server.prclt(ctx, timestamp, parentKey, parent, name, r, value)
		if err != nil {
			return err
		}

		err = setResource(parentKey, parent, txn)
		if err != nil {
			return err
		}

		r = parent.CopyResource()
	}

	// now parent is the root package.
	err = server.api.Pin().Update(ctx, server.id, id)
	if err != nil {
		return err
	}

	server.id = id
	return nil
}

var cidPattern = regexp.MustCompile(`^[a-z2-7]{59}$`)

func (server *Server) prclt(
	ctx context.Context,
	timestamp string,
	key []string,
	pkg *types.Package,
	name string,
	r types.Resource,
	value path.Resolved,
) (id, nextValue path.Resolved, err error) {
	object := server.api.Object()
	nextValue = pkg.ValuePath()

	isCid := cidPattern.MatchString(name)
	t := deletePackageMember(pkg, name, isCid)
	switch t {
	case types.PackageType:
		nextValue, err = object.RmLink(ctx, nextValue, name)
		if err != nil {
			return
		}
		nextValue, err = object.RmLink(ctx, nextValue, name+types.NQuadsFileExtension)
		if err != nil {
			return
		}
	case types.AssertionType:
		nextValue, err = object.RmLink(ctx, nextValue, name+types.NQuadsFileExtension)
		if err != nil {
			return
		}
	case types.FileType:
		nextValue, err = object.RmLink(ctx, nextValue, name)
		if err != nil {
			return
		}
	}

	if r != nil {
		id = r.Path()

		switch r := r.(type) {
		case *types.Reference:
			i, old := pkg.SearchPackages(name, isCid)
			if old == nil {
				pkg.Members.Packages = append(pkg.Members.Packages, nil)
				copy(pkg.Members.Packages[i+1:], pkg.Members.Packages[i:])
			}
			pkg.Members.Packages[i] = r

			nextValue, err = object.AddLink(ctx, nextValue, name, value)
			if err != nil {
				return
			}

			nextValue, err = object.AddLink(ctx, nextValue, name+types.NQuadsFileExtension, id)
			if err != nil {
				return
			}
		case *types.Assertion:
			i, old := pkg.SearchAssertions(name, isCid)
			if old == nil {
				pkg.Members.Assertions = append(pkg.Members.Assertions, nil)
				copy(pkg.Members.Assertions[i+1:], pkg.Members.Assertions[i:])
			}
			pkg.Members.Assertions[i] = r

			nextValue, err = object.AddLink(ctx, nextValue, name+types.NQuadsFileExtension, id)
			if err != nil {
				return
			}
		case *types.File:
			i, old := pkg.SearchFiles(name, isCid)
			if old == nil {
				pkg.Members.Files = append(pkg.Members.Files, nil)
				copy(pkg.Members.Files[i+1:], pkg.Members.Files[i:])
			}
			pkg.Members.Files[i] = r

			nextValue, err = object.AddLink(ctx, nextValue, name, id)
			if err != nil {
				return
			}
		}
	}

	pkg.Modified = timestamp
	pkg.Parent = pkg.ID
	err = server.setValue(ctx, pkg, nextValue)
	if err != nil {
		return
	}

	id, err = server.normalize(ctx, pkg)
	if err != nil {
		return
	}
	return
}

func (server *Server) setValue(ctx context.Context, pkg *types.Package, value path.Resolved) error {
	stat, err := server.api.Object().Stat(ctx, value)
	if err != nil {
		return err
	}
	pkg.Value.Extent = stat.CumulativeSize
	s, err := stat.Cid.StringOfBase(multibase.Base32)
	if err != nil {
		return err
	}
	pkg.Value.ID = "dweb:/ipfs/" + s
	return nil
}

func (server *Server) setFile(ctx context.Context, f *types.File, node files.File) error {
	resolved, err := server.api.Unixfs().Add(ctx, node, addOpts...)
	if err != nil {
		return err
	}

	stat, err := server.api.Object().Stat(ctx, resolved)
	if err != nil {
		return err
	}

	s, err := stat.Cid.StringOfBase(multibase.Base32)
	if err != nil {
		return err
	}

	f.ID = "dweb:/ipfs/" + s
	f.Extent = stat.CumulativeSize
	return nil
}

func (server *Server) setAssertion(ctx context.Context, a *types.Assertion, node files.File) error {
	id, err := server.api.Unixfs().Add(ctx, node, addOpts...)
	if err != nil {
		return err
	}
	s, err := id.Cid().StringOfBase(multibase.Base32)
	if err != nil {
		return err
	}
	a.ID = "ul:" + s
	return nil
}

// normalize adds the normalized n-quads string to IPFS and it sets pkg.ID; nothing else.
func (server *Server) normalize(ctx context.Context, pkg *types.Package) (path.Resolved, error) {
	pkg.ID = ""

	doc, err := pkg.JsonLd(links["package.jsonld"])
	if err != nil {
		return nil, err
	}

	proc := ld.NewJsonLdProcessor()
	opts := ld.NewJsonLdOptions("")
	opts.DocumentLoader = server.documentLoader
	dataset, err := proc.ToRDF(doc, opts)
	if err != nil {
		return nil, err
	}

	opts.Format = "application/n-quads"
	na := ld.NewNormalisationAlgorithm("URDNA2015")
	normalized, err := na.Main(dataset.(*ld.RDFDataset), opts)
	if err != nil {
		return nil, err
	}

	data := []byte(normalized.(string))
	id, err := server.api.Unixfs().Add(ctx, files.NewBytesFile(data), addOpts...)
	if err != nil {
		return nil, err
	}

	value := ld.NewIRI(pkg.Value.ID)

	var fragment string

	for _, quad := range na.Quads() {
		if ld.IsBlankNode(quad.Subject) &&
			quad.Predicate.GetValue() == provValue.Value() &&
			quad.Object.Equal(value) {
			fragment = strings.TrimPrefix(quad.Subject.GetValue(), "_:")
			break
		}
	}

	s, err := id.Cid().StringOfBase(multibase.Base32)
	if err != nil {
		return nil, err
	}

	pkg.ID = "ul:" + s + "#" + fragment
	return id, nil
}
