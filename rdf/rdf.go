package rdf

import (
	"encoding/json"

	ld "github.com/piprate/json-gold/ld"
)

type Term interface {
	TermType() string
	Value() string
	Equal(term Term) bool
}

type term struct {
	T string `json:"termType"`
	V string `json:"value"`
}

func (t *term) TermType() string {
	return t.T
}

func (t *term) Value() string {
	return t.V
}

func (t *term) Equal(term Term) bool {
	return term.TermType() == t.T && term.Value() == t.V
}

type literal struct {
	T string `json:"termType"`
	V string `json:"value"`
	L string `json:"language"`
	D *term  `json:"datatype,omitempty"`
}

func (l *literal) TermType() string {
	return l.T
}

func (l *literal) Value() string {
	return l.V
}

func (l *literal) Language() string {
	return l.L
}

func (l *literal) Datatype() *term {
	return l.D
}

func (l *literal) Equal(term Term) bool {
	if term, is := term.(*literal); is {
		return term.TermType() == l.T &&
			term.Value() == l.V &&
			term.Language() == l.L &&
			term.Datatype().Equal(l.D)
	}
	return false
}

type Quad struct {
	S json.RawMessage `json:"subject"`
	P json.RawMessage `json:"predicate"`
	O json.RawMessage `json:"object"`
	G json.RawMessage `json:"graph"`
	s Term
	p Term
	o Term
	g Term
}

func (q *Quad) Subject() Term {
	if q.s == nil {
		q.s, _ = UnmarshalTerm(q.S)
	}
	return q.s
}

func (q *Quad) Predicate() Term {
	if q.p == nil {
		q.p, _ = UnmarshalTerm(q.P)
	}
	return q.p
}

func (q *Quad) Object() Term {
	if q.o == nil {
		q.o, _ = UnmarshalTerm(q.O)
	}
	return q.o
}

func (q *Quad) Graph() Term {
	if q.g == nil {
		q.g, _ = UnmarshalTerm(q.G)
	}
	return q.g
}

func UnmarshalTerm(data []byte) (t Term, err error) {
	if data == nil {
		return
	}

	t = &term{}
	err = json.Unmarshal(data, t)
	if err != nil {
		return
	} else if t.TermType() == "Literal" {
		t = &literal{}
		err = json.Unmarshal(data, t)
	}
	return
}

func FromTerm(term Term) ld.Node {
	termType := term.TermType()
	if l, is := term.(*literal); is && termType == "Literal" {
		return ld.NewLiteral(l.V, l.D.V, l.L)
	} else if termType == "NamedNode" {
		return ld.NewIRI(term.Value())
	} else if termType == "BlankNode" || termType == "DefaultGraph" {
		return ld.NewBlankNode(term.Value())
	}
	return nil
}

func FromQuad(q *Quad) *ld.Quad {
	return &ld.Quad{
		Subject:   FromTerm(q.Subject()),
		Predicate: FromTerm(q.Predicate()),
		Object:    FromTerm(q.Object()),
		Graph:     FromTerm(q.Graph()),
	}
}

func ToQuad(q *ld.Quad) *Quad {
	S, P, O, G := ToTerm(q.Subject), ToTerm(q.Predicate), ToTerm(q.Object), ToTerm(q.Graph)
	s, _ := json.Marshal(S)
	p, _ := json.Marshal(P)
	o, _ := json.Marshal(O)
	g, _ := json.Marshal(G)
	return &Quad{s, p, o, g, S, P, O, G}
}

func ToTerm(node ld.Node) Term {
	switch node := node.(type) {
	case *ld.IRI:
		return &term{"NamedNode", node.Value}
	case *ld.BlankNode:
		if node.Attribute == "" {
			return &term{"DefaultGraph", ""}
		}
		return &term{"BlankNode", node.Attribute}
	case *ld.Literal:
		datatype := &term{"NamedNode", node.Datatype}
		return &literal{"Literal", node.Value, node.Language, datatype}
	}
	return nil
}
