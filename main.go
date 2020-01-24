package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	ipfs "github.com/ipfs/go-ipfs-http-client"
	cors "github.com/rs/cors"

	pkgs "github.com/underlay/pkgs/http"
	rpc "github.com/underlay/pkgs/rpc"
)

const defaultHost = "http://localhost:5001"

var ipfsHost = os.Getenv("IPFS_HOST")
var pkgsPath = os.Getenv("PKGS_PATH")
var pkgsRoot = os.Getenv("PKGS_ROOT")

func main() {
	if ipfsHost == "" {
		ipfsHost = defaultHost
	}

	go rpc.ServeRPC()

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
	}).Handler(server)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		_ = <-c
		log.Println("Closing database")
		server.Close()
		os.Exit(1)
	}()

	log.Println("http://localhost:8086")
	log.Fatal(http.ListenAndServe(":8086", handler))
}
