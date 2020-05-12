# pkgs

pkgs is an [LDP Server](https://www.w3.org/TR/ldp/) with a [WebDAV](https://en.wikipedia.org/wiki/WebDAV) interface built on IPFS.

"Directories" are called Packages, and they have a direct RDF representation as an LDP Direct Container and they contain members that can be RDF datasets, arbitrary files, or subpackages.

All the RDF sources (packages and datasets) are stored as canonicalized n-quads files.

## Usage

Make sure you have an IPFS daemon running - preferably go-ipfs v0.5.0+.

Compile the main pkgs daemon with

```
go build .
```

This should give you a `pkgs` binary. Running it should give you something like this:

```
2020/05/12 11:22:49 Opening badger database at /tmp/pkgs/badger
badger 2020/05/12 11:22:49 INFO: All 0 tables opened in 0s
2020/05/12 11:22:49 Opening indices
2020/05/12 11:22:49 Log index: init /tmp/pkgs/indices/log
2020/05/12 11:22:49 schema.jsonld:	 dweb:/ipfs/bafkreick3772z66zzvan3t5zweoajutehbggwk4nkkkc2oplkoawxvxgyi
2020/05/12 11:22:49 shex.jsonld:	 dweb:/ipfs/bafkreib6yqj52ndfimyqueoxxxgsxoiolpr6pfjz5ossxi6mc25bjok6fa
2020/05/12 11:22:49 context.jsonld:	 dweb:/ipfs/bafkreibwyzeetse6wpzntw2rj5jkblxccysntihmezupsjwgx532ttrwxm
2020/05/12 11:22:49 package.jsonld:	 dweb:/ipfs/bafkreif7y43zk6p7hlvmarjveblgiqyy2266wxounchapnakmuxdjqdpam
2020/05/12 11:22:49 No root package found; creating new initial package
2020/05/12 11:22:49 Log index: set /context.jsonld dweb:/ipfs/bafkreibwyzeetse6wpzntw2rj5jkblxccysntihmezupsjwgx532ttrwxm
2020/05/12 11:22:49 Log index: set /package.jsonld dweb:/ipfs/bafkreif7y43zk6p7hlvmarjveblgiqyy2266wxounchapnakmuxdjqdpam
2020/05/12 11:22:49 Log index: set /schema.jsonld dweb:/ipfs/bafkreick3772z66zzvan3t5zweoajutehbggwk4nkkkc2oplkoawxvxgyi
2020/05/12 11:22:49 Log index: set /shex.jsonld dweb:/ipfs/bafkreib6yqj52ndfimyqueoxxxgsxoiolpr6pfjz5ossxi6mc25bjok6fa
2020/05/12 11:22:49 Log index: set / ul:bafkreiclxv4ddxu7t5qb6v6eew5bttk22khnoyfnyyurwlftlrnyiaygvy#c14n0
2020/05/12 11:22:49 http://localhost:8086

```

This opens various databases in the default location `/tmp/pkgs/`; you can change this by setting a `PKGS_PATH` environment variable.

You should be able to open `http://localhost:8086` in a web browser and see the root package with the four default initial files.

You can also build a cli tool:

```
go build -o ul ./cli
```

This should give you a `ul` binary. You can use it as described in the [CLI docs](CLI.md).
