package rpc

import (
	rdf "github.com/underlay/go-rdfjs"
	indices "github.com/underlay/pkgs/indices"
)

func getIterator(query []*rdf.Quad, domain, index []rdf.Term) (indices.Iterator, error) {
	// rule, ruleDomain, ruleIndex := getRule(quads, domain)
	// if rule == nil {
	// 	return nil, jsonrpc2.CodeInternalError, errors.New("No matching query signature found")
	// }

	// handler.Iterator, err = wrapGenerator(
	// 	quads, domain, index,
	// 	rule, ruleDomain, ruleIndex,
	// )
	return rpcStyxIndex.Query(query, domain, index)
}
