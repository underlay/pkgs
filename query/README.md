# Query RPC

The Cursor interface that pkgs uses

```golang
// A Cursor is an interactive query interface
type Cursor interface {
	Graph() []*ld.Quad
	Get(node *ld.BlankNode) ld.Node
	Domain() []*ld.BlankNode
	Index() []ld.Node
	Next(node *ld.BlankNode) ([]*ld.BlankNode, error)
	Seek(index []ld.Node) error
	Close()
}
```

is also exposed over a libp2p stream as a JSON RPC-style protocol.

## Create a new Cursor

```json
// Generalized RDF is okay here.
// Use blank nodes as predicates if you need
{ "pattern": {} }
```

### `Cursor.Graph()`

Request:

```json
{ "graph": {} }
```

Response:

```json
{
	"graph": {
		// ...
	}
}
```

### `Cursor.Graph()`

```json
{ "@type": "Cursor" }
```

### `Cursor.Graph()`

```json
{ "@type": "Cursor" }
```
