package types

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	cid "github.com/ipfs/go-cid"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	ld "github.com/underlay/json-gold/ld"

	query "github.com/underlay/pkgs/query"
	v "github.com/underlay/pkgs/vocab"
)

// A Message is just the bytes of a CID
type Message []byte

// Type exposes the message resource type
func (m Message) Type() query.ResourceType {
	return query.MessageType
}

// ETag satisfies the Resource interface
func (m Message) ETag() (cid.Cid, string) {
	c, s, _ := getCid(m)
	return c, s
}

// URI satisfies the Resource interface
func (m Message) URI() string {
	_, s, _ := getCid(m)
	return v.MakeURI(s, "")
}

// Type exposes the package resource type
func (p *Package) Type() query.ResourceType {
	return query.PackageType
}

// ETag satisfies the Resource interface for Packages
func (p *Package) ETag() (cid.Cid, string) {
	c, s, _ := getCid(p.Id)
	return c, s
}

// URI satisfies the Resource interface
func (p *Package) URI() string {
	_, s, _ := getCid(p.Id)
	return v.MakeURI(s, "#"+p.Subject)
}

// Type exposes the file resource type
func (f *File) Type() query.ResourceType {
	return query.FileType
}

// ETag satisfies the Resource interface for Files
func (f *File) ETag() (cid.Cid, string) {
	c, s, _ := getCid(f.Value)
	return c, s
}

// URI satisfies the Resource interface
func (f *File) URI() string {
	_, s, _ := getCid(f.Value)
	return "dweb:/ipfs/" + s
}

const defaultSubject = "_:c14n0"

// EmptyDirectory is the CIDv1 of the empty directory
// By default, IPFS nodes only pin the CIDv0 empty directory,
// so we pin the v1 maually on initialization.
const EmptyDirectory = "bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354"

// PackageIri is an rdf:type for Underlay Packages
var PackageIri = ld.NewIRI("http://underlay.mit.edu/ns#Package")

// EmptyDirectoryCID is the CID instance of EmptyDirectory
var EmptyDirectoryCID, _ = cid.Decode(EmptyDirectory)

var subject = ld.NewBlankNode("_:b0")

var typeIri = ld.NewIRI(ld.RDFType)

var base32 = regexp.MustCompile("^[a-z2-7]{59}$")
var fileURI = regexp.MustCompile("^dweb:/ipfs/([a-z2-7]{59})$")
var messageURI = regexp.MustCompile("^ul:([a-z2-7]{59})$")
var packageURI = regexp.MustCompile("^ul:([a-z2-7]{59})#(_:c14n\\d+)$")

