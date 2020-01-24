package indices

import (
	"log"

	bleve "github.com/blevesearch/bleve"
	ld "github.com/underlay/json-gold/ld"

	query "github.com/underlay/pkgs/query"
)

// NameIndex associates ids with common name and title properties
var _ query.Index = (*nameIndex)(nil)

type nameIndex struct {
	b bleve.Index
}

// Choose a set of bag sizes, more is more accurate but slower
var bagSizes = []int{2, 3, 4}

var filename = "FILENAME"

func (ni *nameIndex) Open() {
	var err error
	mapping := bleve.NewIndexMapping()
	ni.b, err = bleve.New(filename, mapping)
	if err != nil {
		log.Fatal(err)
	}
}

func (ni *nameIndex) Close() { _ = ni.b.Close() }
func (ni *nameIndex) Add(pathname []string, resource query.Resource) {
	_, id := resource.ETag()
	ni.b.Index(id, resourceToData(resource))
}

func (ni *nameIndex) Remove(pathname []string, resource query.Resource) {
	_, id := resource.ETag()
	ni.b.Delete(id)
}

func (ni *nameIndex) Signatures() []query.Signature {
	return []query.Signature{ni} // why not lmao
}

func (ni *nameIndex) Head() []*ld.Quad {
	return nil
}

func (ni *nameIndex) Domain() []*ld.BlankNode {
	return nil
}

func (ni *nameIndex) Query(
	query []*ld.Quad,
	assignments map[string]ld.Node,
	domain []*ld.BlankNode,
	index []ld.Node,
) (query.Cursor, error) {
	return nil, nil
}

func resourceToData(resource query.Resource) interface{} {
	return nil
}
