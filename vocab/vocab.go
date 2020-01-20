package vocab

import (
	ld "github.com/underlay/json-gold/ld"
)

var PROVwasRevisionOf = ld.NewIRI("http://www.w3.org/ns/prov#wasRevisionOf")
var PROVhadMember = ld.NewIRI("http://www.w3.org/ns/prov#hadMember")
var PROVvalue = ld.NewIRI("http://www.w3.org/ns/prov#value")

var LDPmembershipResource = ld.NewIRI("http://www.w3.org/ns/ldp#membershipResource")
var LDPhasMemberRelation = ld.NewIRI("http://www.w3.org/ns/ldp#hasMemberRelation")

var DCTERMStitle = ld.NewIRI("http://purl.org/dc/terms/title")
var DCTERMSextent = ld.NewIRI("http://purl.org/dc/terms/extent")
var DCTERMSformat = ld.NewIRI("http://purl.org/dc/terms/format")
var DCTERMSdescription = ld.NewIRI("http://purl.org/dc/terms/description")
var DCTERMSsubject = ld.NewIRI("http://purl.org/dc/terms/subject")
var DCTERMSmodified = ld.NewIRI("http://purl.org/dc/terms/modified")
var DCTERMScreated = ld.NewIRI("http://purl.org/dc/terms/created")

const XSDdateTime = "http://www.w3.org/2001/XMLSchema#dateTime"

func MakeURI(root string, fragment string) string {
	return "u:" + root + fragment
}
