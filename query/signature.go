package query

import ld "github.com/piprate/json-gold/ld"

// A Signature is what database indices expose to support querying
type Signature interface {
	Head() []*ld.Quad
	Domain() []*ld.BlankNode
	Query(query []*ld.Quad, assignments map[string]ld.Node, domain []*ld.BlankNode, index []ld.Node) (Cursor, error)
}
