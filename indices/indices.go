package indices

import (
	"github.com/underlay/pkgs/query"
)

// INDICES is the built-in set of indices
var INDICES = []query.Index{&nameIndex{}}
