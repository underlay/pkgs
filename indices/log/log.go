package log

import (
	"log"
	"strings"

	badger "github.com/dgraph-io/badger/v2"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	indices "github.com/underlay/pkgs/indices"
	types "github.com/underlay/pkgs/types"
)

var _ indices.Index = (*logIndex)(nil)

type logIndex struct{}

// NewLogIndex creates a new log index
func NewLogIndex() indices.Index { return &logIndex{} }

func (logIndex) Name() string { return "log" }

func (logIndex) Init(api coreiface.CoreAPI, db *badger.DB, path string) {
	log.Println("Log index: init", path)
}

func (logIndex) Close() { log.Println("Log index: close") }
func (logIndex) Set(key []string, resource types.Resource) {
	log.Println("Log index: set", "/"+strings.Join(key, "/"), resource.URI())
}

func (logIndex) Delete(key []string, resource types.Resource) {
	log.Println("Log index: delete", "/"+strings.Join(key, "/"), resource.URI())
}

func (logIndex) Signatures() []indices.Signature { return nil }
