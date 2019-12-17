# pkgs

pkgs is an [LDP Server](https://www.w3.org/TR/ldp/) with a [WebDAV](https://en.wikipedia.org/wiki/WebDAV) interface built on IPFS.

"Directories" are called Packages, and they have a direct RDF representation as an LDP Direct Container and they contain members that can be RDF datasets, arbitrary files, or subpackages.

All the RDF sources (packages and datasets) are stored as canonicalized n-quads files.

## Usage

Some environment variables:

- `PKGS_PATH` (default `/tmp/pkgs`) is the directory that pkgs will open a [Badger](https://github.com/dgraph-io/badger) database in for caching the directory tree.
- `PKGS_ROOT` (default `dweb:/ipns/Qm...`, where `Qm...` is the PeerID of the IPFS node) root for local URIs - root package `/` will be identified with it, and all other resources will be identified as paths relative it. Don't include a trailing slash.

Reading about WebDAV and LDP are good background for what pkgs is and how to use it.

Essentially, pkgs manages a filesystem that you interact with over HTTP, using an extended set of HTTP verbs like `MKCOL` ("make collection" a la Unix `mkdir`). Directories in this filesystem are called _Packages_, and every package is identified by a URI - you configure pkgs with a "root" URI like `http://example.com`, which lets us address the rest of the directory tree with URIs like `http://example.com/foo`, `http://example.com/bar/baz`, etc.

This root URI isn't used for anything other than identifiers in RDF, so it doesn't need to resolve to anything.

### GET

`GET` requests to a resource will return `Content-Type: application/n-quads` by default.

```
% curl -i http://localhost:8086
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

The remote context `ipfs://bafkreifcqgsljpst2fabpvmlzcf5fqdthzvhf4imvqvnymk5iifi6mdtru` is from [types/utils.go](types/utils.go) and it's added as a file `/context.jsonld` to the root package when you start pkgs for the first time.

### HEAD

```
% curl -I http://localhost:8086/8-cell-orig.gif
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

`PUT` requires at least one `Link` header and one `Content-Type` header. The `Link` must be one of:

- `<http://www.w3.org/ns/ldp#DirectContainer>; rel="type"`
- `<http://www.w3.org/ns/ldp#RDFSource>; rel="type"`
- `<http://www.w3.org/ns/ldp#NonRDFSource>; rel="type"`

for packages, messages, and files, respectively. For now, only messages and files can be `PUT` - new packages must be created with `MKCOL`.

`Content-Type` must be either `application/n-quads` or `application/ld+json` for packages and messages, and can be any value (although a value is still required) for files.

In addition, if `PUT` is being used to modify an existing resource (it can also be used to create new resources), the request must have an `If-Match` header whose value is the resource's current `ETag`, just like `DELETE`. If the header is not provided, or if it does not match, the request will fail with `412 Precondition Failed`.

You'll get the CID of the file or message back in the `ETag` response header.

```
% curl -i -X PUT -T 8-cell-orig.gif \
-H 'Link: <http://www.w3.org/ns/ldp#NonRDFSource>; rel="type"' \
-H 'Content-Type: image/gif' \
http://localhost:8086/8-cell-orig.gif
HTTP/1.1 100 Continue

HTTP/1.1 201 Created
Etag: bafybeiatr6vzozvaxtp5f32ghixj4bvauz6wgl4lbbh6np4yrrsvtep3y4
Date: Sat, 07 Dec 2019 07:05:16 GMT
Content-Length: 0

```

### MKCOL

Create (empty) packages with `MKCOL`:

```
% curl -i -X MKCOL http://localhost:8086/bar
HTTP/1.1 201 Created
Etag: bafkreif42bur6q7n476a54f55nfnzngojyf3emnm77nzu4aui4swjyrzua
Date: Tue, 10 Dec 2019 01:22:28 GMT
Content-Length: 0

% curl -H 'Accept: application/ld+json' http://localhost:8086/bar | jq
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100   463  100   463    0     0  77166      0 --:--:-- --:--:-- --:--:-- 77166
{
  "@context": "ipfs://bafkreifcqgsljpst2fabpvmlzcf5fqdthzvhf4imvqvnymk5iifi6mdtru",
  "@type": "http://underlay.mit.edu/ns#Package",
  "dcterms:created": "2019-12-09T20:22:27-05:00",
  "dcterms:modified": "2019-12-09T20:22:27-05:00",
  "ldp:hasMemberRelation": "prov:hadMember",
  "ldp:membershipResource": "dweb:/ipns/QmRybuaATHF1mnVy3VhhcbRhUedc3DkrpgMQBVEXx7oT9r/bar",
  "prov:value": {
    "@id": "dweb:/ipfs/bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354",
    "dcterms:extent": 4
  }
}

```

### DELETE

You need to provide a resource's current `ETag` in an `If-Match` header in order to delete it. If you don't, your request will be rejected with `412 Precondition Failed`.

```
% curl -i -X DELETE http://localhost:8086/8-cell-orig.gif
HTTP/1.1 412 Precondition Failed
Date: Sat, 07 Dec 2019 07:23:09 GMT
Content-Length: 0

% curl -i -X DELETE \
-H 'If-Match: bafybeiatr6vzozvaxtp5f32ghixj4bvauz6wgl4lbbh6np4yrrsvtep3y4' \
http://localhost:8086/8-cell-orig.gif
HTTP/1.1 200 OK
Date: Sat, 07 Dec 2019 07:25:01 GMT
Content-Length: 0

```

## Installation

### Building as an HTTP API client

If you already have a local IPFS daemon, you can build pkgs as a standalone application:

```
% go build .
% ./pkgs
```

This will interface with the IPFS daemon over its HTTP API, which is slower than the CoreAPI interface that plugins get, but is more convenient for debugging.

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
