package types

import (
	"log"

	badger "github.com/dgraph-io/badger/v2"
	proto "github.com/gogo/protobuf/proto"
	cid "github.com/ipfs/go-cid"
	multibase "github.com/multiformats/go-multibase"
	query "github.com/underlay/pkgs/query"
)

// ContextURL shouldn't be hardcoded; will factor out in the future
const ContextURL = "ipfs://bafkreifcqgsljpst2fabpvmlzcf5fqdthzvhf4imvqvnymk5iifi6mdtru"

const messageLength = 36

// RawContext is the raw package compaction context
var RawContext = []byte(`{
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
`)

// GetPackage retrieves a resource, assuming that it is a package
func GetPackage(pathname string, txn *badger.Txn) (p *Package, err error) {
	item, err := txn.Get([]byte(pathname))
	if err != nil {
		return nil, err
	}
	if item.UserMeta() == uint8(query.PackageType) {
		p = &Package{}
		err = item.Value(func(val []byte) error {
			return proto.Unmarshal(val, p)
		})
	} else {
		err = ErrNotPackage
	}
	return
}

// GetResource retrives the appropriate resource from the given path
func GetResource(pathname string, txn *badger.Txn) (r query.Resource, err error) {
	var item *badger.Item
	item, err = txn.Get([]byte(pathname))
	if err != nil {
		return
	}

	switch query.ResourceType(item.UserMeta()) {
	case query.PackageType:
		item.Value(func(val []byte) error {
			p := &Package{}
			r = p
			return proto.Unmarshal(val, p)
		})
	case query.MessageType:
		var val []byte
		val, err = item.ValueCopy(make([]byte, messageLength))
		r = Message(val)
	case query.FileType:
		item.Value(func(val []byte) error {
			f := &File{}
			r = f
			return proto.Unmarshal(val, f)
		})
	}
	return
}

// SetResource marshalls a resource and writes it to the database
func SetResource(value query.Resource, pathname string, txn *badger.Txn) (err error) {
	var val []byte
	switch t := value.(type) {
	case *Package:
		val, err = proto.Marshal(t)
	case Message:
		val = t
	case *File:
		val, err = proto.Marshal(t)
	default:
		log.Fatalln("Invalid resource")
	}

	if err != nil {
		return
	}

	key := []byte(pathname)
	e := badger.NewEntry(key, val).WithMeta(byte(value.Type()))
	return txn.SetEntry(e)
}

// getCid is a convenience method for turning byte slices
// into CID strings and instances at the same time.
func getCid(val []byte) (cid.Cid, string, error) {
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
