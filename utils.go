package main

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	badger "github.com/dgraph-io/badger/v2"
	cid "github.com/ipfs/go-cid"

	options "github.com/ipfs/interface-go-ipfs-core/options"
	path "github.com/ipfs/interface-go-ipfs-core/path"

	types "github.com/underlay/pkgs/types"
)

// ErrNotPackage is returned when a resource unmarhsalls into a non-package unexpectedly
var ErrNotPackage = errors.New("Unexpected non-package resource")

var emptyDirectoryCID, _ = cid.Decode("bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354")

// EmptyDirectoryPath is the path for the empty directory
var EmptyDirectoryPath = path.IpfsPath(emptyDirectoryCID)

var addOpts = []options.UnixfsAddOption{
	options.Unixfs.CidVersion(1),
	options.Unixfs.RawLeaves(true),
	options.Unixfs.Pin(false),
}

func getKey(key []string) []byte { return []byte("/" + strings.Join(key, "/")) }

func getName(resource string) (name string) {
	i := strings.LastIndexByte(resource, '/')
	if i != -1 {
		name = resource[i+1:]
	}
	return
}

func deletePackageMember(pkg *types.Package, name string, isCid bool) types.ResourceType {
	if i, old := pkg.SearchPackages(name, isCid); old != nil {
		l := len(pkg.Members.Packages)
		copy(pkg.Members.Packages[i:], pkg.Members.Packages[i+1:])
		pkg.Members.Packages[l-1] = nil
		pkg.Members.Packages = pkg.Members.Packages[:l-1]
		return types.PackageType
	} else if i, old := pkg.SearchAssertions(name, isCid); old != nil {
		l := len(pkg.Members.Assertions)
		copy(pkg.Members.Assertions[i:], pkg.Members.Assertions[i+1:])
		pkg.Members.Assertions[l-1] = nil
		pkg.Members.Assertions = pkg.Members.Assertions[:l-1]
		return types.AssertionType
	} else if i, old := pkg.SearchFiles(name, isCid); old != nil {
		l := len(pkg.Members.Files)
		copy(pkg.Members.Files[i:], pkg.Members.Files[i+1:])
		pkg.Members.Files[l-1] = nil
		pkg.Members.Files = pkg.Members.Files[:l-1]
		return types.FileType
	}

	return 0
}

func setResource(key []string, r types.Resource, txn *badger.Txn) error {
	k := getKey(key)
	v, err := json.Marshal(r)
	if err != nil {
		return err
	}

	e := badger.NewEntry(k, v)
	switch r.(type) {
	case *types.Package:
		e = e.WithMeta(byte(types.PackageType))
	case *types.Assertion:
		e = e.WithMeta(byte(types.AssertionType))
	case *types.File:
		e = e.WithMeta(byte(types.FileType))
	default:
		log.Fatalln("Attempted to set *resource in setResource")
	}

	return txn.SetEntry(e)
}

func getResource(key []string, txn *badger.Txn) (types.Resource, error) {
	item, err := txn.Get(getKey(key))
	if err != nil {
		return nil, err
	}

	var r types.Resource
	switch types.ResourceType(item.UserMeta()) {
	case types.PackageType:
		r = &types.Package{}
	case types.AssertionType:
		r = &types.Assertion{}
	case types.FileType:
		r = &types.File{}
	}

	return r, item.Value(func(val []byte) error { return json.Unmarshal(val, r) })
}

func getPackage(key []string, txn *badger.Txn) (*types.Package, error) {
	item, err := txn.Get(getKey(key))
	if err != nil {
		return nil, err
	}

	if item.UserMeta() != uint8(types.PackageType) {
		return nil, ErrNotPackage
	}

	p := &types.Package{}
	err = item.Value(func(val []byte) error { return json.Unmarshal(val, p) })
	if err != nil {
		return nil, err
	}

	return p, nil
}
