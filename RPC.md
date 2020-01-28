# RPC Query API

In addition to a REST HTTP API for managing the resources in a package server, pkgs also exposes an RPC API for interactively querying its contents.

The general query paradigm is inherited from Datalog, with some adaptation to graphs.

## Overview

The client begins a query "session" by opening a connection to the server. We just use TCP for now, but will generalize to libp2p streams so that e.g. browser clients can use websockets.

The client sends the sever a [generalized RDF graph](http://www.w3.org/TR/rdf11-concepts/#section-generalized-rdf) (a set of generalized RDF Triples that allow e.g. blank nodes as predicates) called the "query pattern" or just "query". This query pattern is only sent once and forms the basis of the entire session. Any blank nodes in the query are interpreted by the sever as existential variables.

Every graph with one or more blank nodes defines an infinitely large set of _[ground](https://www.w3.org/TR/rdf11-mt/#dfn-ground) [instances](https://www.w3.org/TR/rdf11-mt/#dfn-instance)_ - assignments of each blank node to an IRI or RDF Literal. Since we are dealing with generalized RDF query patterns, we only consider ground instances that are also RDF graphs (i.e. a blank node cannot be assigned a Literal value if it appears as a subject or predicate in the query).

For example, the following are all ground instances of the query `_:b0 <http://schema.org/name> "Johnny Appleseed" .`:

- `<http://people.com/johnny> <http://schema.org/name> "Johnny Appleseed" .`
- `<http://people.com/johnnnnnnnny> <http://schema.org/name> "Johnny Appleseed" .`
- `<http://foo.com/foo/foo/jfkdlsa> <http://schema.org/name> "Johnny Appleseed" .`
- `<dweb:/ipfs/bafybeiatr6vzozvaxtp5f32ghixj4bvauz6wgl4lbbh6np4yrrsvtep3y4> <http://schema.org/name> "Johnny Appleseed" .`

These ground instances are potential solutions to the query, and it's up to each server to determine which (if any) of them are "true". A query solution is _true_

if it is entailed by the union of all the RDF resources (i.e. messages and packages) in the package server.

The [RDF 1.1 Semantics](https://www.w3.org/TR/rdf11-mt/#dfn-rdf-entail) document frames the truth of an RDF graph in terms of "interpretations" and "entailment regimes", which are

A "solution" to a query is an assignment of all the blank nodes in the query pattern to IRIs or RDF Literals, and a "result set" is the set of distinct solutions that the server deems "true". Result sets might be infinitely large, and it's up to each package server to pick an entailment regime that will determine which solutions are true.

- The query parametrizes a (potentially infinite) result set, which the client then can interactively explore.
  - A result in the result set is an assignment of all the blank nodes in the query pattern to IRIs or RDF Literals that the server deems "true".
  - Result sets may be infinitely large, but the server guarantees that the results are totally ordered: there is a deterministic first result, and for each result there is a deterministic next result.
  - Since the result set may be infinite, and since many of its members differ only by one variable assignment, entire result sets are never materialized.
  - Instead, the server instantiates a query "cursor" that , and the client can
  - Evaluting which assignments are true could be as simple as strict subgraph matching ("what are assignments to the variables such that the resulting graph is a subgraph of the merged graph of all messages?")
    - More complex
  - The

pkgs doesn't resolve queries for you - all it does is route new sessions to appropriate indices (like database indexes) that know how to handle them.

## Indices

pkgs ships with a few built-in indices, but you typically will want to provide you own. If you know a lot about the resources that will be in your packages and the kinds of access patterns you want to support, you can write very specific indices that do exactly what you want and nothing else. If you don't know very much about your data or how you need to access it, you can fall back onto more general (and slower and more expensive) indices.

An index is any value that satisfies the `Index` interface:

```golang
type Index interface {
	Open()
	Close()
	Add(path []string, resource Resource)
	Remove(path []string, resource Resource)
	Signatures() []Signature
}
```

Indices accumulate state through their `Add` and `Remove` methods. Those will get called on every change to any resource in the entire server - for additions, only `Add` will be called; for deletions, only `Remove` will be called; and for mutations, `Remove` will be called first, followed by `Add`. All that they're passed is the path of the changed resource (or `nil` if the resource is the root package), and a representation of the resource exposing the its type, CID, and URI.

```golang
type Resource interface {
	Type() ResourceType
	ETag() (cid.Cid, string)
	URI() string
}

type ResourceType uint8

const (
	Package ResourceType = iota
	Message
	File
)
```

Every index is associated with a (static) set of _signatures_ (as in type signature, not cryptographic signature).

```golang
type Signature interface {
	Head() []*ld.Quad
	Domain() []*ld.BlankNode
	Query(
    query []*ld.Quad,
    assignments map[string]ld.Node,
    domain []*ld.BlankNode,
    index []ld.Node,
  ) (Cursor, error)
}
```

Here, a signature `Head` is like the head of a Datalog rule: it serves as a pattern-matching handle for what queries it's able to resolve. The `Domain` property is a set of blank nodes in the head that must match to IRIs or Literals in the query - only blank nodes not in the domain may match blank nodes in a query pattern.

For example, a head for an index that lets users look up URI ids by `http://schema.org/name` name might have a `Head`:

```
_:b0 <http://schema.org/name> _:b1 .
```

with a `Domain`:

```
{ _:b1 }
```

which would match the query

```
_:b0 <http://schema.org/name> "Johnny Appleseed" .
```

but would _not_ match the query

```
<http://example.com/johnny> <http://schema.org/name> _:b0 .
```

If our name index was bidirectional - that is, if the `Query` method of the signature could handle blank nodes in either the subject or object positions (or both!) - then we would change our Domain to be empty, which would cause both example queries to match.

The reason that signatures have to declare explicit materialized heads and domains (as opposed to just a `Match(query: []*ld.Quad): bool`) is to allow _composable indices_. In the future, packages servers can be clever about combining multiple indices to create composite cursors for queries that don't validate any one index, but for now, pkgs just performs strict matching on query graphs and signature heads.

```golang
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
```
