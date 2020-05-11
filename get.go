package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"

	badger "github.com/dgraph-io/badger/v2"
	files "github.com/ipfs/go-ipfs-files"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	content "github.com/joeltg/negotiate/content"
	ld "github.com/piprate/json-gold/ld"
	rdf "github.com/underlay/go-rdfjs"
	types "github.com/underlay/pkgs/types"
	ui "github.com/underlay/pkgs/ui"
)

var offers = []string{"application/n-quads", "application/ld+json", "application/json"}

// Get handles HTTP GET requests
func (server *Server) Get(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	key := types.ParsePath(req.URL.Path)
	txn := server.db.NewTransaction(false)
	defer txn.Discard()
	r, err := getResource(key, txn)
	if err == badger.ErrKeyNotFound {
		res.WriteHeader(404)
		return
	}

	res.Header().Add("ETag", r.ETag())
	res.Header().Add("Link", makeSelfLink(r.URI()))
	res.Header().Add("Link", types.MakeLinkType(types.LDPResource))
	res.Header().Add("Link", types.MakeLinkType(r.Type()))
	switch r := r.(type) {
	case *types.Package:
		format := content.NegotiateContentType(req, append(offers, "text/html"), offers[0])
		res.Header().Add("Content-Type", format)
		switch format {
		case offers[0]:
			server.copyFile(ctx, res, r.Path())
		case offers[1]:
			doc, err := r.JsonLd(links["package.jsonld"])
			if err != nil {
				res.WriteHeader(500)
				return
			}
			err = json.NewEncoder(res).Encode(doc)
		case offers[2]:
			server.writeRDFJS(ctx, res, r.Path())
		case "text/html":
			res.WriteHeader(200)
			err = ui.PageTemplate.Execute(res, &struct {
				Pkg *types.Package
				Key []string
			}{r, key})
			return
		}
	case *types.Assertion:
		format := content.NegotiateContentType(req, offers, offers[0])
		res.Header().Add("Content-Type", format)
		switch format {
		case offers[0]:
			server.copyFile(ctx, res, r.Path())
		case offers[1]:
			node, err := server.api.Unixfs().Get(ctx, r.Path())
			if err != nil {
				res.WriteHeader(502)
				return
			}

			ns := &ld.NQuadRDFSerializer{}
			dataset, err := ns.Parse(files.ToFile(node))
			if err != nil {
				res.WriteHeader(500)
				return
			}

			opts := ld.NewJsonLdOptions(r.Resource)
			expanded, err := ld.NewJsonLdApi().FromRDF(dataset, opts)
			if err != nil {
				res.WriteHeader(500)
				return
			}

			opts.DocumentLoader = server.documentLoader
			opts.OmitGraph = true
			compacted, err := ld.NewJsonLdProcessor().Compact(expanded, links["context.jsonld"], opts)
			if err != nil {
				res.WriteHeader(500)
				return
			}
			err = json.NewEncoder(res).Encode(compacted)
		case offers[2]:
			server.writeRDFJS(ctx, res, r.Path())
		}
	case *types.File:
		res.Header().Add("Content-Type", r.Format)
		res.WriteHeader(200)
		server.copyFile(ctx, res, r.Path())
	}
}

func (server *Server) copyFile(ctx context.Context, res http.ResponseWriter, id path.Resolved) {
	node, err := server.api.Unixfs().Get(ctx, id)
	if err != nil {
		res.WriteHeader(502)
		return
	}

	res.WriteHeader(200)
	_, err = io.Copy(res, files.ToFile(node))
}

func (server *Server) writeRDFJS(ctx context.Context, res http.ResponseWriter, id path.Resolved) {
	node, err := server.api.Unixfs().Get(ctx, id)
	if err != nil {
		res.WriteHeader(502)
		return
	}
	res.WriteHeader(200)
	scanner := bufio.NewScanner(files.ToFile(node))
	res.Write([]byte{'['})
	for limit := false; scanner.Scan(); {
		if limit {
			res.Write([]byte{','})
		} else {
			limit = true
		}

		quad := rdf.ParseQuad(scanner.Text())
		b, _ := quad.MarshalJSON()
		res.Write(b)
	}
	res.Write([]byte{']', '\n'})
}
