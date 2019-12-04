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
	multibase "github.com/multiformats/go-multibase"

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

	var err error

	opts := badger.DefaultOptions(pkgsPath)

	sp.db, err = badger.Open(opts)
	if err != nil {
		return err
	}

	fs, pin := api.Unixfs(), api.Pin()

	ctx := context.Background()
	err = pin.Add(ctx, path.IpfsPath(types.EmptyDirectoryCID))
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

	index := "/"

	var root string
	err = sp.db.Update(func(txn *badger.Txn) error {
		r := &types.Resource{}
		err := r.Get(index, txn)
		if err == badger.ErrKeyNotFound {
			c, p, err := types.NewPackage(ctx, index, resource, fs)
			if err != nil {
				return err
			}

			root, err = c.StringOfBase(multibase.Base32)
			if err != nil {
				return err
			}

			r.Resource = &types.Resource_Package{Package: p}
			return r.Set(index, txn)
		} else if err != nil {
			return err
		}

		pkg := r.GetPackage()
		if pkg == nil {
			return fmt.Errorf("Invalid index: %v", r)
		}

		_, root, err = types.GetCid(pkg.Id)
		return err
	})

	if err != nil {
		return err
	}

	log.Println("pkgs root:", root)

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
