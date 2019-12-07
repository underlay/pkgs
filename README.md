# pkgs

## Usage

If you have a file `8-cell-orig.gif`, you can store it in your index package with:

```
% curl -i -X PUT -T 8-cell-orig.gif -H 'Link: <http://www.w3.org/ns/ldp#Resource>; rel="type"' -H 'Link: <http://www.w3.org/ns/ldp#NonRDFSource>; rel="type"' -H 'Content-Type: image/gif' http://localhost:8086/8-cell-orig.gif
HTTP/1.1 100 Continue

HTTP/1.1 200 OK
Date: Wed, 04 Dec 2019 21:19:55 GMT
Content-Length: 0

```

And then look at its properties with HEAD:

```

```

### GET

`GET` requests to a resource _require_ an explicit `Accpet` header of either `application/ld+json` or `application/n-quads`.

```
% curl -i -H 'Accept: application/n-quads' http://localhost:8086/
HTTP/1.1 200 OK
Content-Type: application/n-quads
Etag: bafkreifjc7gebvrm3jbsdjobgpshcfo5twx2suyercykybcrvtwpy5angu
Link: <http://www.w3.org/ns/ldp#Resource>; rel="type"
Link: <http://www.w3.org/ns/ldp#DirectContainer>; rel="type"
Link: <#_:c14n0>; rel="self"
Date: Thu, 05 Dec 2019 21:51:24 GMT
Content-Length: 817

<dweb:/ipfs/bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354> <http://purl.org/dc/terms/extent> "4"^^<http://www.w3.org/2001/XMLSchema#integer> .
_:c14n0 <http://purl.org/dc/terms/created> "2019-12-05T10:00:22-05:00"^^<http://www.w3.org/2001/XMLSchema#dateTime> .
_:c14n0 <http://purl.org/dc/terms/modified> "2019-12-05T10:00:22-05:00"^^<http://www.w3.org/2001/XMLSchema#dateTime> .
_:c14n0 <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://underlay.mit.edu/ns#Package> .
_:c14n0 <http://www.w3.org/ns/ldp#hasMemberRelation> <http://www.w3.org/ns/prov#hadMember> .
_:c14n0 <http://www.w3.org/ns/ldp#membershipResource> <dweb:/ipns/QmXS2hw3KjFC19uSzYJwXn5Fp5GRjueEpBLyQheuSMLs1D> .
_:c14n0 <http://www.w3.org/ns/prov#value> <dweb:/ipfs/bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354> .
```

Note the three `Link` response headers: the first declares that the URL is an LDP Resource, the second further specifies that the resource is a LDP Direct Container, and the last links to the local blank node that represents the URL in the attached RDF dataset.

Requests at package paths with `Accept: application/ld+json` will be framed and compacted with a default package frame:

```
% curl -H 'Accept: application/ld+json' http://localhost:8086 | jq
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100   459  100   459    0     0   112k      0 --:--:-- --:--:-- --:--:--  112k
{
  "@context": "ipfs://bafkreifcqgsljpst2fabpvmlzcf5fqdthzvhf4imvqvnymk5iifi6mdtru",
  "@type": "http://underlay.mit.edu/ns#Package",
  "dcterms:created": "2019-12-05T10:00:22-05:00",
  "dcterms:modified": "2019-12-05T10:00:22-05:00",
  "ldp:hasMemberRelation": "prov:hadMember",
  "ldp:membershipResource": "dweb:/ipns/QmXS2hw3KjFC19uSzYJwXn5Fp5GRjueEpBLyQheuSMLs1D",
  "prov:value": {
    "@id": "dweb:/ipfs/bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354",
    "dcterms:extent": 4
  }
}
```

The remote context reference here - `ipfs://bafkreifcqgsljpst2fabpvmlzcf5fqdthzvhf4imvqvnymk5iifi6mdtru` - is from [types/utils.go](types/utils.go) and it's added as a file `/context.jsonld` to the root package.

### HEAD

```
% curl -i -I http://localhost:8086/8-cell-orig.gif
HTTP/1.1 200 OK
Content-Length: 640580
Content-Type: image/gif
Etag: bafybeiatr6vzozvaxtp5f32ghixj4bvauz6wgl4lbbh6np4yrrsvtep3y4
Link: <http://www.w3.org/ns/ldp#Resource>; rel="type"
Link: <http://www.w3.org/ns/ldp#NonRDFSource>; rel="type"
Date: Sat, 07 Dec 2019 07:22:01 GMT

```

### POST

### PUT

You need to give three headers for `PUT` requests: two `Link` headers and one `Content-Type`. The first `Link` has to be `<http://www.w3.org/ns/ldp#Resource>; rel="type"`, and the second is one of:

