package types

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	proto "github.com/gogo/protobuf/proto"
	cid "github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	core "github.com/ipfs/interface-go-ipfs-core"
	options "github.com/ipfs/interface-go-ipfs-core/options"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	multibase "github.com/multiformats/go-multibase"
	ld "github.com/piprate/json-gold/ld"
)

const defaultSubject = "_:c14n0"
const EmptyDirectory = "bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354"

const dateTime = "http://www.w3.org/2001/XMLSchema#dateTime"

var EmptyDirectoryCID, _ = cid.Decode(EmptyDirectory)

var emptyDirectoryURI = fmt.Sprintf("dweb:/ipfs/%s", EmptyDirectory)
var subject = ld.NewBlankNode("_:b0")
var packageIri = ld.NewIRI("http://underlay.mit.edu/ns#Package")
var valueIri = ld.NewIRI("http://www.w3.org/ns/prov#value")
var extentIri = ld.NewIRI("http://purl.org/dc/terms/extent")
var formatIri = ld.NewIRI("http://purl.org/dc/terms/format")
var createdIri = ld.NewIRI("http://purl.org/dc/terms/created")
var modifiedIri = ld.NewIRI("http://purl.org/dc/terms/modified")
var typeIri = ld.NewIRI(ld.RDFType)
var hadMemberIri = ld.NewIRI("http://www.w3.org/ns/prov#hadMember")
var membershipResourceIri = ld.NewIRI("http://www.w3.org/ns/ldp#membershipResource")
var hasMemberRelationIri = ld.NewIRI("http://www.w3.org/ns/ldp#hasMemberRelation")

var base32 = regexp.MustCompile("^[a-z2-7]{59}$")
var fileURI = regexp.MustCompile("^dweb:/ipfs/([a-z2-7]{59})$")
var messageURI = regexp.MustCompile("^ul:/ipfs/([a-z2-7]{59})$")
var packageURI = regexp.MustCompile("^ul:/ipfs/([a-z2-7]{59})#(_:c14n\\d+)$")

var base = []*ld.Quad{
	ld.NewQuad(subject, typeIri, packageIri, ""),
	ld.NewQuad(subject, hasMemberRelationIri, hadMemberIri, ""),
}

func NewPackage(ctx context.Context, pathname, resource string, fs core.UnixfsAPI) (cid.Cid, *Package, error) {
	dateTime := time.Now().Format(time.RFC3339)

	pkg := &Package{
		Resource: resource,
		Subject:  defaultSubject,
		Value:    EmptyDirectoryCID.Bytes(),
		Extent:   4,
		Created:  dateTime,
		Modified: dateTime,
		Member:   make([]string, 0),
	}

	c, err := pkg.Normalize(ctx, pathname, fs, nil)
	return c, pkg, err
}

var ErrNotPackage = fmt.Errorf("Unexpected non-package resource")

func GetPackage(pathname string, txn *badger.Txn) (*Package, error) {
	r := &Resource{}
	err := r.Get(pathname, txn)
	if err != nil {
		return nil, err
	}
	p := r.GetPackage()
	if p == nil {
		return nil, ErrNotPackage
	}
	return p, nil
}

func (pkg *Package) Set(pathname string, txn *badger.Txn) error {
	r := &Resource{}
	r.Resource = &Resource_Package{Package: pkg}
	return r.Set(pathname, txn)
}

func (pkg *Package) Paths() (path.Resolved, path.Resolved, error) {
	id, err := cid.Cast(pkg.Id)
	if err != nil {
		return nil, nil, err
	}
	value, err := cid.Cast(pkg.Value)
	if err != nil {
		return nil, nil, err
	}
	return path.IpfsPath(id), path.IpfsPath(value), nil
}

// Normalize re-computes the normalized n-quads representation of the package,
// pins it to IPFS, and sets the pkg.Id with the result. It returns the string cid.
// You probably want to be careful about unpinning the resulting CID sometime afterwards
func (pkg *Package) Normalize(ctx context.Context, path string, fs core.UnixfsAPI, txn *badger.Txn) (c cid.Cid, err error) {
	ds := ld.NewRDFDataset()
	ds.Graphs["@default"], err = pkg.NQuads(path, txn)
	if err != nil {
		return
	}

	api := ld.NewJsonLdApi()
	var res interface{}
	res, err = api.Normalize(ds, Opts)
	if err != nil {
		return
	}

	reader := strings.NewReader(res.(string))
	resolved, err := fs.Add(
		ctx,
		files.NewReaderFile(reader),
		options.Unixfs.Pin(true),
		options.Unixfs.RawLeaves(true),
		options.Unixfs.CidVersion(1),
	)

	if err != nil {
		return
	}

	c = resolved.Cid()
	pkg.Id = c.Bytes()
	return
}

