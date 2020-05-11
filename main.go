package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	badger "github.com/dgraph-io/badger/v2"
	ipfs "github.com/ipfs/go-ipfs-http-client"
	cors "github.com/rs/cors"

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

	badgerPath := pkgsPath + "/badger"
	err = os.MkdirAll(badgerPath, 0755)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Opening badger database at", badgerPath)
	opts := badger.DefaultOptions(badgerPath)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Opening indices")
	for _, index := range rpc.INDICES {
		path := pkgsPath + "/indices/" + index.Name()
		err = os.MkdirAll(path, 0755)
		if err != nil {
			log.Fatalln(err)
		}

		index.Init(api, db, path)
	}

	server, err := NewServer(ctx, pkgsRoot, db, api)
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
		},
		AllowedHeaders: []string{"Link", "If-Match", "If-None-Match", "Content-Type", "Accept"},
		ExposedHeaders: []string{"Content-Type", "Link", "ETag", "Content-Disposition", "Content-Length"},
		Debug:          false,
	}).Handler(server)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		_ = <-c
		log.Println("Closing database")
		server.Close()
		log.Println("Closing indices")
		for _, index := range rpc.INDICES {
			index.Close()
		}
		os.Exit(1)
	}()

	log.Println("http://localhost:8086")
	log.Fatal(http.ListenAndServe(":8086", handler))
}
