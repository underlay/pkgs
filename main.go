package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	ipfs "github.com/ipfs/go-ipfs-http-client"

	pkgs "github.com/underlay/pkgs/server"
)

const defaultHost = "http://localhost:5001"

var ipfsHost = os.Getenv("IPFS_HOST")
var pkgsPath = os.Getenv("PKGS_PATH")
var pkgsRoot = os.Getenv("PKGS_ROOT")

var pathRegex = regexp.MustCompile("^(/[a-zA-Z0-9-\\.]+)+$")

func main() {
	if ipfsHost == "" {
		ipfsHost = defaultHost
	}

	api, err := ipfs.NewURLApiWithClient(ipfsHost, http.DefaultClient)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	if pkgsRoot == "" {
		key, err := api.Key().Self(ctx)
		if err != nil {
			log.Fatal(err)
		}
		pkgsRoot = fmt.Sprintf("dweb:/ipns/%s", key.ID().String())
	}

	if pkgsPath == "" {
		pkgsPath = "/tmp/pkgs"
	}

	server, err := pkgs.Initialize(ctx, pkgsPath, pkgsRoot, api)

	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", server.Handle)
	log.Printf("http://localhost:8086\n")
	log.Fatal(http.ListenAndServe(":8086", nil))
}