// NQuads converts the Package to a slice of ld.*Quads
func (pkg *Package) NQuads(pathname string, txn *badger.Txn) ([]*ld.Quad, error) {
	doc := make([]*ld.Quad, len(base), len(base)+5+len(pkg.Member)*2)
	copy(doc, base)

	c, err := cid.Cast(pkg.Value)
	if err != nil {
		return nil, err
	}
	s, err := c.StringOfBase(multibase.Base32)
	if err != nil {
		return nil, err
	}
	value := ld.NewIRI(fmt.Sprintf("dweb:/ipfs/%s", s))
	extent := strconv.FormatUint(pkg.Extent, 10)
	doc = append(doc,
		ld.NewQuad(subject, membershipResourceIri, ld.NewIRI(pkg.Resource), ""),
		ld.NewQuad(subject, valueIri, value, ""),
		ld.NewQuad(value, extentIri, ld.NewLiteral(extent, ld.XSDInteger, ""), ""),
		ld.NewQuad(subject, createdIri, ld.NewLiteral(pkg.Created, dateTime, ""), ""),
		ld.NewQuad(subject, modifiedIri, ld.NewLiteral(pkg.Modified, dateTime, ""), ""),
	)

	for _, name := range pkg.Member {
		var key string
		if pathname == "/" {
			key = "/" + name
		} else {
			key = fmt.Sprintf("%s/%s", pathname, name)
		}
		item, err := txn.Get([]byte(key))
		if err != nil {
			return nil, err
		}
		resource := &Resource{}
		err = item.Value(func(val []byte) error {
			return proto.Unmarshal(val, resource)
		})
		if err != nil {
			return nil, err
		}
		p, m, f := resource.GetPackage(), resource.GetMessage(), resource.GetFile()
		if p != nil {
			resource := ld.NewIRI(p.Resource)
			uri := ld.NewIRI(fmt.Sprintf("ul:/ipfs/%s#%s", p.Id, p.Subject))
			doc = append(doc,
				ld.NewQuad(subject, hadMemberIri, uri, ""),
				ld.NewQuad(uri, membershipResourceIri, resource, ""),
			)
		} else if m != nil {
			c, err := cid.Cast(m)
			if err != nil {
				return nil, err
			}
			s, err := c.StringOfBase(multibase.Base32)
			if err != nil {
				return nil, err
			}
			member := ld.NewIRI("ul:/ipfs/" + s)
			doc = append(doc, ld.NewQuad(subject, hadMemberIri, member, ""))
			if s != name {
				resource := fmt.Sprintf("%s/%s", pkg.Resource, name)
				doc = append(doc, ld.NewQuad(subject, membershipResourceIri, ld.NewIRI(resource), ""))
			}
		} else if f != nil {
			c, err := cid.Cast(f.Value)
			if err != nil {
				return nil, err
			}
			s, err := c.StringOfBase(multibase.Base32)
			if err != nil {
				return nil, err
			}

			member := ld.NewIRI("dweb:/ipfs/" + s)
			extent := strconv.FormatUint(f.Extent, 10)
			doc = append(doc,
				ld.NewQuad(subject, hadMemberIri, member, ""),
				ld.NewQuad(member, extentIri, ld.NewLiteral(extent, ld.XSDInteger, ""), ""),
				ld.NewQuad(member, formatIri, ld.NewLiteral(f.Format, ld.XSDString, ""), ""),
			)
			if s != name {
				resource := fmt.Sprintf("%s/%s", pkg.Resource, name)
				doc = append(doc, ld.NewQuad(subject, membershipResourceIri, ld.NewIRI(resource), ""))
			}
		}
	}

	return doc, nil
}

// JSON converts the Package to a JSON-LD document
func (pkg *Package) JSON(path string, txn *badger.Txn) (map[string]interface{}, error) {
	members := make([]map[string]interface{}, 0, len(pkg.Member))

	for _, name := range pkg.Member {
		key := fmt.Sprintf("%s/%s", path, name)
		item, err := txn.Get([]byte(key))
		if err != nil {
			return nil, err
		}
		resource := &Resource{}
		err = item.Value(func(val []byte) error {
			return proto.Unmarshal(val, resource)
		})
		if err != nil {
			return nil, err
		}
		p, m, f := resource.GetPackage(), resource.GetMessage(), resource.GetFile()
		if p != nil {
			members = append(members, map[string]interface{}{
				"@id":                    fmt.Sprintf("ul:/ipfs/%s#%s", p.Id, p.Subject),
				"ldp:membershipResource": p.Resource,
			})
		} else if m != nil {
			_, s, err := GetCid(m)
			if err != nil {
				return nil, err
			}
			member := map[string]interface{}{"@id": "ul:/ipfs/" + s}
			if name != s {
				member["ldp:membershipResource"] = fmt.Sprintf("%s/%s", pkg.Resource, name)
			}
			members = append(members, member)
		} else if f != nil {
			_, s, err := GetCid(f.Value)
			if err != nil {
				return nil, err
			}

			member := map[string]interface{}{
				"@id":            "dweb:/ipfs/" + s,
				"dcterms:extent": f.Extent,
				"dcterms:format": f.Format,
			}

			if name != s {
				member["ldp:membershipResource"] = fmt.Sprintf("%s/%s", pkg.Resource, name)
			}

			members = append(members, member)
		}
	}

	_, s, err := GetCid(pkg.Value)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"@context":               ContextURL,
		"@type":                  packageIri.Value,
		"ldp:hasMemberRelation":  "prov:hadMember",
		"ldp:membershipResource": pkg.Resource,
		"dcterms:created":        pkg.Created,
		"dcterms:modified":       pkg.Modified,
		"prov:value": map[string]interface{}{
			"@id":            fmt.Sprintf("dweb:/ipfs/%s", s),
			"dcterms:extent": pkg.Extent,
		},
		"prov:hadMember": members,
	}, nil
}
