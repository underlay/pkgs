package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	"github.com/gogo/protobuf/proto"
	cid "github.com/ipfs/go-cid"
	ipfs "github.com/ipfs/go-ipfs-api"
	ld "github.com/piprate/json-gold/ld"
)

const defaultSubject = "_:c14n0"
const emptyDirectory = "bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354"

const dateTime = "http://www.w3.org/2001/XMLSchema#dateTime"

var emptyDirectoryCID, _ = cid.Decode(emptyDirectory)
var emptyDirectoryURI = fmt.Sprintf("dweb:/ipfs/%s", emptyDirectory)
var subject = ld.NewBlankNode("_:b0")
var packageIri = ld.NewIRI("http://underlay.mit.edu/ns#Package")
var valueIri = ld.NewIRI("http://www.w3.org/ns/prov#value")
var extentIri = ld.NewIRI("http://www.w3.org/ns/prov#extent")
var formatIri = ld.NewIRI("http://www.w3.org/ns/prov#format")
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

func NewPackage(path, resource string, sh *ipfs.Shell) (*Package, error) {
	dateTime := time.Now().Format(time.RFC3339)

	pkg := &Package{
		Resource: resource,
		Subject:  defaultSubject,
		Value:    emptyDirectory,
		Extent:   0,
		Created:  dateTime,
		Modified: dateTime,
		Members:  make([]string, 0),
	}

	normalized, err := pkg.Normalize(path, nil)
	if err != nil {
		return nil, err
	}

	reader := strings.NewReader(normalized)
	pkg.Id, err = sh.Add(reader, ipfs.Pin(true), ipfs.RawLeaves(true), ipfs.CidVersion(1))
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

// func UnmarshalRoot(reader io.Reader, sh *ipfs.Shell) (p *Package, err error) {
// 	doc := map[string]interface{}{}
// 	decoder := json.NewDecoder(reader)
// 	if err = decoder.Decode(&doc); err != nil {
// 		return
// 	}

// 	p = &Package{
// 		Members: make([]string, 0),
// 	}

// 	id, _ := doc["@id"].(string)
// 	packageMatch := packageURI.FindStringSubmatch(id)
// 	if packageMatch != nil {
// 		p.Id, p.Subject = packageMatch[1], packageMatch[2]
// 	} else {
// 		err = fmt.Errorf("Invalid package @id: %v", doc["@id"])
// 		return
// 	}

// 	p.Resource, _ = doc["ldp:membershipResource"].(string)

// 	value, is := doc["prov:value"].(map[string]interface{})
// 	if !is {
// 		err = fmt.Errorf("Invalid package prov:value: %v", doc["prov:value"])
// 		return
// 	}

// 	valueID, _ := value["@id"].(string)
// 	valueMatch := fileURI.FindStringSubmatch(valueID)
// 	if valueMatch == nil {
// 		err = fmt.Errorf("Invalid package prov:value/@id: %v", doc["prov:value"])
// 		return
// 	}

// 	p.Value = valueMatch[1]

// 	p.Extent, is = value["dcterms:extent"].(uint64)
// 	if !is {
// 		err = fmt.Errorf("Invalid package prov:value/dcterms:extent: %v", value["dcterms:extent"])
// 	}

// 	p.Value = valueMatch[1]

// 	created, _ := doc["dcterms:created"].(string)
// 	modified, _ := doc["dcterms:modified"].(string)
// 	p.Created, p.Modified = created, modified

// 	members, _ := doc["prov:hadMember"].([]interface{})
// 	for _, element := range members {
// 		member, _ := element.(map[string]interface{})
// 		key, _ := member["@id"].(string)
// 		val, _ := member["ldp:membershipResource"].(string)

// 		u, err := url.Parse(val)
// 		if err != nil {
// 			return nil, err
// 		}

// 		tail := strings.LastIndex(u.Path, "/") + 1
// 		name := u.Path[tail:]

// 		packageMatch := packageURI.FindStringSubmatch(key)
// 		if packageMatch != nil {
// 			p.Packages[name], err = LoadPackage(packageMatch[1], packageMatch[2], val, sh)
// 			if err != nil {
// 				return nil, err
// 			}
// 			continue
// 		}

// 		messageMatch := messageURI.FindStringSubmatch(key)
// 		if messageMatch != nil {
// 			p.Messages[name] = messageMatch[1]
// 			continue
// 		}

// 		fileMatch := fileURI.FindStringSubmatch(key)
// 		if fileMatch != nil {
// 			extent, _ := member["dcterms:extent"].(uint64)
// 			format, _ := member["dcterms:format"].(string)
// 			p.Files[name] = &File{Value: fileMatch[1], Format: format, Extent: extent}
// 			continue
// 		}
// 	}

// 	return p, nil
// }

func (pkg *Package) Normalize(path string, txn *badger.Txn) (s string, err error) {
	ds := ld.NewRDFDataset()
	ds.Graphs["@default"], err = pkg.NQuads(path, txn)
	if err != nil {
		return
	}

	api := ld.NewJsonLdApi()
	opts := ld.NewJsonLdOptions("")
	opts.Format = "application/n-quads"
	res, err := api.Normalize(ds, opts)
	if err != nil {
		return
	}

	return res.(string), nil
}

// NQuads converts the Package to a slice of ld.*Quads
func (pkg *Package) NQuads(path string, txn *badger.Txn) ([]*ld.Quad, error) {
	doc := make([]*ld.Quad, len(base), len(base)+7+len(pkg.Members)*2)
	copy(doc, base)
	value := fmt.Sprintf("dweb:/ipfs/%s", pkg.Value)
	extent := strconv.FormatUint(pkg.Extent, 10)
	doc = append(doc,
		ld.NewQuad(subject, typeIri, packageIri, ""),
		ld.NewQuad(subject, hasMemberRelationIri, hadMemberIri, ""),
		ld.NewQuad(subject, membershipResourceIri, ld.NewIRI(pkg.Resource), ""),
		ld.NewQuad(subject, valueIri, ld.NewIRI(value), ""),
		ld.NewQuad(subject, extentIri, ld.NewLiteral(extent, ld.XSDInteger, ""), ""),
		ld.NewQuad(subject, createdIri, ld.NewLiteral(pkg.Created, dateTime, ""), ""),
		ld.NewQuad(subject, modifiedIri, ld.NewLiteral(pkg.Modified, dateTime, ""), ""),
	)

	for _, name := range pkg.Members {
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
			resource := ld.NewIRI(p.Resource)
			uri := ld.NewIRI(fmt.Sprintf("ul:/ipfs/%s#%s", p.Id, p.Subject))
			doc = append(doc,
				ld.NewQuad(subject, hadMemberIri, uri, ""),
				ld.NewQuad(uri, membershipResourceIri, resource, ""),
			)
		} else if m != "" {
			member := ld.NewIRI("ul:/ipfs/" + m)
			doc = append(doc, ld.NewQuad(subject, hadMemberIri, member, ""))
			if m != name {
				resource := fmt.Sprintf("%s/%s", pkg.Resource, name)
				doc = append(doc, ld.NewQuad(subject, membershipResourceIri, ld.NewIRI(resource), ""))
			}
		} else if f != nil {
			member := ld.NewIRI("dweb:/ipfs/" + f.Value)
			extent := strconv.FormatUint(f.Extent, 10)
			doc = append(doc,
				ld.NewQuad(subject, hadMemberIri, member, ""),
				ld.NewQuad(member, extentIri, ld.NewLiteral(extent, ld.XSDInteger, ""), ""),
				ld.NewQuad(member, formatIri, ld.NewLiteral(f.Format, ld.XSDString, ""), ""),
			)
			if f.Value != name {
				resource := fmt.Sprintf("%s/%s", pkg.Resource, name)
				doc = append(doc, ld.NewQuad(subject, membershipResourceIri, ld.NewIRI(resource), ""))
			}
		}
	}

	return doc, nil
}

