package main

import (
	"bytes"
	"context"
	"net/http"
	"time"

	files "github.com/ipfs/go-ipfs-files"
	ld "github.com/piprate/json-gold/ld"
	rdf "github.com/underlay/go-rdfjs"
	types "github.com/underlay/pkgs/types"
)

// Post handles HTTP POST requests
func (server *Server) Post(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	self, t := types.ParseLinks(req.Header["Link"])
	parentKey := types.ParsePath(req.URL.Path)

	parentResource := types.GetURI(server.resource, parentKey)

	var r types.Resource
	var err error
	format := req.Header.Get("Content-Type")
	timestamp := time.Now().Format(time.RFC3339)

	switch t {
	case types.PackageType:
		res.WriteHeader(422)
		return
	case types.AssertionType:
		if format == "" && self != "" && types.AssertionURIPattern.MatchString(self) {
			a := &types.Assertion{ID: self, Created: timestamp}
			node, err := server.api.Unixfs().Get(ctx, a.Path())
			if err != nil {
				res.WriteHeader(502)
				res.Write([]byte(err.Error()))
				return
			}
			a.Dataset, err = rdf.ReadQuads(files.ToFile(node))
			if err != nil {
				res.WriteHeader(400)
				res.Write([]byte(err.Error()))
				return
			}

			r = a
		} else if format == offers[0] || format == offers[1] || format == offers[2] {
			dataset, err := parseDataset(format, parentResource, req.Body)
			if err != nil {
				res.WriteHeader(400)
				res.Write([]byte(err.Error()))
				return
			}

			opts := ld.NewJsonLdOptions(parentResource)
			opts.Format = "application/n-quads"
			na := ld.NewNormalisationAlgorithm("URDNA2015")
			normalised, err := na.Main(dataset, opts)
			if err != nil {
				res.WriteHeader(400)
				res.Write([]byte(err.Error()))
				return
			}

			data := []byte(normalised.(string))
			file := files.NewBytesFile(data)

			a := &types.Assertion{Created: timestamp}
			err = server.setAssertion(ctx, a, file)
			if err != nil {
				res.WriteHeader(502)
				res.Write([]byte(err.Error()))
				return
			}

			a.Dataset, err = rdf.ReadQuads(bytes.NewReader(data))
			if err != nil {
				res.WriteHeader(400)
				res.Write([]byte(err.Error()))
				return
			}

			r = a
		} else {
			res.WriteHeader(415)
			return
		}
	case types.FileType:
		if format == "" && self != "" && types.FileURIPattern.MatchString(self) {
			f := &types.File{ID: self, Created: timestamp, Format: format}
			stat, err := server.api.Object().Stat(ctx, f.Path())
			if err != nil {
				res.WriteHeader(502)
				res.Write([]byte(err.Error()))
				return
			}
			f.Extent = stat.CumulativeSize
			r = f
		} else if format != "" {
			f := &types.File{Created: timestamp, Format: format}
			err := server.setFile(ctx, f, files.NewReaderFile(req.Body))
			if err != nil {
				res.WriteHeader(502)
				return
			}
			r = f
		} else {
			res.WriteHeader(415)
			return
		}
	default:
		res.WriteHeader(400)
		return
	}

	txn := server.db.NewTransaction(true)
	defer txn.Discard()

	key := append(parentKey, r.Name())
	err = server.set(ctx, key, r, txn)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(err.Error()))
		return
	}

	err = server.commit(ctx, timestamp, key, r, txn)
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(err.Error()))
		return
	}

	err = txn.Commit()
	if err != nil {
		res.WriteHeader(500)
		res.Write([]byte(err.Error()))
		return
	}

	res.Header().Add("ETag", r.ETag())
	res.Header().Add("Link", makeSelfLink(r.URI()))
	res.WriteHeader(201)
}
