package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	files "github.com/ipfs/go-ipfs-files"
	options "github.com/ipfs/interface-go-ipfs-core/options"
	path "github.com/ipfs/interface-go-ipfs-core/path"

	types "github.com/underlay/pkgs/types"
)

// Put handles HTTP PUT requests
func (server *Server) Put(ctx context.Context, res http.ResponseWriter, req *http.Request) error {
	contentType := req.Header.Get("Content-Type")
	if contentType == "" || len(req.Header["Content-Type"]) != 1 {
		// Content-Type is required for all requests.
		res.WriteHeader(400)
		return nil
	}

	linkType := req.Header.Get("Link")

	if linkType == "" {
		// No link type found
		res.WriteHeader(400)
		return nil
	}

	if _, has := linkTypes[linkType]; !has {
		res.WriteHeader(422)
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

	if !PathRegex.MatchString(pathname) {
		res.WriteHeader(404)
		return nil
	}

	ifMatch := req.Header.Get("If-Match")
	var match string
	if ifMatch != "" {
		if etagRegex.MatchString(ifMatch) {
			match = etagRegex.FindStringSubmatch(ifMatch)[1]
		} else {
			res.WriteHeader(412)
			return nil
		}
	}

	var parentPath string
	tail := strings.LastIndex(pathname, "/")
	if tail > 0 {
		parentPath = pathname[:tail]
	} else {
		parentPath = "/"
		tail = 0
	}

	name := pathname[tail+1:]

	// Acquire lock
	var lock *sync.Mutex
	var has bool
	if lock, has = server.locks[pathname]; !has {
		lock = &sync.Mutex{}
		server.locks[pathname] = lock
	}
	lock.Lock()
	defer lock.Unlock()
	defer delete(server.locks, pathname)

	// time.Sleep(time.Second * 10)

	return server.db.Update(func(txn *badger.Txn) error {
		parent, err := types.GetPackage(parentPath, txn)
		if err == badger.ErrKeyNotFound {
			// Parent doesn't exist!
			res.WriteHeader(404)
			return nil
		} else if err == types.ErrNotPackage {
			res.WriteHeader(409)
			return nil
		} else if err != nil {
			res.WriteHeader(500)
			return err
		}

		parentID, parentValue, err := parent.Paths()
		if err != nil {
			return err
		}

		var leaf path.Resolved
		var mutation bool
		var etag string
		var value types.Resource

		for _, member := range parent.Member {
			if member == name {
				mutation = true
				break
			}
		}

		if mutation {
			// The resource already exists!
			resource, _, err := types.GetResource(pathname, txn)
			if err != nil {
				res.WriteHeader(500)
				return err
			}

			// For now we can at least check the If-Match tag
			_, etag := resource.ETag()
			if err != nil {
				res.WriteHeader(500)
				return err
			}

			if etag != match {
				res.WriteHeader(412)
				return nil
			}

			res.WriteHeader(501)
			return nil
		}

		// Okay so parent is an existing package and this is a new
		// route beneath it, with link type linkType.

		// It's safe to start mutating p because it we encouter
		// errors we'll return before we write it back to the database
		parent.Member = append(parent.Member, name)

		if linkType == linkTypeNonRDFSource {
			// New file!
			leaf, err = server.fs.Add(
				ctx,
				files.NewReaderFile(req.Body),
				options.Unixfs.Pin(false),
				options.Unixfs.RawLeaves(true),
				options.Unixfs.CidVersion(1),
			)

			if err != nil {
				res.WriteHeader(502)
				return err
			}

			stat, err := server.object.Stat(ctx, leaf)
			if err != nil {
				res.WriteHeader(502)
				return err
			}

			file := &types.File{
				Value:  leaf.Cid().Bytes(),
				Format: contentType,
				Extent: uint64(stat.CumulativeSize),
			}

			value = file
		} else if linkType == linkTypeRDFSource {
			// New message!
			var doc interface{}
			if contentType == "application/ld+json" {
				doc = req.Body
			} else if contentType == "application/n-quads" {
				doc, err = server.proc.FromRDF(req.Body, server.opts)
				if err != nil {
					res.WriteHeader(400)
					return err
				}
			}

			n, err := server.proc.Normalize(doc, server.opts)
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

			leaf, err = server.fs.Add(
				ctx,
				files.NewReaderFile(reader),
				options.Unixfs.Pin(false),
				options.Unixfs.RawLeaves(true),
				options.Unixfs.CidVersion(1),
			)

			if err != nil {
				res.WriteHeader(502)
				return err
			}

			value = types.Message(leaf.Cid().Bytes())
		} else if linkType == linkTypeDirectContainer {
			// New subpackage!
			// ... we'll implement this later :-/
			res.WriteHeader(501)
			return nil
		}

		// newResource := types.NewResource(value)

		_, etag = value.ETag()
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		err = types.SetResource(value, pathname, txn)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		nextValue, err := server.object.AddLink(ctx, parentValue, name, leaf)

		err = server.percolate(
			ctx,
			time.Now(),
			parentPath,
			parent,
			parentID, parentValue,
			nextValue,
			txn,
		)

		if err != nil {
			res.WriteHeader(500)
			return err
		}

		res.Header().Add("ETag", fmt.Sprintf("\"%s\"", etag))
		res.Header().Add("Access-Control-Allow-Origin", "http://localhost:8000")
		res.Header().Add("Access-Control-Allow-Methods", "GET, HEAD, PUT, DELETE")
		res.Header().Add("Access-Control-Allow-Headers", "Content-Type, Accept, Link, If-Match")
		res.Header().Add("Access-Control-Expose-Headers", "ETag")
		if mutation {
			res.WriteHeader(200)
		} else {
			res.WriteHeader(201)
		}

		res.Write(nil)
		return nil
	})
}
