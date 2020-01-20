package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	content "github.com/joeltg/negotiate/content"
	ld "github.com/underlay/json-gold/ld"

	types "github.com/underlay/pkgs/types"
	"github.com/underlay/pkgs/vocab"
)

var linkHeader = regexp.MustCompile("^<#(_:c14n\\d+)>; rel=\"self\"$")

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

	defaultOffer := "application/n-quads"
	accept := content.NegotiateContentType(req, []string{defaultOffer, "application/ld+json"}, defaultOffer)

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

	u := p.URI()
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
	buf := new(bytes.Buffer)
	buf.ReadFrom(req.Body)
	k := buf.String()
	log.Println(k)
	a := strings.NewReader(k)
	quads, err := server.parseDataset(a, contentType)
	if err != nil {
		res.WriteHeader(400)
		return err
	} else if quads == nil {
		res.WriteHeader(415)
		return nil
	}

	subjects := []string{}
	var descriptionChanged bool
	for _, quad := range quads {
		if quad.Subject.GetValue() != subject {
			res.WriteHeader(400)
			return nil
		} else if literal, is := quad.Object.(*ld.Literal); is && literal.Datatype == ld.XSDString {
			if quad.Predicate.Equal(vocab.DCTERMSdescription) {
				descriptionChanged = true
				p.Description = literal.Value
			} else if quad.Predicate.Equal(vocab.DCTERMSsubject) {
				subjects = append(subjects, literal.Value)
			}
		} else {
			res.WriteHeader(400)
			return nil
		}
	}

	if len(subjects) > 0 {
		p.Keyword = subjects
	}

	if !descriptionChanged && len(subjects) == 0 {
		res.Header().Add("ETag", fmt.Sprintf("\"%s\"", etag))
		res.WriteHeader(204)
		return nil
	}

	id, value, err := p.Paths()
	if err != nil {
		res.WriteHeader(400)
		return err
	}
	t := time.Now()
	err = server.db.Update(func(txn *badger.Txn) error {
		return server.percolate(ctx, t, pathname, p, id, value, nil, txn)
	})
	if err != nil {
		res.WriteHeader(400)
		return err
	}

	_, etag = p.ETag()
	res.Header().Add("ETag", fmt.Sprintf("\"%s\"", etag))
	res.Header().Add("Link", `<#_:c14n0>; rel="self"`)
	res.Header().Add("Content-Type", accept)
	res.WriteHeader(200)
	m := t.Format(time.RFC3339)
	if accept == defaultOffer {
		rdf := ld.NewRDFDataset()
		s := ld.NewBlankNode("_:c14n0")
		rdf.Graphs["@default"] = append(
			rdf.Graphs["@default"],
			ld.NewQuad(s, vocab.DCTERMSmodified, ld.NewLiteral(m, vocab.XSDdateTime, ""), ""),
			ld.NewQuad(s, vocab.PROVwasRevisionOf, ld.NewIRI(u), ""),
		)
		if descriptionChanged {
			rdf.Graphs["@default"] = append(
				rdf.Graphs["@default"],
				ld.NewQuad(s, vocab.DCTERMSdescription, ld.NewLiteral(p.Description, "", ""), ""),
			)
		}
		if len(subjects) > 0 {
			subjectQuads := make([]*ld.Quad, len(subjects))
			for i, subject := range subjects {
				subjectQuads[i] = ld.NewQuad(s, vocab.DCTERMSsubject, ld.NewLiteral(subject, "", ""), "")
			}
			rdf.Graphs["@default"] = append(
				rdf.Graphs["@default"],
				subjectQuads...,
			)
		}
		ns := &ld.NQuadRDFSerializer{}
		_ = ns.SerializeTo(res, rdf)
	} else {
		patch := map[string]interface{}{
			vocab.PROVwasRevisionOf.Value: map[string]interface{}{"@id": u},
			vocab.DCTERMSmodified.Value: map[string]interface{}{
				"@value": m,
				"@type":  vocab.XSDdateTime,
			},
		}
		if descriptionChanged {
			patch[vocab.DCTERMSdescription.Value] = p.Description
		}
		if len(subjects) > 0 {
			patch[vocab.DCTERMSsubject.Value] = subjects
		}
		json.NewEncoder(res).Encode(patch)
	}
	return nil
}
