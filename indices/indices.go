package indices

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	badger "github.com/dgraph-io/badger/v2"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	rdf "github.com/underlay/go-rdfjs"
	types "github.com/underlay/pkgs/types"
)

// An Index is the interface for database indices
type Index interface {
	Name() string
	Init(api coreiface.CoreAPI, db *badger.DB, path string)
	Close()
	Set(key []string, resource types.Resource)
	Delete(key []string, resource types.Resource)
	Signatures() []Signature
}

// A Signature is what database indices expose to support querying
type Signature interface {
	Head() []*rdf.Quad
	Base() []rdf.Term
	Query(
		query []*rdf.Quad,
		domain []rdf.Term,
		index []rdf.Term,
	) (Iterator, error)
}

// An Iterator is an interactive query interface
type Iterator interface {
	Get(node rdf.Term) rdf.Term
	Domain() []rdf.Term
	Index() []rdf.Term
	Next(node rdf.Term) ([]rdf.Term, error)
	Seek(index []rdf.Term) error
	Close()
}

func LogIterator(iter Iterator) {
	domain := iter.Domain()
	values := make([]string, len(domain))
	for i, node := range domain {
		values[i] = node.String()
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, strings.Join(values, "\t"))
	for d, err := iter.Next(nil); d != nil; d, err = iter.Next(nil) {
		if err != nil {
			return
		}

		values := make([]string, len(domain))
		start := len(domain) - len(d)
		for i, node := range d {
			values[start+i] = node.String()
		}
		fmt.Fprintln(w, strings.Join(values, "\t"))
	}
	_ = w.Flush()
}
