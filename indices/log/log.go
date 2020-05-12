package log

import (
	"log"
	"strings"

	badger "github.com/dgraph-io/badger/v2"
	iface "github.com/ipfs/interface-go-ipfs-core"
	rdf "github.com/underlay/go-rdfjs"
	indices "github.com/underlay/pkgs/indices"
	types "github.com/underlay/pkgs/types"
	styx "github.com/underlay/styx"
)

type logIndex struct{}

// NewLogIndex creates a new log index
func NewLogIndex() indices.Index { return &logIndex{} }

func (logIndex) Name() string { return "log" }

func (logIndex) Init(resource string, api iface.CoreAPI, db *badger.DB, path string) {
	log.Println("Log index: init", path)
}

func (logIndex) Close() { log.Println("Log index: close") }
func (logIndex) Set(key []string, resource types.Resource, dataset []*rdf.Quad, store *styx.Store) error {
	log.Println("Log index: set", "/"+strings.Join(key, "/"), resource.URI())
	return nil
}

func (logIndex) Delete(key []string, resource types.Resource, dataset []*rdf.Quad, store *styx.Store) error {
	log.Println("Log index: delete", "/"+strings.Join(key, "/"), resource.URI())
	return nil
}
