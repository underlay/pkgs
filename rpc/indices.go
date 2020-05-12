package rpc

import (
	"log"
	"sync"

	iface "github.com/ipfs/interface-go-ipfs-core"
	rdf "github.com/underlay/go-rdfjs"
	indices "github.com/underlay/pkgs/indices"
	log_index "github.com/underlay/pkgs/indices/log"
	styx_index "github.com/underlay/pkgs/indices/styx"
	types "github.com/underlay/pkgs/types"
	styx "github.com/underlay/styx"
)

var rpcStyxIndex = styx_index.NewStyxIndex()

// INDICES is the built-in set of indices
var INDICES = []indices.Index{
	log_index.NewLogIndex(),
	rpcStyxIndex,
}

// RULES is the built-in set of generators
var RULES = []indices.Rule{}

// Delete a resource from all indices
func Delete(key []string, resource types.Resource, api iface.CoreAPI) {
	var dataset []*rdf.Quad
	var store *styx.Store
	switch resource := resource.(type) {
	case *types.Assertion:
		dataset = resource.GetDataset(api)

		var err error
		store, err = styx.NewMemoryStore(nil)
		if err != nil {
			log.Println("Error creating memory store:", err)
			log.Println("Closing store:", store.Close())
			return
		}

		err = store.Set(rdf.Default, dataset)
		if err != nil {
			log.Println("Error setting assertion in memory store:", err)
			log.Println("Closing store:", store.Close())
			return
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(INDICES))
	for _, index := range INDICES {
		go setIndex(false, key, resource, index, dataset, store, &wg)
	}

	wg.Wait()
	if store != nil {
		store.Close()
	}
}

// Set a resource in all the indices
func Set(key []string, resource types.Resource, api iface.CoreAPI) {
	var store *styx.Store
	var dataset []*rdf.Quad
	switch resource := resource.(type) {
	case *types.Assertion:
		dataset = resource.GetDataset(api)

		var err error
		store, err = styx.NewMemoryStore(nil)
		if err != nil {
			log.Println("Error creating memory store:", err)
			log.Println("Closing store:", store.Close())
			return
		}

		err = store.Set(rdf.Default, dataset)
		if err != nil {
			log.Println("Error setting assertion in memory store:", err)
			log.Println("Closing store:", store.Close())
			return
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(INDICES))
	for _, index := range INDICES {
		go setIndex(true, key, resource, index, dataset, store, &wg)
	}

	wg.Wait()
	if store != nil {
		store.Close()
	}
}

func setIndex(
	set bool,
	key []string, resource types.Resource,
	index indices.Index,
	dataset []*rdf.Quad, store *styx.Store,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	var err error
	if set {
		err = index.Set(key, resource, dataset, store)
	} else {
		err = index.Delete(key, resource, dataset, store)
	}

	if err != nil {
		log.Printf("Error setting index %s: %s\n", index.Name(), err.Error())
		return
	}
}
