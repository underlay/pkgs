package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	ipfs "github.com/ipfs/go-ipfs-http-client"
	cors "github.com/rs/cors"

	pkgs "github.com/underlay/pkgs/http"
)

const defaultHost = "http://localhost:5001"

var ipfsHost = os.Getenv("IPFS_HOST")
var pkgsPath = os.Getenv("PKGS_PATH")
var pkgsRoot = os.Getenv("PKGS_ROOT")

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

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.Handle)
	handler := cors.New(cors.Options{
		AllowCredentials: false,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPatch,
			http.MethodPut,
			http.MethodHead,
			http.MethodDelete,
			"MKCOL",
			"MOVE",
			"COPY",
		},
		AllowedHeaders: []string{"Link", "If-Match", "If-None-Match", "Content-Type", "Accept"},
		ExposedHeaders: []string{"Content-Type", "Link", "ETag", "Content-Disposition", "Content-Length"},
		Debug:          true,
	}).Handler(mux)

	log.Println("http://localhost:8086")
	log.Fatal(http.ListenAndServe(":8086", handler))
}
