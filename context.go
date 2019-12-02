package main

const contextURL = "ipfs://bafkreiarfiphrufhuu2pm5uzykmyy6d63tf3wffdqlvkthqhxb4ipkjlau"

const context = `{
	"@context": {
    "dcterms": "http://purl.org/dc/terms/",
		"prov": "http://www.w3.org/ns/prov#",
		"ldp": "http://www.w3.org/ns/ldp#",
    "xsd": "http://www.w3.org/2001/XMLSchema#",
    "dcterms:created": {
      "@type": "xsd:dateTime"
    },
    "dcterms:modified": {
      "@type": "xsd:dateTime"
    },
		"prov:value": {
      "@type": "@id"
    },
		"ldp:membershipResource": {
      "@type": "@id"
    },
		"ldp:hasMemberRelation": {
      "@type": "@id"
    }
	}
}
`
