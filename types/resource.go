package types

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	cid "github.com/ipfs/go-cid"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	ld "github.com/piprate/json-gold/ld"
)

// Resource is interface type for resources (packages, messages, and files)
type Resource interface {
	ETag() (cid.Cid, string)
	URI() string
}

// A Message is just the bytes of a CID
type Message []byte

// ETag satisfies the Resource interface
func (m Message) ETag() (cid.Cid, string) {
	c, s, _ := getCid(m)
	return c, s
}

// URI satisfies the Resource interface
func (m Message) URI() string {
	_, s, _ := getCid(m)
	return fmt.Sprintf("ul:/ipfs/%s", s)
}

// ETag satisfies the Resource interface for Packages
func (p *Package) ETag() (cid.Cid, string) {
	c, s, _ := getCid(p.Id)
	return c, s
}

// URI satisfies the Resource interface
func (p *Package) URI() string {
	_, s, _ := getCid(p.Id)
	return fmt.Sprintf("ul:/ipfs/%s#%s", s, p.Subject)
}

// ETag satisfies the Resource interface for Files
func (f *File) ETag() (cid.Cid, string) {
	c, s, _ := getCid(f.Value)
	return c, s
}

// URI satisfies the Resource interface
func (f *File) URI() string {
	_, s, _ := getCid(f.Value)
	return fmt.Sprintf("dweb:/ipfs/%s", s)
}

// ResourceType is an enum for resource types
type ResourceType uint8

const (
	// PackageType the ResourceType for Packages
	PackageType ResourceType = iota
	// MessageType the ResourceType for Messages
	MessageType
	// FileType is the ResourceType for Files
	FileType
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
func NewPackage(ctx context.Context, t time.Time, pathname, resource string) *Package {
	dateTime := t.Format(time.RFC3339)

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

// Paths is a convenience method for getting the path.Resolved version
// of a packages ID and Value CIDs at the same time.
func (p *Package) Paths() (path.Resolved, path.Resolved, error) {
	id, err := cid.Cast(p.Id)
	if err != nil {
		return nil, nil, err
	}
	value, err := cid.Cast(p.Value)
	if err != nil {
		return nil, nil, err
	}
	return path.IpfsPath(id), path.IpfsPath(value), nil
}

// NQuads converts the Package to a slice of ld.*Quads
func (p *Package) NQuads(pathname string, txn *badger.Txn) ([]*ld.Quad, error) {
	doc := make([]*ld.Quad, len(base), len(base)+6+len(p.Member)*2)
	copy(doc, base)

	_, s, err := getCid(p.Value)
	if err != nil {
		return nil, err
	}
	value := ld.NewIRI(fmt.Sprintf("dweb:/ipfs/%s", s))
	extent := strconv.FormatUint(p.Extent, 10)
	doc = append(doc,
		ld.NewQuad(subject, membershipResourceIri, ld.NewIRI(p.Resource), ""),
		ld.NewQuad(subject, valueIri, value, ""),
		ld.NewQuad(value, extentIri, ld.NewLiteral(extent, ld.XSDInteger, ""), ""),
		ld.NewQuad(subject, createdIri, ld.NewLiteral(p.Created, dateTime, ""), ""),
		ld.NewQuad(subject, modifiedIri, ld.NewLiteral(p.Modified, dateTime, ""), ""),
	)

	if p.RevisionOf != nil && p.RevisionOfSubject != "" {
		_, r, err := getCid(p.RevisionOf)
		if err != nil {
			return nil, err
		}
		object := ld.NewIRI(fmt.Sprintf("ul:/ipfs/%s#%s", r, p.RevisionOfSubject))
		doc = append(doc, ld.NewQuad(subject, wasRevisionOfIri, object, ""))
	}

	for _, name := range p.Member {
		var key string
		if pathname == "/" {
			key = "/" + name
		} else {
			key = fmt.Sprintf("%s/%s", pathname, name)
		}

		resource, _, err := GetResource(key, txn)
		if err != nil {
			return nil, err
		}

		switch t := resource.(type) {
		case *Package:
			_, s, err := getCid(t.Id)
			if err != nil {
				return nil, err
			}
			uri := ld.NewIRI(fmt.Sprintf("ul:/ipfs/%s#%s", s, t.Subject))
			doc = append(doc,
				ld.NewQuad(subject, hadMemberIri, uri, ""),
				ld.NewQuad(uri, membershipResourceIri, ld.NewIRI(t.Resource), ""),
			)
		case Message:
			_, s, err := getCid(t)
			if err != nil {
				return nil, err
			}
			member := ld.NewIRI("ul:/ipfs/" + s)
			doc = append(doc, ld.NewQuad(subject, hadMemberIri, member, ""))
			if s != name {
				resource := fmt.Sprintf("%s/%s", p.Resource, name)
				doc = append(doc, ld.NewQuad(subject, membershipResourceIri, ld.NewIRI(resource), ""))
			}
		case *File:
			_, s, err := getCid(t.Value)
			if err != nil {
				return nil, err
			}

			member := ld.NewIRI("dweb:/ipfs/" + s)
			extent := strconv.FormatUint(t.Extent, 10)
			doc = append(doc,
				ld.NewQuad(subject, hadMemberIri, member, ""),
				ld.NewQuad(member, extentIri, ld.NewLiteral(extent, ld.XSDInteger, ""), ""),
				ld.NewQuad(member, formatIri, ld.NewLiteral(t.Format, ld.XSDString, ""), ""),
			)
			if s != name {
				resource := fmt.Sprintf("%s/%s", p.Resource, name)
				doc = append(doc, ld.NewQuad(member, membershipResourceIri, ld.NewIRI(resource), ""))
			}
		}
	}

	return doc, nil
}
