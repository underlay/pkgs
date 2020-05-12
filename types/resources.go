package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	cid "github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	iface "github.com/ipfs/interface-go-ipfs-core"
	path "github.com/ipfs/interface-go-ipfs-core/path"
	rdf "github.com/underlay/go-rdfjs"
	"golang.org/x/net/context"
)

var EmptyDirectoryURI = "dweb:/ipfs/bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354"

var PackageURIPattern = regexp.MustCompile("^ul:([a-z2-7]{59})#(c14n\\d+)$")
var AssertionURIPattern = regexp.MustCompile("^ul:([a-z2-7]{59})$")
var FileURIPattern = regexp.MustCompile("^dweb:/ipfs/([a-z2-7]{59})$")

// ResourceType is an enum for resource types
type ResourceType uint8

const (
	_ ResourceType = iota
	// PackageType = 1 the ResourceType for Packages
	PackageType
	// AssertionType = 2 the ResourceType for Assertions
	AssertionType
	// FileType = 3 is the ResourceType for Files
	FileType
)

type Resource interface {
	T() ResourceType
	Type() string
	Path() path.Resolved
	ETag() string
	URI() string
	Name() string
}

type Reference struct {
	ID       string `json:"id,omitempty"`
	Resource string `json:"resource,omitempty"`
	Title    string `json:"title,omitempty"`
}

func (r *Reference) T() ResourceType { return PackageType }
func (r *Reference) Type() string    { return LDPDirectContainer }
func (r *Reference) URI() string     { return r.ID }

func (r *Reference) parseID() (id string, fragment string) {
	match := PackageURIPattern.FindStringSubmatch(r.ID)
	if match != nil {
		id, fragment = match[1], match[2]
	}
	return
}

func (r *Reference) Path() path.Resolved {
	id, _ := r.parseID()
	c, _ := cid.Decode(id)
	return path.IpfsPath(c)
}

func (r *Reference) ETag() string {
	id, _ := r.parseID()
	return "\"" + id + "\""
}

func (r *Reference) Name() string { return r.Title }

func (r *Reference) Fragment() string {
	match := PackageURIPattern.FindStringSubmatch(r.ID)
	if match != nil {
		return match[1]
	}
	return ""
}

type Assertion struct {
	ID       string      `json:"id,omitempty"`
	Resource string      `json:"resource,omitempty"`
	Title    string      `json:"title,omitempty"`
	Created  string      `json:"created,omitempty"`
	Modified string      `json:"modified,omitempty"`
	Dataset  []*rdf.Quad `json:"-"`
}

func (a *Assertion) T() ResourceType { return AssertionType }
func (a *Assertion) Type() string    { return LDPRDFSource }
func (a *Assertion) URI() string     { return a.ID }
func (a *Assertion) Path() path.Resolved {
	match := AssertionURIPattern.FindStringSubmatch(a.ID)
	if match != nil {
		c, _ := cid.Decode(match[1])
		return path.IpfsPath(c)
	}
	return nil
}

func (a *Assertion) GetDataset(api iface.CoreAPI) []*rdf.Quad {
	if a.Dataset != nil {
		return a.Dataset
	}

	ctx := context.Background()
	node, err := api.Unixfs().Get(ctx, a.Path())
	if err != nil {
		log.Println(err)
		return nil
	}
	quads, err := rdf.ReadQuads(files.ToFile(node))
	if err != nil {
		log.Println(err)
		return nil
	}
	a.Dataset = quads
	return quads
}

func (a *Assertion) ETag() string {
	match := AssertionURIPattern.FindStringSubmatch(a.ID)
	if match != nil {
		return "\"" + match[1] + "\""
	}
	return ""
}

func (a *Assertion) Name() string {
	if a.Resource != "" && a.Title != "" {
		return a.Title
	}
	match := AssertionURIPattern.FindStringSubmatch(a.ID)
	if match != nil {
		return match[1]
	}
	return ""
}

var NQuadsFileExtension = ".nq"

type File struct {
	ID       string `json:"id,omitempty"`
	Resource string `json:"resource,omitempty"`
	Title    string `json:"title,omitempty"`
	Created  string `json:"created,omitempty"`
	Modified string `json:"modified,omitempty"`
	Extent   int    `json:"extent"`
	Format   string `json:"format"`
}

func (f *File) T() ResourceType { return FileType }
func (f *File) Type() string    { return LDPNonRDFSource }
func (f *File) URI() string     { return f.ID }
func (f *File) Path() path.Resolved {
	match := FileURIPattern.FindStringSubmatch(f.ID)
	if match != nil {
		c, _ := cid.Decode(match[1])
		return path.IpfsPath(c)
	}
	return nil
}

func (f *File) ETag() string {
	match := FileURIPattern.FindStringSubmatch(f.ID)
	if match != nil {
		return "\"" + match[1] + "\""
	}
	return ""
}

func (f *File) Name() string {
	if f.Resource != "" && f.Title != "" {
		return f.Title
	}
	match := FileURIPattern.FindStringSubmatch(f.ID)
	if match != nil {
		return match[1]
	}
	return ""
}

