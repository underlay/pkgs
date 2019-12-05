package plugin

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	badger "github.com/dgraph-io/badger/v2"
	plugin "github.com/ipfs/go-ipfs/plugin"
	core "github.com/ipfs/interface-go-ipfs-core"
	path "github.com/ipfs/interface-go-ipfs-core/path"

	server "github.com/underlay/pkgs/server"
	types "github.com/underlay/pkgs/types"
)

const defaultOrigin = "dweb:/ipns"

// var port = os.Getenv("PKGS_PORT")
var pkgsPath = os.Getenv("PKGS_PATH")

var name = os.Getenv("PKGS_NAME")
var origin = os.Getenv("PKGS_ORIGIN")

// PkgsPlugin is an IPFS deamon plugin
type PkgsPlugin struct {
	srv *http.Server
	db  *badger.DB
}

// Compile-time type check
var _ plugin.PluginDaemon = (*PkgsPlugin)(nil)

// Name returns the plugin's name, satisfying the plugin.Plugin interface.
func (sp *PkgsPlugin) Name() string {
	return "pkgs"
}

// Version returns the plugin's version, satisfying the plugin.Plugin interface.
func (sp *PkgsPlugin) Version() string {
	return "0.1.0"
}

// Init initializes plugin, satisfying the plugin.Plugin interface.
func (sp *PkgsPlugin) Init(env *plugin.Environment) error {
	if origin == "" {
		origin = defaultOrigin
	}

	if pkgsPath == "" {
		pkgsPath = "/tmp/pkgs"
	}

	return nil
}

func (sp *PkgsPlugin) listen() {
	log.Printf("Listening on http://localhost%s\n", sp.srv.Addr)
	err := sp.srv.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe(): %s", err)
	}
}

// Start gets passed a CoreAPI instance, satisfying the plugin.PluginDaemon interface.
func (sp *PkgsPlugin) Start(api core.CoreAPI) error {
	log.Println("Starting pkgs plugin")
	ctx := context.Background()
	err := api.Pin().Add(ctx, path.IpfsPath(types.EmptyDirectoryCID))
	if err != nil {
		return err
	}

	if name == "" {
		key, err := api.Key().Self(ctx)
		if err != nil {
			return err
		}
		name = key.ID().String()
	}

	resource := fmt.Sprintf("%s/%s", origin, name)

	sp.db, err = server.Initialize(ctx, pkgsPath, resource, api)
	if err != nil {
		return err
	}

	sp.srv = &http.Server{Addr: ":8086"}
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		server.Handler(res, req, sp.db, api)
	})
	go sp.listen()
	return nil
}

// Close gets called when the IPFS deamon shuts down, satisfying the plugin.PluginDaemon interface.
func (sp *PkgsPlugin) Close() error {
	if sp.srv != nil {
		return sp.srv.Shutdown(context.TODO())
	}
	return nil
}

// Plugins is an exported list of plugins that will be loaded by go-ipfs.
var Plugins = []plugin.Plugin{&PkgsPlugin{}}
