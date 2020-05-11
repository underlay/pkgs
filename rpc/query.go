package rpc

import (
	rdf "github.com/underlay/go-rdfjs"
	indices "github.com/underlay/pkgs/indices"
)

func makeIterator(
	query []*rdf.Quad,
	domain, index []rdf.Term,
	signature indices.Signature,
	// matchDomain begins with signature.Base() as a prefix
	// matchIndex are the values from query that are bound to signature.Head()
	matchDomain, matchIndex []rdf.Term,
) (indices.Iterator, error) {
	// There are three sections to the domain we assemble here
	// 1. signature.Base() - of course
	// 2. the nodes whose bound values (i.e. from query) are not variables
	// 3. the user-provided domain

	base := signature.Base()
	signatureDomain := make([]rdf.Term, 0, len(base))
	signatureIndex := make([]rdf.Term, 0, len(base))

	// queryToSignature is *from* variables in query *to* variables in .Head()
	queryToSignature := map[string]rdf.Term{}
	// signatureToQuery is *from* variables in .Head()
	signatureToQuery := map[string]rdf.Term{}
	variables := []int{}
	for i, value := range matchIndex {
		if ground(value) {
			signatureDomain = append(signatureDomain, matchDomain[i])
			signatureIndex = append(signatureIndex, value)
		} else {
			variables = append(variables, i)
			queryToSignature[value.String()] = matchDomain[i]
			signatureToQuery[matchDomain[i].String()] = value
		}
	}

	prefix := signatureIndex

	for i, term := range domain {
		signatureDomain = append(signatureDomain, queryToSignature[term.String()])
		if i < len(index) {
			signatureIndex = append(signatureIndex, index[i])
		}
	}

	iter, err := signature.Query(query, signatureDomain, signatureIndex)
	if err != nil {
		return nil, err
	}

	signatureDomain = iter.Domain()
	queryDomain := make([]rdf.Term, len(signatureDomain)-len(prefix))
	for i := range queryDomain {
		term := signatureDomain[len(prefix)+i]
		queryDomain[i] = signatureToQuery[term.String()]
	}

	return &wrapper{
		false,
		prefix,
		queryDomain,
		queryToSignature,
		iter,
	}, nil
}

type wrapper struct {
	overflow         bool
	prefix           []rdf.Term
	queryDomain      []rdf.Term
	queryToSignature map[string]rdf.Term
	iter             indices.Iterator
}

func (w *wrapper) Get(node rdf.Term) rdf.Term {
	if node == nil {
		return nil
	}

	return w.iter.Get(w.queryToSignature[node.String()])
}

func (w *wrapper) Domain() []rdf.Term { return w.queryDomain }

func (w *wrapper) Index() []rdf.Term {
	signatureIndex := w.iter.Index()
	return signatureIndex[len(w.prefix):]
}

func (w *wrapper) Next(node rdf.Term) ([]rdf.Term, error) {
	if w.overflow {
		return nil, nil
	}

	if node != nil {
		node = w.queryToSignature[node.String()]
	}

	delta, err := w.iter.Next(node)
	if err != nil {
		return nil, err
	}

	if len(delta) > len(w.queryDomain) {
		l := len(w.prefix) + len(w.queryDomain)
		for i := l - len(delta); i < len(w.prefix); i++ {
			p, d := w.prefix[i], delta[i+len(delta)-l]
			if !p.Equal(d) {
				w.overflow = true
				return nil, nil
			}
		}
		delta = delta[len(delta)-len(w.queryDomain):]
	}

	return delta, nil
}

func (w *wrapper) Seek(index []rdf.Term) error {
	w.overflow = false
	signatureIndex := append(w.prefix, index...)
	return w.iter.Seek(signatureIndex)
}

func (w *wrapper) Close() { w.iter.Close() }
