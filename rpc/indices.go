package rpc

import (
	indices "github.com/underlay/pkgs/indices"
	log_index "github.com/underlay/pkgs/indices/log"
	text_index "github.com/underlay/pkgs/indices/text"
	types "github.com/underlay/pkgs/types"
)

// INDICES is the built-in set of indices
var INDICES = []indices.Index{
	log_index.NewLogIndex(),
	text_index.NewTextIndex(),
}

// Delete a resource from all indices
func Delete(key []string, resource types.Resource) {
	for _, index := range INDICES {
		go index.Delete(key, resource)
	}
}

// Set a resource in all indices
func Set(key []string, resource types.Resource) {
	for _, index := range INDICES {
		go index.Set(key, resource)
	}
}