var base = []*ld.Quad{
	ld.NewQuad(subject, typeIri, PackageIri, ""),
	ld.NewQuad(subject, v.LDPhasMemberRelation, v.PROVhadMember, ""),
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
func (p *Package) Paths() (id path.Resolved, value path.Resolved, err error) {
	cidID, err := cid.Cast(p.Id)
	if err != nil {
		return nil, nil, err
	}
	cidValue, err := cid.Cast(p.Value)
	if err != nil {
		return nil, nil, err
	}
	return path.IpfsPath(cidID), path.IpfsPath(cidValue), nil
}

// ValueURI returns the dweb URI for the package value
func (p *Package) ValueURI() string {
	_, s, _ := getCid(p.Value)
	return "dweb:/ipfs/" + s
}

var printLayout = "Mon, 02 Jan 2006 15:04:05 -0700"

// PrintModified formats p.Modified as printLayout
func (p *Package) PrintModified() string {
	t, _ := time.Parse(time.RFC3339, p.Modified)
	return t.Format(printLayout)
}

// PrintCreated formats p.Created as printLayout
func (p *Package) PrintCreated() string {
	t, _ := time.Parse(time.RFC3339, p.Created)
	return t.Format(printLayout)
}

// NQuads converts the Package to a slice of ld.*Quads
func (p *Package) NQuads(pathname string, txn *badger.Txn) ([]*ld.Quad, error) {
	doc := make([]*ld.Quad, len(base), len(base)+6+len(p.Member))
	copy(doc, base)

	tail := strings.LastIndex(p.Resource, "/")
	title := ld.NewLiteral(p.Resource[tail+1:], "", "")

	_, s, err := getCid(p.Value)
	if err != nil {
		return nil, err
	}

	value := ld.NewIRI("dweb:/ipfs/" + s)
	extent := strconv.FormatUint(p.Extent, 10)
	doc = append(doc,
		ld.NewQuad(subject, v.DCTERMStitle, title, ""),
		ld.NewQuad(subject, v.LDPmembershipResource, ld.NewIRI(p.Resource), ""),
		ld.NewQuad(subject, v.PROVvalue, value, ""),
		ld.NewQuad(value, v.DCTERMSextent, ld.NewLiteral(extent, ld.XSDInteger, ""), ""),
		ld.NewQuad(subject, v.DCTERMScreated, ld.NewLiteral(p.Created, v.XSDdateTime, ""), ""),
		ld.NewQuad(subject, v.DCTERMSmodified, ld.NewLiteral(p.Modified, v.XSDdateTime, ""), ""),
	)

	if p.RevisionOf != nil && p.RevisionOfSubject != "" {
		_, r, err := getCid(p.RevisionOf)
		if err != nil {
			return nil, err
		}
		object := ld.NewIRI(v.MakeURI(r, "#"+p.RevisionOfSubject))
		doc = append(doc, ld.NewQuad(subject, v.PROVwasRevisionOf, object, ""))
	}

	if p.Description != "" {
		description := ld.NewLiteral(p.Description, "", "")
		doc = append(doc, ld.NewQuad(subject, v.DCTERMSdescription, description, ""))
	}

	if len(p.Keyword) > 0 {
		keywords := make([]*ld.Quad, len(p.Keyword))
		for i, keyword := range p.Keyword {
			object := ld.NewLiteral(keyword, "", "")
			keywords[i] = ld.NewQuad(subject, v.DCTERMSsubject, object, "")
		}
		doc = append(doc, keywords...)
	}

	for _, name := range p.Member {
		key := "/" + name
		if pathname != "/" {
			key = pathname + key
		}

		resource, err := GetResource(key, txn)
		if err != nil {
			return nil, err
		}

		switch t := resource.(type) {
		case *Package:
			_, s, err := getCid(t.Id)
			if err != nil {
				return nil, err
			}

			member := ld.NewIRI(v.MakeURI(s, "#"+t.Subject))
			uri := ld.NewIRI(t.Resource)
			doc = append(doc,
				ld.NewQuad(subject, v.PROVhadMember, member, ""),
				ld.NewQuad(member, v.LDPmembershipResource, uri, ""),
				ld.NewQuad(member, v.DCTERMStitle, ld.NewLiteral(name, "", ""), ""),
			)
		case Message:
			_, s, err := getCid(t)
			if err != nil {
				return nil, err
			}
			member := ld.NewIRI(v.MakeURI(s, ""))
			doc = append(doc, ld.NewQuad(subject, v.PROVhadMember, member, ""))
			if s != name {
				uri := ld.NewIRI(p.Resource + "/" + name)
				doc = append(
					doc,
					ld.NewQuad(member, v.LDPmembershipResource, uri, ""),
					ld.NewQuad(member, v.DCTERMStitle, ld.NewLiteral(name, "", ""), ""),
				)
			}
		case *File:
			_, s, err := getCid(t.Value)
			if err != nil {
				return nil, err
			}

			member := ld.NewIRI("dweb:/ipfs/" + s)
			extent := strconv.FormatUint(t.Extent, 10)
			doc = append(doc,
				ld.NewQuad(subject, v.PROVhadMember, member, ""),
				ld.NewQuad(member, v.DCTERMSextent, ld.NewLiteral(extent, ld.XSDInteger, ""), ""),
				ld.NewQuad(member, v.DCTERMSformat, ld.NewLiteral(t.Format, ld.XSDString, ""), ""),
			)
			if s != name {
				uri := ld.NewIRI(p.Resource + "/" + name)
				doc = append(
					doc,
					ld.NewQuad(member, v.LDPmembershipResource, uri, ""),
					ld.NewQuad(member, v.DCTERMStitle, ld.NewLiteral(name, "", ""), ""),
				)
			}
		}
	}

	return doc, nil
}
