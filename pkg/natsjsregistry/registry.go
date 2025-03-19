// Package natsjsregistry implements a registry using natsjs kv store
package natsjsregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	natsjskv "github.com/go-micro/plugins/v4/store/nats-js-kv"
	"github.com/nats-io/nats.go"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/server"
	"go-micro.dev/v4/store"
	"go-micro.dev/v4/util/cmd"
)

var (
	_registryName        = "nats-js-kv"
	_registryAddressEnv  = "MICRO_REGISTRY_ADDRESS"
	_registryUsernameEnv = "MICRO_REGISTRY_AUTH_USERNAME"
	_registryPasswordEnv = "MICRO_REGISTRY_AUTH_PASSWORD"

	_serviceDelimiter = "@"
)

func init() {
	cmd.DefaultRegistries[_registryName] = NewRegistry
}

// NewRegistry returns a new natsjs registry
func NewRegistry(opts ...registry.Option) registry.Registry {
	options := registry.Options{
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}
	defaultTTL, _ := options.Context.Value(defaultTTLKey{}).(time.Duration)
	n := &storeregistry{
		opts:       options,
		typ:        _registryName,
		defaultTTL: defaultTTL,
	}
	n.store = natsjskv.NewStore(n.storeOptions(options)...)
	return n
}

type storeregistry struct {
	opts       registry.Options
	store      store.Store
	typ        string
	defaultTTL time.Duration
	lock       sync.RWMutex
}

// Init inits the registry
func (n *storeregistry) Init(opts ...registry.Option) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	for _, o := range opts {
		o(&n.opts)
	}
	n.store = natsjskv.NewStore(n.storeOptions(n.opts)...)
	return n.store.Init(n.storeOptions(n.opts)...)
}

// Options returns the configured options
func (n *storeregistry) Options() registry.Options {
	return n.opts
}

// Register adds a service to the registry
func (n *storeregistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	n.lock.RLock()
	defer n.lock.RUnlock()

	if s == nil {
		return errors.New("wont store nil service")
	}

	var options registry.RegisterOptions
	options.TTL = n.defaultTTL
	for _, o := range opts {
		o(&options)
	}

	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return n.store.Write(&store.Record{
		Key:    s.Name + _serviceDelimiter + server.DefaultId + _serviceDelimiter + s.Version,
		Value:  b,
		Expiry: options.TTL,
	})
}

// Deregister removes a service from the registry.
func (n *storeregistry) Deregister(s *registry.Service, _ ...registry.DeregisterOption) error {
	n.lock.RLock()
	defer n.lock.RUnlock()
	return n.store.Delete(s.Name + _serviceDelimiter + server.DefaultId + _serviceDelimiter + s.Version)
}

// GetService gets a specific service from the registry
func (n *storeregistry) GetService(s string, _ ...registry.GetOption) ([]*registry.Service, error) {
	// avoid listing e.g. `webfinger` when requesting `web` by adding the delimiter to the service name
	return n.listServices(store.ListPrefix(s + _serviceDelimiter))
}

// ListServices lists all registered services
func (n *storeregistry) ListServices(...registry.ListOption) ([]*registry.Service, error) {
	return n.listServices()
}

// Watch allowes following the changes in the registry if it would be implemented
func (n *storeregistry) Watch(...registry.WatchOption) (registry.Watcher, error) {
	return NewWatcher(n)
}

// String returns the name of the registry
func (n *storeregistry) String() string {
	return n.typ
}

func (n *storeregistry) listServices(opts ...store.ListOption) ([]*registry.Service, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	keys, err := n.store.List(opts...)
	if err != nil {
		return nil, err
	}

	versions := map[string]*registry.Service{}
	for _, k := range keys {
		s, err := n.getNode(k)
		if err != nil {
			// TODO: continue ?
			return nil, err
		}
		if versions[s.Version] == nil {
			versions[s.Version] = s
		} else {
			versions[s.Version].Nodes = append(versions[s.Version].Nodes, s.Nodes...)
		}
	}
	svcs := make([]*registry.Service, 0, len(versions))
	for _, s := range versions {
		svcs = append(svcs, s)
	}
	return svcs, nil
}

// getNode retrieves a node from the store. It returns a service to also keep track of the version.
func (n *storeregistry) getNode(s string) (*registry.Service, error) {
	recs, err := n.store.Read(s)
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, registry.ErrNotFound
	}
	var svc registry.Service
	if err := json.Unmarshal(recs[0].Value, &svc); err != nil {
		return nil, err
	}
	return &svc, nil
}

func (n *storeregistry) storeOptions(opts registry.Options) []store.Option {
	storeoptions := []store.Option{
		store.Database("service-registry"),
		store.Table("service-registry"),
		natsjskv.DefaultMemory(),
		natsjskv.EncodeKeys(),
	}

	if defaultTTL, ok := opts.Context.Value(defaultTTLKey{}).(time.Duration); ok {
		storeoptions = append(storeoptions, natsjskv.DefaultTTL(defaultTTL))
	}

	serviceName := "_oc" // use "_oc" as default service name if nothing else is provided
	if name, ok := opts.Context.Value(serviceNameKey{}).(string); ok {
		serviceName = name
	}

	addr := []string{"127.0.0.1:9233"}
	if len(opts.Addrs) > 0 {
		addr = opts.Addrs
	} else if a := strings.Split(os.Getenv(_registryAddressEnv), ","); len(a) > 0 && a[0] != "" {
		addr = a
	}
	storeoptions = append(storeoptions, store.Nodes(addr...))

	natsOptions := nats.GetDefaultOptions()
	natsOptions.Name = generators.GenerateConnectionName(serviceName, generators.NTypeRegistry)
	natsOptions.User, natsOptions.Password = getAuth()
	natsOptions.ReconnectedCB = func(_ *nats.Conn) {
		if err := n.Init(); err != nil {
			fmt.Println("cannot reconnect to nats")
			os.Exit(1)
		}
	}
	natsOptions.ClosedCB = func(_ *nats.Conn) {
		fmt.Println("nats connection closed")
		os.Exit(1)
	}
	storeoptions = append(storeoptions, natsjskv.NatsOptions(natsOptions))

	if so, ok := opts.Context.Value(storeOptionsKey{}).([]store.Option); ok {
		storeoptions = append(storeoptions, so...)
	}

	return storeoptions
}

func getAuth() (string, string) {
	return os.Getenv(_registryUsernameEnv), os.Getenv(_registryPasswordEnv)
}
