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
)

func (r *Resource) Get(path string, txn *badger.Txn) error {
	item, err := txn.Get([]byte(path))
	if err != nil {
		return err
	}

	return item.Value(func(val []byte) error {
		return proto.Unmarshal(val, r)
	})
}

func (r *Resource) Set(path string, txn *badger.Txn) error {
	val, err := proto.Marshal(r)
	if err != nil {
		return err
	}

	return txn.Set([]byte(path), val)
}

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
