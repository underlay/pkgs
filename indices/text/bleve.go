package text

import (
	"log"
	"strings"

	bleve "github.com/blevesearch/bleve"
	badger "github.com/dgraph-io/badger/v2"
	iface "github.com/ipfs/interface-go-ipfs-core"
	rdf "github.com/underlay/go-rdfjs"

	indices "github.com/underlay/pkgs/indices"
	types "github.com/underlay/pkgs/types"
	styx "github.com/underlay/styx"
)

// MatchPredicate is the text matching predicate
var MatchPredicate = "pkgs:/text/match"

type textIndex struct {
	bleve.Index
}

// NewTextIndex creates a new text index
func NewTextIndex() indices.GeneratorIndex { return &textIndex{} }

func (ti *textIndex) Name() string { return "text" }

func (ti *textIndex) Init(resource string, api iface.CoreAPI, db *badger.DB, path string) {
	log.Println("Initializing text index", path)
	index, err := bleve.Open(path)
	if err == bleve.ErrorIndexMetaMissing {
		log.Println("Creating new text index at", path)
		mapping := getMapping()
		index, err = bleve.New(path, mapping)
		if err != nil {
			log.Println(err)
			return
		}
	} else if err != nil {
		log.Println(err)
		return
	}

	ti.Index = index
}

func (ti *textIndex) Close() {
	if ti.Index == nil {
		return
	}

	ti.Index.Close()
}

func (ti *textIndex) Set(key []string, resource types.Resource, dataset []*rdf.Quad, store *styx.Store) error {
	if ti.Index == nil {
		return nil
	}

	id := "/" + strings.Join(key, "/")
	err := ti.Index.Index(id, resource)
	if err != nil {
		return err
	}

	return nil
}

func (ti *textIndex) Delete(key []string, resource types.Resource, dataset []*rdf.Quad, store *styx.Store) error {
	if ti.Index == nil {
		return nil
	}

	id := "/" + strings.Join(key, "/")
	err := ti.Index.Delete(id)
	if err != nil {
		return err
	}

	return nil
}

var subject = rdf.NewVariable("subject")
var predicate = rdf.NewNamedNode(MatchPredicate)
var object = rdf.NewVariable("object")
var title = rdf.NewNamedNode("http://purl.org/dc/terms/title")
var head = rdf.NewQuad(subject, predicate, object, nil)
var body = rdf.NewQuad(subject, title, object, nil)

func (ti *textIndex) Head() []*rdf.Quad { return []*rdf.Quad{head} }
func (ti *textIndex) Base() []rdf.Term  { return []rdf.Term{object} }
func (ti *textIndex) Body() []*rdf.Quad { return []*rdf.Quad{body} }
func (ti *textIndex) Query(
	query []*rdf.Quad,
	domain, index []rdf.Term,
) (indices.Iterator, error) {
	iter := &textIterator{textIndex: ti}
	if len(index) > 0 && len(domain) > 0 && domain[0].Equal(object) {
		iter.Seek(index)
	}
	return iter, nil
}

type textIterator struct {
	*textIndex
	object rdf.Term
	index  int
	result *bleve.SearchResult
}

func (iter *textIterator) value() *rdf.NamedNode {
	if iter.result != nil && 0 <= iter.index && iter.index < iter.result.Hits.Len() {
		hit := iter.result.Hits[iter.index]
		id, has := hit.Fields["id"]
		if has {
			return rdf.NewNamedNode(id.(string))
		}
	}
	return nil
}

func (iter *textIterator) Get(node rdf.Term) rdf.Term {
	if node.Equal(object) {
		return iter.object
	} else if node.Equal(subject) {
		return iter.value()
	} else {
		return nil
	}
}

func (iter *textIterator) Domain() []rdf.Term { return []rdf.Term{object, subject} }
func (iter *textIterator) Index() []rdf.Term {
	value := iter.value()
	if value == nil {
		return nil
	}
	return []rdf.Term{iter.object, value}
}

func (iter *textIterator) Next(node rdf.Term) ([]rdf.Term, error) {
	if iter.result == nil {
		return nil, nil
	}

	if node == nil || node.Equal(subject) {
		iter.index++
		if iter.index < iter.result.Hits.Len() {
			value := iter.value()
			if iter.index == 0 {
				return []rdf.Term{iter.object, value}, nil
			}
			return []rdf.Term{value}, nil
		}
	} else if node.Equal(object) {
		iter.index = iter.result.Hits.Len()
		return nil, nil
	}

	return nil, nil
}

func (iter *textIterator) Seek(index []rdf.Term) error {
	iter.index = -1
	if len(index) == 0 {
		iter.object = nil
		iter.result = nil
		return nil
	}

	value := index[0]
	if iter.object != nil && value.Equal(iter.object) {
		if len(index) == 1 {
			return nil
		}
	} else if value.TermType() != rdf.LiteralType {
		iter.object = nil
		iter.result = nil
		return nil
	} else {
		query := bleve.NewMatchQuery(value.Value())
		query.Fuzziness = 2
		search := bleve.NewSearchRequest(query)
		search.Fields = []string{"id"}
		result, err := iter.textIndex.Search(search)
		if err != nil {
			return err
		}

		iter.object = value
		iter.result = result
	}

	if len(index) == 2 {
		for i, hit := range iter.result.Hits {
			if hit.Fields["id"] == index[2].Value() {
				iter.index = i
				return nil
			}
		}
		iter.index = len(iter.result.Hits)
	}

	return nil
}

func (iter *textIterator) Prov() ([][]rdf.Term, error) { return nil, nil }
func (iter *textIterator) Close()                      {}
