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

## Installation

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
