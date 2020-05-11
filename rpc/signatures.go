package rpc

import (
	"log"

	rdf "github.com/underlay/go-rdfjs"
	indices "github.com/underlay/pkgs/indices"
	styx "github.com/underlay/styx"
)

var signatures = []indices.Signature{}

func init() {
	for _, index := range INDICES {
		signatures = append(signatures, index.Signatures()...)
	}
}

func getSignature(query []*rdf.Quad, domain []rdf.Term) (indices.Signature, []rdf.Term, []rdf.Term) {
	store, err := styx.NewMemoryStore(nil)
	defer store.Close()
	if err != nil {
		log.Fatalln(err)
	}

	err = store.Set(rdf.Default, query)
	if err != nil {
		log.Fatalln(err)
	}

	for _, signature := range signatures {
		head, base := signature.Head(), signature.Base()
		if head == nil {
			return signature, nil, nil
		} else if len(query) != len(head) {
			continue
		}

		iter, err := store.Query(head, base, nil)
		if err != nil {
			log.Fatalln(err)
		}

		delta, err := iter.Next(nil)
		if err != nil {
			log.Fatalln(err)
		}

		values := make([]rdf.Term, len(delta))
		copy(values, delta)
		for delta != nil {
			complete := true
			for i, term := range base {
				if !ground(values[i]) {
					delta, err = iter.Next(term)
					if err != nil {
						log.Fatalln(err)
					}
					copy(values[len(values)-len(delta):], delta)
					complete = false
					break
				}
			}
			if complete {
				break
			}
		}

		iter.Close()

		if delta == nil {
			continue
		}

		return signature, iter.Domain(), values
	}
	return nil, nil, nil
}

func ground(term rdf.Term) bool {
	t := term.TermType()
	return t != rdf.VariableType && t != rdf.BlankNodeType
}
