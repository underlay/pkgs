package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"

	files "github.com/ipfs/go-ipfs-files"
	ld "github.com/piprate/json-gold/ld"
	rdf "github.com/underlay/go-rdfjs"

	types "github.com/underlay/pkgs/types"
)

// ErrParsePackage means a remote package failed parsing
var ErrParsePackage = errors.New("Error parsing package")

var rdfType = rdf.NewNamedNode("http://www.w3.org/1999/02/22-rdf-syntax-ns#")
var ldpDirectContainer = rdf.NewNamedNode("http://www.w3.org/ns/ldp#DirectContainer")
var ldpRDFSource = rdf.NewNamedNode("http://www.w3.org/ns/ldp#RDFSource")
var ldpNonRDFSource = rdf.NewNamedNode("http://www.w3.org/ns/ldp#NonRDFSource")
var ldpMembershipResource = rdf.NewNamedNode("http://www.w3.org/ns/ldp#membershipResource")

var provHadMember = rdf.NewNamedNode("http://www.w3.org/ns/prov#hadMember")
var provValue = rdf.NewNamedNode("http://www.w3.org/ns/prov#value")

var dctermsTitle = rdf.NewNamedNode("http://purl.org/dc/terms/title")
var dctermsCreated = rdf.NewNamedNode("http://purl.org/dc/terms/created")
var dctermsModified = rdf.NewNamedNode("http://purl.org/dc/terms/modified")
var dctermsFormat = rdf.NewNamedNode("http://purl.org/dc/terms/format")
var dctermsExtent = rdf.NewNamedNode("http://purl.org/dc/terms/extent")

var xsdDateTime = rdf.NewNamedNode("http://www.w3.org/2001/XMLSchema#dateTime")
var xsdInteger = rdf.NewNamedNode("http://www.w3.org/2001/XMLSchema#integer")

var initialFiles = [][3]string{}

func init() {
	wd, _ := os.Getwd()

	// It's actually important that these names are sorted here
	names := []string{"context", "package", "schema", "shex"}
	for _, name := range names {
		filename := name + ".jsonld"
		data, _ := ioutil.ReadFile(wd + "/files/" + filename)
		initial := [3]string{filename, "application/ld+json", string(data)}
		initialFiles = append(initialFiles, initial)
	}
}

// parse exactly one level
func (server *Server) parse(ctx context.Context, r *types.Reference) (*types.Package, error) {
	node, err := server.api.Unixfs().Get(ctx, r.Path())
	if err != nil {
		return nil, err
	}

	file := files.ToFile(node)
	opts := ld.NewJsonLdOptions(r.Resource)
	input, err := ld.NewJsonLdProcessor().FromRDF(file, opts)
	if err != nil {
		return nil, err
	}

	pkg, err := server.framePackage(r.Resource, input)
	pkg.ID = r.ID
	return pkg, err
}

func (server *Server) framePackage(base string, input interface{}) (*types.Package, error) {
	opts := ld.NewJsonLdOptions(base)
	opts.DocumentLoader = server.documentLoader
	framed, err := ld.NewJsonLdProcessor().Frame(input, links["package.jsonld"], opts)

	p := &types.Package{}

	// This is a little crazy but whatever
	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(framed)
	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(buffer).Decode(p)
	if err != nil {
		return nil, err
	}

	// TODO: sort the member resources!
	return p, nil
}