type Package struct {
	Reference
	Created     string   `json:"created,omitempty"`
	Modified    string   `json:"modified,omitempty"`
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Parent      string   `json:"parent,omitempty"`
	Value       struct {
		ID     string `json:"id"`
		Extent int    `json:"extent"`
	} `json:"value"`
	Members struct {
		Packages   []*Reference `json:"packages,omitempty"`
		Assertions []*Assertion `json:"assertions,omitempty"`
		Files      []*File      `json:"files,omitempty"`
	} `json:"members,omitempty"`
}

// NewPackage creates a new timestamped package.
func NewPackage(resource, title string) *Package {
	created := time.Now().Format(time.RFC3339)
	pkg := &Package{Created: created, Modified: created}
	pkg.Resource = resource
	pkg.Title = title
	pkg.Value.ID = EmptyDirectoryURI
	pkg.Value.Extent = 4
	return pkg
}

func (pkg *Package) T() ResourceType     { return PackageType }
func (pkg *Package) Type() string        { return LDPDirectContainer }
func (pkg *Package) URI() string         { return pkg.ID }
func (pkg *Package) Path() path.Resolved { return pkg.Reference.Path() }
func (pkg *Package) ETag() string        { return pkg.Reference.ETag() }
func (pkg *Package) Name() string        { return pkg.Title }

func (pkg *Package) SearchPackages(name string, isCid bool) (int, *Reference) {
	l := len(pkg.Members.Packages)
	f := func(i int) bool { return name <= pkg.Members.Packages[i].Title }
	i := sort.Search(l, f)
	if i < l && pkg.Members.Packages[i].Title == name {
		return i, pkg.Members.Packages[i]
	}
	return i, nil
}

func (pkg *Package) SearchAssertions(name string, isCid bool) (int, *Assertion) {
	l := len(pkg.Members.Assertions)
	f := func(i int) bool {
		assertion := pkg.Members.Assertions[i]
		if (assertion.Resource == "") == isCid {
			return name <= assertion.Name()
		}
		return isCid
	}
	i := sort.Search(l, f)
	if i < l && pkg.Members.Assertions[i].Name() == name {
		return i, pkg.Members.Assertions[i]
	}
	return i, nil
}

func (pkg *Package) SearchFiles(name string, isCid bool) (int, *File) {
	l := len(pkg.Members.Files)
	f := func(i int) bool {
		file := pkg.Members.Files[i]
		if (file.Resource == "") == isCid {
			return name <= file.Name()
		}
		return isCid
	}
	i := sort.Search(l, f)
	if i < l && pkg.Members.Files[i].Name() == name {
		return i, pkg.Members.Files[i]
	}
	return i, nil
}

func (pkg *Package) CopyResource() *Reference {
	return &Reference{pkg.ID, pkg.Resource, pkg.Title}
}

func (pkg *Package) Fragment() (c cid.Cid, fragment string) {
	match := PackageURIPattern.FindStringSubmatch(pkg.Reference.ID)
	if match != nil {
		c, _ = cid.Decode(match[1])
		fragment = match[2]
	}
	return
}

func (pkg *Package) ValuePath() path.Resolved {
	match := FileURIPattern.FindStringSubmatch(pkg.Value.ID)
	if match != nil {
		c, _ := cid.Decode(match[1])
		return path.IpfsPath(c)
	}
	return nil
}

func (pkg *Package) JsonLd(context string) (map[string]interface{}, error) {
	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(pkg)
	if err != nil {
		return nil, err
	}

	var doc map[string]interface{}

	err = json.NewDecoder(buf).Decode(&doc)
	if err != nil {
		return nil, err
	}

	doc["@context"] = context
	doc["@type"] = []interface{}{"ldp:DirectContainer", "prov:Collection"}
	doc["ldp:hasMemberRelation"] = map[string]interface{}{"@id": "prov:hadMember"}
	if _, has := doc["id"]; has {
		delete(doc, "id")
	}

	return doc, nil
}

func MakeLinkType(t string) string { return fmt.Sprintf(`<%s>; rel="type"`, t) }

const (
	LDPResource        = "http://www.w3.org/ns/ldp#Resource"
	LDPDirectContainer = "http://www.w3.org/ns/ldp#DirectContainer"
	LDPRDFSource       = "http://www.w3.org/ns/ldp#RDFSource"
	LDPNonRDFSource    = "http://www.w3.org/ns/ldp#NonRDFSource"
)

var LinkTypeResource = MakeLinkType(LDPResource)
var LinkTypeDirectContainer = MakeLinkType(LDPDirectContainer)
var LinkTypeRDFSource = MakeLinkType(LDPRDFSource)
var LinkTypeNonRDFSource = MakeLinkType(LDPNonRDFSource)

func ParseLinks(links []string) (self string, t ResourceType) {
	// var isResource bool
	for _, link := range links {
		if link == LinkTypeResource {
			// isResource = true
		} else if link == LinkTypeDirectContainer {
			t |= PackageType
		} else if link == LinkTypeRDFSource {
			t |= AssertionType
		} else if link == LinkTypeNonRDFSource {
			t |= FileType
		} else if linkSelfPattern.MatchString(link) {
			match := linkSelfPattern.FindStringSubmatch(link)
			self = match[1]
		}
	}
	return
}

var linkSelfPattern = regexp.MustCompile(`^<([^<>; \t]+)>; rel="self"$`)

func ParsePath(p string) []string {
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimSuffix(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

func GetURI(base string, key []string) (u string) {
	u = base
	for _, name := range key {
		u += "/" + name
	}
	return
}
