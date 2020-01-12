package server

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	ld "github.com/underlay/json-gold/ld"

	types "github.com/underlay/pkgs/types"
)

var linkHeader = regexp.MustCompile("^<#(_:c14n\\d+)>; rel=\"self\"$")
var descriptionIri = ld.NewIRI("http://purl.org/dc/terms/description")
var subjectIri = ld.NewIRI("http://purl.org/dc/terms/subject")

// Patch handles HTTP PATCH requests
func (server *Server) Patch(ctx context.Context, res http.ResponseWriter, req *http.Request) error {
	pathname := req.URL.Path
	if pathname != "/" && !PathRegex.MatchString(pathname) {
		res.WriteHeader(404)
		return nil
	}

	link := req.Header.Get("Link")
	if !linkHeader.MatchString(link) {
		res.WriteHeader(400)
		return nil
	}

	subject := linkHeader.FindStringSubmatch(link)[1]

	// Should we require If-Match? I bet we should.
	ifMatch := req.Header.Get("If-Match")
	if !etagRegex.MatchString(ifMatch) {
		res.WriteHeader(412)
		return nil
	}

	match := etagRegex.FindStringSubmatch(ifMatch)[1]

	var p *types.Package
	err := server.db.View(func(txn *badger.Txn) (err error) {
		p, err = types.GetPackage(pathname, txn)
		return
	})

	if err == badger.ErrKeyNotFound {
		res.WriteHeader(404)
		return nil
	} else if err != nil {
		res.WriteHeader(500)
		return err
	}

	_, etag := p.ETag()
	if err != nil {
		res.WriteHeader(500)
		return err
	}

	if etag != match {
		res.WriteHeader(412)
		return nil
	}

	contentType := req.Header.Get("Content-Type")
	quads, err := server.parseDataset(req.Body, contentType)
	if err != nil {
		res.WriteHeader(400)
		return err
	} else if quads == nil {
		res.WriteHeader(415)
		return nil
	}

	subjects := []string{}
	var description string
	var changed bool
	for _, quad := range quads {
		if quad.Subject.GetValue() != subject {
			res.WriteHeader(400)
			return nil
		} else if literal, is := quad.Object.(*ld.Literal); is && literal.Datatype == ld.XSDString {
			if quad.Predicate.Equal(descriptionIri) {
				changed = true
				description = literal.Value
			} else if quad.Predicate.Equal(subjectIri) {
				subjects = append(subjects, literal.Value)
			}
		} else {
			res.WriteHeader(400)
			return nil
		}
	}
	if changed {
		p.Description = description
	}
	if len(subjects) > 0 {
		p.Keyword = subjects
		changed = true
	}
	if changed {
		id, value, err := p.Paths()
		if err != nil {
			res.WriteHeader(400)
			return err
		}
		err = server.db.Update(func(txn *badger.Txn) error {
			return server.percolate(ctx, time.Now(), pathname, p, id, value, nil, txn)
		})
		if err != nil {
			res.WriteHeader(400)
			return err
		}
	}

	_, etag = p.ETag()
	res.Header().Add("ETag", fmt.Sprintf("\"%s\"", etag))
	res.Header().Add("Access-Control-Allow-Origin", "http://localhost:8000")
	res.Header().Add("Access-Control-Allow-Methods", "GET, HEAD, POST, PATCH, DELETE")
	res.Header().Add("Access-Control-Allow-Headers", "Content-Type, Accept, Link, If-Match")
	res.Header().Add("Access-Control-Expose-Headers", "ETag")
	res.WriteHeader(204)

	return nil
}
