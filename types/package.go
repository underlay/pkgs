package types

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	proto "github.com/gogo/protobuf/proto"
	cid "github.com/ipfs/go-cid"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	ld "github.com/piprate/json-gold/ld"
)

const defaultSubject = "_:c14n0"

// EmptyDirectory is the CIDv1 of the empty directory
// By default, IPFS nodes only pin the CIDv0 empty directory,
// so we pin the v1 maually on initialization.
const EmptyDirectory = "bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354"

// PackageIri is an rdf:type for Underlay Packages
var PackageIri = ld.NewIRI("http://underlay.mit.edu/ns#Package")

const dateTime = "http://www.w3.org/2001/XMLSchema#dateTime"

// EmptyDirectoryCID is the CID instance of EmptyDirectory
var EmptyDirectoryCID, _ = cid.Decode(EmptyDirectory)

var emptyDirectoryURI = fmt.Sprintf("dweb:/ipfs/%s", EmptyDirectory)
var subject = ld.NewBlankNode("_:b0")
var valueIri = ld.NewIRI("http://www.w3.org/ns/prov#value")
var extentIri = ld.NewIRI("http://purl.org/dc/terms/extent")
var formatIri = ld.NewIRI("http://purl.org/dc/terms/format")
var createdIri = ld.NewIRI("http://purl.org/dc/terms/created")
var modifiedIri = ld.NewIRI("http://purl.org/dc/terms/modified")
var typeIri = ld.NewIRI(ld.RDFType)
var hadMemberIri = ld.NewIRI("http://www.w3.org/ns/prov#hadMember")
var wasRevisionOfIri = ld.NewIRI("http://www.w3.org/ns/prov#wasRevisionOf")
var membershipResourceIri = ld.NewIRI("http://www.w3.org/ns/ldp#membershipResource")
var hasMemberRelationIri = ld.NewIRI("http://www.w3.org/ns/ldp#hasMemberRelation")

var base32 = regexp.MustCompile("^[a-z2-7]{59}$")
var fileURI = regexp.MustCompile("^dweb:/ipfs/([a-z2-7]{59})$")
var messageURI = regexp.MustCompile("^ul:/ipfs/([a-z2-7]{59})$")
var packageURI = regexp.MustCompile("^ul:/ipfs/([a-z2-7]{59})#(_:c14n\\d+)$")

var base = []*ld.Quad{
	ld.NewQuad(subject, typeIri, PackageIri, ""),
	ld.NewQuad(subject, hasMemberRelationIri, hadMemberIri, ""),
}

// NewPackage creates a new timestamped package.
// It does not pin it to IPFS or write it to the database.
func NewPackage(ctx context.Context, pathname, resource string) *Package {
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

	return pkg
}

// ErrNotPackage is returned when a resource unmarhsalls into a non-package unexpectedly
var ErrNotPackage = fmt.Errorf("Unexpected non-package resource")

// GetPackage is a convenience method for retriving a package instance from the database
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

// Set writes the package back to the database.
// It does *not* normalize it; you have to do that yourself.
func (pkg *Package) Set(pathname string, txn *badger.Txn) error {
	r := &Resource{}
	r.Resource = &Resource_Package{Package: pkg}
	return r.Set(pathname, txn)
}

// Paths is a convenience method for getting the path.Resolved version
// of a packages ID and Value CIDs at the same time.
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

// NQuads converts the Package to a slice of ld.*Quads
func (pkg *Package) NQuads(pathname string, txn *badger.Txn) ([]*ld.Quad, error) {
	doc := make([]*ld.Quad, len(base), len(base)+6+len(pkg.Member)*2)
	copy(doc, base)

	_, s, err := GetCid(pkg.Value)
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

	if pkg.RevisionOf != nil && pkg.RevisionOfSubject != "" {
		_, r, err := GetCid(pkg.RevisionOf)
		if err != nil {
			return nil, err
		}
		object := ld.NewIRI(fmt.Sprintf("ul:/ipfs/%s#%s", r, pkg.RevisionOfSubject))
		doc = append(doc, ld.NewQuad(subject, wasRevisionOfIri, object, ""))
	}

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
			_, s, err := GetCid(p.Id)
			if err != nil {
				return nil, err
			}
			uri := ld.NewIRI(fmt.Sprintf("ul:/ipfs/%s#%s", s, p.Subject))
			doc = append(doc,
				ld.NewQuad(subject, hadMemberIri, uri, ""),
				ld.NewQuad(uri, membershipResourceIri, ld.NewIRI(p.Resource), ""),
			)
		} else if m != nil {
			_, s, err := GetCid(m)
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
			_, s, err := GetCid(f.Value)
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
				doc = append(doc, ld.NewQuad(member, membershipResourceIri, ld.NewIRI(resource), ""))
			}
		}
	}

	return doc, nil
}
