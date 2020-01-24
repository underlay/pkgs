package query

import ld "github.com/underlay/json-gold/ld"

// A Cursor is an interactive query interface
type Cursor interface {
	Len() int
	Graph() []*ld.Quad
	Get(node *ld.BlankNode) ld.Node
	Domain() []*ld.BlankNode
	Index() []ld.Node
	Next(node *ld.BlankNode) ([]*ld.BlankNode, error)
	Seek(index []ld.Node) error
	Close()
}