// JSON converts the Package to a JSON-LD document
func (pkg *Package) JSON(path string, txn *badger.Txn) (map[string]interface{}, error) {
	members := make([]map[string]interface{}, 0, len(pkg.Members))

	for _, name := range pkg.Members {
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
		} else if m != "" {
			member := map[string]interface{}{"@id": "ul:/ipfs/%s" + m}
			if name != m {
				member["ldp:membershipResource"] = fmt.Sprintf("%s/%s", pkg.Resource, name)
			}
			members = append(members, member)
		} else if f != nil {
			member := map[string]interface{}{
				"@id":            "dweb:/ipfs/%s" + f.Value,
				"dcterms:extent": f.Extent,
				"dcterms:format": f.Format,
			}

			if name != f.Value {
				member["ldp:membershipResource"] = fmt.Sprintf("%s/%s", pkg.Resource, name)
			}

			members = append(members, member)
		}
	}

	return map[string]interface{}{
		"@context":               contextURL,
		"@type":                  packageIri.Value,
		"ldp:hasMemberRelation":  "prov:hadMember",
		"ldp:membershipResource": pkg.Resource,
		"dcterms:created":        pkg.Created,
		"dcterms:modified":       pkg.Modified,
		"prov:value": map[string]interface{}{
			"@id":            fmt.Sprintf("dweb:/ipfs/%s", pkg.Value),
			"dcterms:extent": strconv.FormatUint(pkg.Extent, 10),
		},
		"prov:hadMember": members,
	}, nil
}

// var frame = map[string]interface{}{
// 	"@context": contextURL,
// 	"@type":    packageIri.Value,
// }

// func LoadPackage(root, subject, resource string, sh *ipfs.Shell) (*Package, error) {
// 	proc := ld.NewJsonLdProcessor()
// 	opts := ld.NewJsonLdOptions("")
// 	opts.Format = "application/n-quads"
// 	opts.DocumentLoader = loader.NewHTTPDocumentLoader(sh)

// 	reader, err := sh.Cat(root)
// 	if err != nil {
// 		return nil, err
// 	}

// 	doc, err := proc.FromRDF(reader, opts)
// 	if err != nil {
// 		return nil, err
// 	}

// 	framed, err := proc.Frame(doc, frame, opts)
// 	if err != nil {
// 		return nil, err
// 	}

// 	log.Println(framed)

// 	return nil, nil
// }
