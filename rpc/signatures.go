package rpc

import (
	ld "github.com/piprate/json-gold/ld"

	indices "github.com/underlay/pkgs/indices"
	query "github.com/underlay/pkgs/query"
)

var signatures = []query.Signature{}

func init() {
	for _, index := range indices.INDICES {
		signatures = append(signatures, index.Signatures()...)
	}
}

func getSignature(query []*ld.Quad) (query.Signature, map[string]ld.Node) {
	for _, signature := range signatures {
		head, domain := signature.Head(), signature.Domain()
		unified, assignments := matchSignature(query, head, domain)
		if unified {
			return signature, assignments
		}
	}
	return nil, nil
}

func matchSignature(query []*ld.Quad, head []*ld.Quad, domain []*ld.BlankNode) (bool, map[string]ld.Node) {
	if head == nil {
		return true, nil
	} else if len(query) != len(head) {
		return false, nil
	}

	// Could be a lot smarter about pruning bad permutations
	for p := make([]int, len(query)); p[0] < len(p); nextPerm(p) {
		d := map[string]ld.Node{}
		unified := true
		for i, quad := range getPerm(query, p) {
			if !unifyQuad(quad, head[i], domain, d) {
				unified = false
				break
			}
		}
		if unified {
			return true, d
		}
	}
	return false, nil
}

func nextPerm(p []int) {
	for i := len(p) - 1; i >= 0; i-- {
		if i == 0 || p[i] < len(p)-i-1 {
			p[i]++
			return
		}
		p[i] = 0
	}
}

func getPerm(orig []*ld.Quad, p []int) []*ld.Quad {
	result := append([]*ld.Quad{}, orig...)
	for i, v := range p {
		result[i], result[i+v] = result[i+v], result[i]
	}
	return result
}

func unifyQuad(a, b *ld.Quad, domain []*ld.BlankNode, d map[string]ld.Node) (unified bool) {
	return unify(a.Subject, b.Subject, domain, d) &&
		unify(a.Predicate, b.Predicate, domain, d) &&
		unify(a.Object, b.Object, domain, d)
}

// a is from the query, b is from the signature
func unify(a, b ld.Node, domain []*ld.BlankNode, d map[string]ld.Node) bool {
	switch b := b.(type) {
	case *ld.IRI:
		return a.Equal(b)
	case *ld.Literal:
		return a.Equal(b)
	case *ld.BlankNode:
		if c, has := d[b.Attribute]; has {
			return a.Equal(c)
		}
		if ld.IsBlankNode(a) {
			for _, node := range domain {
				if node.Equal(b) {
					return false
				}
			}
		}
		d[b.Attribute] = a
		return true
	}
	return false
}
