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

```

### HEAD

### POST

### PUT

### MKCOL

### DELETE

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
