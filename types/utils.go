package types

import (
	"context"

	badger "github.com/dgraph-io/badger/v2"
	proto "github.com/gogo/protobuf/proto"
	cid "github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	core "github.com/ipfs/interface-go-ipfs-core"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	multibase "github.com/multiformats/go-multibase"
	ld "github.com/piprate/json-gold/ld"
)

// ContextURL shouldn't be hardcoded; will factor out in the future
const ContextURL = "ipfs://bafkreifcqgsljpst2fabpvmlzcf5fqdthzvhf4imvqvnymk5iifi6mdtru"

const rawContext = `{
	"@context": {
		"dcterms": "http://purl.org/dc/terms/",
		"prov": "http://www.w3.org/ns/prov#",
		"ldp": "http://www.w3.org/ns/ldp#",
		"xsd": "http://www.w3.org/2001/XMLSchema#",
		"dcterms:created": {
			"@type": "xsd:dateTime"
		},
		"dcterms:modified": {
			"@type": "xsd:dateTime"
		},
		"ldp:membershipResource": {
			"@type": "@id"
		},
		"ldp:hasMemberRelation": {
			"@type": "@id"
		}
	}
}
`

// PackageFrame is the JSON-LD Frame used for framing packages
var PackageFrame = map[string]interface{}{
	"@context": ContextURL,
	"@type":    packageIri.Value,
}

// Proc is the multi-purpose JSON-LD processor we use for everything
var Proc = ld.NewJsonLdProcessor()

// Opts are the JSON-LD processing options we use for everything
var Opts = ld.NewJsonLdOptions("")

// ETag is a convenience function that multiplexes between Resource
// types to return their CID as a string (ID for packages, not Value)
func (r *Resource) ETag() (etag []byte) {
	p, m, f := r.GetPackage(), r.GetMessage(), r.GetFile()
	if p != nil {
		etag = p.Id
	} else if m != nil {
		etag = m
	} else if f != nil {
		etag = f.Value
	}
	return
}

// Get unmarshalls a resource from the database
func (r *Resource) Get(pathname string, txn *badger.Txn) error {
	item, err := txn.Get([]byte(pathname))
	if err != nil {
		return err
	}

	return item.Value(func(val []byte) error {
		return proto.Unmarshal(val, r)
	})
}

// Set marshalls a resource and writes it to the database
func (r *Resource) Set(pathname string, txn *badger.Txn) error {
	val, err := proto.Marshal(r)
	if err != nil {
		return err
	}

	return txn.Set([]byte(pathname), val)
}

// GetCid is a convenience method for turning byte slices
// into CID strings and instances at the same time.
func GetCid(val []byte) (cid.Cid, string, error) {
	c, err := cid.Cast(val)
	if err != nil {
		return cid.Undef, "", err
	}

	s, err := c.StringOfBase(multibase.Base32)
	if err != nil {
		return cid.Undef, "", err
	}

	return c, s, nil
}

// GetFile Do we use this?
func GetFile(ctx context.Context, c cid.Cid, fs core.UnixfsAPI) (files.File, error) {
	node, err := fs.Get(ctx, path.IpfsPath(c))
	if err != nil {
		return nil, err
	}

	file, is := node.(files.File)
	if !is {
		return nil, files.ErrNotReader
	}

	return file, nil
}
