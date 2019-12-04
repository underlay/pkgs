package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	badger "github.com/dgraph-io/badger/v2"
	proto "github.com/gogo/protobuf/proto"
	core "github.com/ipfs/interface-go-ipfs-core"

	types "github.com/underlay/pkgs/types"
)

func Head(ctx context.Context, res http.ResponseWriter, req *http.Request, db *badger.DB, api core.CoreAPI) error {
	path := req.URL.Path
	if path != "/" && !pathRegex.MatchString(path) {
		res.WriteHeader(404)
		return nil
	}

	accept := req.Header.Get("Accept")
	ifNoneMatch := req.Header.Get("If-None-Match")

	resource := &types.Resource{}
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(path))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return proto.Unmarshal(val, resource)
		})
	})

	if err == badger.ErrKeyNotFound {
		res.WriteHeader(404)
		return nil
	} else if err != nil {
		res.WriteHeader(500)
		return err
	}

	// Now that we know there's a resource here, we add the first Link header
	res.Header().Add("Link", linkTypeResource)

	// Okay now we have a Resource and we get to respond with its representation
	p, m, f := resource.GetPackage(), resource.GetMessage(), resource.GetFile()
	if f != nil {

		res.Header().Add("Link", linkTypeNonRDFSource)

		_, s, err := types.GetCid(f.Value)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		if ifNoneMatch == s {
			res.WriteHeader(304)
			return nil
		}

		extent := strconv.FormatUint(f.Extent, 10)
		res.Header().Add("ETag", s)
		res.Header().Add("Content-Type", f.Format)
		res.Header().Add("Content-Length", extent)
	} else if m != nil {
		_, s, err := types.GetCid(m)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		res.Header().Add("Link", linkTypeRDFSource)

		if ifNoneMatch == s {
			res.WriteHeader(304)
			return nil
		}

		res.Header().Add("ETag", s)
		if accept == "application/n-quads" || accept == "application/ld+json" {
			res.Header().Add("Content-Type", accept)
		} else {
			res.WriteHeader(406)
			return nil
		}
	} else if p != nil {
		_, s, err := types.GetCid(p.Id)
		if err != nil {
			res.WriteHeader(500)
			return err
		}

		if ifNoneMatch == s {
			res.WriteHeader(304)
			return nil
		}

		if accept == "application/n-quads" || accept == "application/ld+json" {
			res.Header().Add("ETag", s)
			res.Header().Add("Link", fmt.Sprintf(`<#%s>; rel="self"`, p.Subject))
			res.Header().Add("Content-Type", accept)
		} else {
			res.WriteHeader(406)
		}
	}
	return nil
}
