# pkgs

pkgs is an [LDP Server](https://www.w3.org/TR/ldp/) with a [WebDAV](https://en.wikipedia.org/wiki/WebDAV) interface built on IPFS.

"Directories" are called Packages, and they have a direct RDF representation as an LDP Direct Container and they contain members that can be RDF datasets, arbitrary files, or subpackages.

All the RDF sources (packages and datasets) are stored as canonicalized n-quads files.

## Development

Regenerate the protobuf type definitions with:

```
protoc --go_out=. types/types.proto
```
