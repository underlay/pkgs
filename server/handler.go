package server

import (
	"context"
	"net/http"
	"regexp"

	badger "github.com/dgraph-io/badger/v2"
	core "github.com/ipfs/interface-go-ipfs-core"
	ld "github.com/piprate/json-gold/ld"
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

func Handler(res http.ResponseWriter, req *http.Request, db *badger.DB, api core.CoreAPI) {
	var err error
	ctx := context.TODO()
	if req.Method == "GET" {
		err = Get(ctx, res, req, db, api)
	} else if req.Method == "PUT" {
		err = Put(ctx, res, req, db, api)
	} else if req.Method == "HEAD" {
		err = Head(ctx, res, req, db, api)
	} else if req.Method == "DELETE" {
	} else if req.Method == "OPTIONS" {
	}

	if err != nil {
		res.Write([]byte(err.Error()))
	}
	return
}