- `<http://www.w3.org/ns/ldp#DirectContainer>; rel="type"`
- `<http://www.w3.org/ns/ldp#RDFSource>; rel="type"`
- `<http://www.w3.org/ns/ldp#NonRDFSource>; rel="type"`

for packages, messages, and files, respectively.

`Content-Type` must be either `application/n-quads` or `application/ls+json` for packages and messages, and can be any value (although a value is still required) for files.

You'll get the CID of the files back in the `ETag` response header.

```
% curl -i -X PUT -T 8-cell-orig.gif \
-H 'Content-Type: image/gif' \
-H 'Link: <http://www.w3.org/ns/ldp#Resource>; rel="type"' \
-H 'Link: <http://www.w3.org/ns/ldp#NonRDFSource>; rel="type"' \
http://localhost:8086/8-cell-orig.gif
HTTP/1.1 100 Continue

HTTP/1.1 201 Created
Etag: bafybeiatr6vzozvaxtp5f32ghixj4bvauz6wgl4lbbh6np4yrrsvtep3y4
Date: Sat, 07 Dec 2019 07:05:16 GMT
Content-Length: 0

```

### MKCOL

### DELETE

You need to provide a resource's current `ETag` in an `If-Match` header in order to delete it. If you don't, your request will be rejected eith `416 Requested Range Not Satisfiable`.

```
% curl -i -X DELETE http://localhost:8086/8-cell-orig.gif
HTTP/1.1 416 Requested Range Not Satisfiable
Date: Sat, 07 Dec 2019 07:23:09 GMT
Content-Length: 0

```

```
% curl -i -X DELETE \
-H 'If-Match: bafybeiatr6vzozvaxtp5f32ghixj4bvauz6wgl4lbbh6np4yrrsvtep3y4' \
http://localhost:8086/8-cell-orig.gif
HTTP/1.1 200 OK
Date: Sat, 07 Dec 2019 07:25:01 GMT
Content-Length: 0

```

## Installation

### Building as an HTTP API client

### Building as a plugin

pkgs is a [go-ipfs plugin](https://github.com/ipfs/go-ipfs/blob/master/docs/plugins.md), which means the best way to use it is to build IPFS from source and follow the instructions for a preloaded plugin.

You'll have to do some `go.mod` edits to get everything to compile.

First, clone this repo into your `$GOPATH/src/github.com/underlay` folder (`$GOPATH` is usually `~/go`, but you can check with `go env GOPATH`).

```
mkdir -p $(go env GOPATH)/src/github.com/underlay/pkgs
git clone https://github.com/underlay/pkgs $(go env GOPATH)/src/github.com/underlay/pkgs
```

... and do the same for go-ipfs (make sure it's not already there)

```
mkdir -p $(go env GOPATH)/src/github.com/ipfs/go-ipfs
git clone https://github.com/ipfs/go-ipfs $(go env GOPATH)/src/github.com/ipfs/go-ipfs
```

... then point the `underlay/pkgs/go.mod` reference to `go-ipfs` to your local `go-ipfs`:

```
cd $(go env GOPATH)/src/github.com/underlay/pkgs
go mod edit -replace github.com/ipfs/go-ipfs $(go env GOPATH)/src/github.com/ipfs/go-ipfs
```

... then over in `ipfs/go-ipfs`, add `pkgs` to the `preload_list` and do the same replacement:

```
cd $(go env GOPATH)/src/github.com/ipfs/go-ipfs
echo 'pkgs github.com/underlay/pkgs/plugin *' >> plugin/loader/preload_list
go mod edit -replace github.com/underlay/pkgs $(go env GOPATH)/src/github.com/underlay/pkgs
make clean
make install
```

Then when you run `ipfs daemon --initialize` (assuming `ipfs` resolves to `$(go env GOPATH)/bin/ipfs`), we should see the pkgs plugin logs in the console:

```
2019/12/04 16:19:49 Starting pkgs plugin
badger 2019/12/04 16:19:49 INFO: All 0 tables opened in 0s
badger 2019/12/04 16:19:49 INFO: Replaying file id: 0 at offset: 0
badger 2019/12/04 16:19:49 INFO: Replay took: 43.738µs
badger 2019/12/04 16:19:49 DEBUG: Value log discard stats empty
2019/12/04 16:19:49 pkgs root: bafkreiamamtl7tmcftoejhjaho3udik5blnmzxrz62vgegldbwqz6nwuei
2019/12/04 16:19:49 Listening on http://localhost:8086
```

## Development

Regenerate the protobuf type definitions with:

```
protoc --go_out=. types/types.proto
```
