package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	ipfs "github.com/ipfs/go-ipfs-http-client"
	loader "github.com/underlay/go-dweb-loader/loader"

	server "github.com/underlay/pkgs/server"
	types "github.com/underlay/pkgs/types"
)

const defaultHost = "http://localhost:5001"
const defaultOrigin = "dweb:/ipns"

var ipfsHost = os.Getenv("IPFS_HOST")
var pkgsPath = os.Getenv("PKGS_PATH")
var pkgsName = os.Getenv("PKGS_NAME")
var pkgsOrigin = os.Getenv("PKGS_ORIGIN")

var pathRegex = regexp.MustCompile("^(/[a-zA-Z0-9-\\.]+)+$")

func main() {
	if pkgsOrigin == "" {
		pkgsOrigin = defaultOrigin
	}

	if ipfsHost == "" {
		ipfsHost = defaultHost
	}

	api, err := ipfs.NewURLApiWithClient(ipfsHost, http.DefaultClient)

	if err != nil {
		log.Fatal(err)
	}

	types.Opts.DocumentLoader = loader.NewDwebDocumentLoader(api)
	types.Opts.Format = "application/n-quads"
	types.Opts.CompactArrays = true
	types.Opts.UseNativeTypes = true

	ctx := context.Background()
	if pkgsName == "" {
		key, err := api.Key().Self(ctx)
		if err != nil {
			log.Fatal(err)
		}
		pkgsName = key.ID().String()
	}

	resource := fmt.Sprintf("%s/%s", pkgsOrigin, pkgsName)

	if pkgsPath == "" {
		pkgsPath = "/tmp/pkgs"
	}

	db, err := server.Initialize(ctx, pkgsPath, resource, api)

	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		server.Handler(res, req, db, api)
	})

	log.Printf("http://localhost:8086\n")

	log.Fatal(http.ListenAndServe(":8086", nil))
}
