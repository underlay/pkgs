package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	badger "github.com/dgraph-io/badger/v2"
	files "github.com/ipfs/go-ipfs-files"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	content "github.com/joeltg/negotiate/content"

	types "github.com/underlay/pkgs/types"
	ui "github.com/underlay/pkgs/ui"
)

// var defaultContentType = "application/n-quads"

var offers = map[types.ResourceType][]string{
	types.PackageType: []string{"application/n-quads", "application/ld+json", "text/html"},
	types.MessageType: []string{"application/n-quads", "application/ld+json"},
	types.FileType:    []string{},
}

// Get handles HTTP GET requests
func (server *Server) Get(ctx context.Context, res http.ResponseWriter, req *http.Request) error {
	ifNoneMatch := req.Header.Get("If-None-Match")

	pathname := req.URL.Path

	if pathname != "/" && !PathRegex.MatchString(pathname) {
		res.WriteHeader(404)
		return nil
	}

	var name string
	if pathname == "/" {
		name = "index"
	} else {
		index := strings.LastIndex(pathname, "/")
		name = fmt.Sprintf(pathname[index+1:])
	}

	var contentType string
	var resource types.Resource
	var u types.ResourceType
	var page *ui.Page
	err := server.db.View(func(txn *badger.Txn) (err error) {
		resource, u, err = types.GetResource(pathname, txn)
		defaultOffer := "application/n-quads"
		if f, is := resource.(*types.File); is && u == types.FileType {
			defaultOffer = f.Format
		}

		// It's a little awkward to render the HTML for the web ui here,
		// but it's the best way to do it
		contentType = content.NegotiateContentType(req, offers[u], defaultOffer)
		if p, is := resource.(*types.Package); is && contentType == "text/html" {
			page, err = ui.MakePage(pathname, p, txn)
		}
		return
	})

	if err == badger.ErrKeyNotFound {
		res.WriteHeader(404)
		return nil
	} else if err != nil {
		res.WriteHeader(500)
		return err
	}

	// It's important to check for more than contentType == "text/html" because
	// some files will have f.Format == "text/html"!
	if u == types.PackageType && contentType == "text/html" {
		res.Header().Add("Content-Type", contentType)
		err = ui.PageTemplate.Execute(res, page)
		if err != nil {
			log.Println(err)
		}
		return nil
	}

	res.Header().Add("Link", linkTypeResource)

	c, etag := resource.ETag()
	if ifNoneMatch == etag {
		res.WriteHeader(304)
		return nil
	}

	res.Header().Add("ETag", fmt.Sprintf("\"%s\"", etag))

	node, err := server.fs.Get(ctx, path.IpfsPath(c))
	if err != nil {
		res.WriteHeader(502)
		return err
	}

	file := files.ToFile(node)

	// Okay now we have a Resource and we get to respond with its representation
	switch t := resource.(type) {
	case *types.Package:
		contentDisposition := fmt.Sprintf("attachment; filename=%s.nt", name)
		res.Header().Add("Content-Disposition", contentDisposition)
		res.Header().Add("Link", linkTypeDirectContainer)
		res.Header().Add("Link", fmt.Sprintf(`<#%s>; rel="self"`, t.Subject))
		if contentType == "application/ld+json" {
			res.Header().Add("Content-Type", contentType)
			doc, err := server.proc.FromRDF(file, server.opts)
			if err != nil {
				res.WriteHeader(500)
				return err
			}

			frame := map[string]interface{}{
				"@context": types.ContextURL,
				"@type":    types.PackageIri.Value,
			}

			framed, err := server.proc.Frame(doc, frame, server.opts)
			if err != nil {
				res.WriteHeader(500)
				return err
			}

			framed["@context"] = types.ContextURL
			_ = json.NewEncoder(res).Encode(framed)
		} else if contentType == "application/n-quads" {
			res.Header().Add("Content-Type", contentType)
			_, _ = io.Copy(res, file)
		}
	case types.Message:
		contentDisposition := fmt.Sprintf("attachment; filename=%s.nt", name)
		res.Header().Add("Content-Disposition", contentDisposition)
		res.Header().Add("Link", linkTypeNonRDFSource)
		if contentType == "application/ld+json" {
			doc, err := server.proc.FromRDF(file, server.opts)
			if err != nil {
				res.WriteHeader(500)
				return err
			}

			res.Header().Add("Content-Type", contentType)
			_ = json.NewEncoder(res).Encode(doc)
		} else if contentType == "application/n-quads" {
			res.Header().Add("Content-Type", contentType)
			_, _ = io.Copy(res, file)
		}
	case *types.File:
		contentDisposition := fmt.Sprintf("attachment; filename=%s", name)
		res.Header().Add("Content-Disposition", contentDisposition)
		res.Header().Add("Link", linkTypeNonRDFSource)
		extent := strconv.FormatUint(t.Extent, 10)
		res.Header().Add("Content-Type", t.Format)
		res.Header().Add("Content-Length", extent)
		_, _ = io.Copy(res, file)
	}
	return nil
}
