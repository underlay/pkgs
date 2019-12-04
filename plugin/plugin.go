package plugin

import (
	"context"
	"log"
	"net/http"

	badger "github.com/dgraph-io/badger/v2"
	plugin "github.com/ipfs/go-ipfs/plugin"
	core "github.com/ipfs/interface-go-ipfs-core"

	server "github.com/underlay/pkgs/server"
)

// options "github.com/ipfs/interface-go-ipfs-core/options"
// path "github.com/ipfs/interface-go-ipfs-core/path"

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
func (sp *PkgsPlugin) Init() error {
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
	sp.srv = &http.Server{Addr: ":8080"}
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
