package types

import ld "github.com/underlay/json-gold/ld"

// A Signature is what database indices expose to support querying
type Signature interface {
	Head() []*ld.Quad
	Domain() []string
	Query(query []*ld.Quad, domain []*ld.BlankNode, index []ld.Node) (Cursor, error)
}
