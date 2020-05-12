package styx

import (
	badger "github.com/dgraph-io/badger/v2"
	iface "github.com/ipfs/interface-go-ipfs-core"

	rdf "github.com/underlay/go-rdfjs"
	indices "github.com/underlay/pkgs/indices"
	types "github.com/underlay/pkgs/types"
	styx "github.com/underlay/styx"
)

type styxIndex struct {
	resource string
	api      iface.CoreAPI
	db       *badger.DB
	store    *styx.Store
}

// NewStyxIndex creates a new Styx index
func NewStyxIndex() indices.GeneratorIndex { return &styxIndex{} }

func (*styxIndex) Close()       {}
func (*styxIndex) Name() string { return "styx" }
func (si *styxIndex) Init(resource string, api iface.CoreAPI, db *badger.DB, path string) {
	si.resource, si.api, si.db = resource, api, db
	tagScheme := styx.NewPrefixTagScheme(resource)
	dictionary, _ := styx.MakeIriDictionary(tagScheme, db)
	quadStore := styx.MakeBadgerStore(db)
	si.store, _ = styx.NewStore(&styx.Config{
		TagScheme:  tagScheme,
		Dictionary: dictionary,
		QuadStore:  quadStore,
	}, db)
}

func (si *styxIndex) Set(key []string, resource types.Resource, dataset []*rdf.Quad, store *styx.Store) error {
	if resource.T() == types.AssertionType && dataset != nil {
		uri := types.GetURI(si.resource, key)
		node := rdf.NewNamedNode(uri)
		err := si.store.Set(node, dataset)
		if err != nil {
			return err
		}
	}
	return nil
}

func (si *styxIndex) Delete(key []string, resource types.Resource, dataset []*rdf.Quad, store *styx.Store) error {
	switch resource.(type) {
	case *types.Assertion:
		uri := types.GetURI(si.resource, key)
		node := rdf.NewNamedNode(uri)
		err := si.store.Delete(node)
		if err != nil {
			return err
		}
	}
	return nil
}

func (si *styxIndex) Head() []*rdf.Quad { return nil }
func (si *styxIndex) Base() []rdf.Term  { return nil }
func (si *styxIndex) Body() []*rdf.Quad { return nil }
func (si *styxIndex) Query(query []*rdf.Quad, domain, index []rdf.Term) (indices.Iterator, error) {
	return si.store.Query(query, domain, index)
}
