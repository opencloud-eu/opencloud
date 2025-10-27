package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/mohae/deepcopy"
	"github.com/olekukonko/tablewriter"
	occfg "github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	ogrpc "github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	activitylog "github.com/opencloud-eu/opencloud/services/activitylog/pkg/command"
	antivirus "github.com/opencloud-eu/opencloud/services/antivirus/pkg/command"
	appProvider "github.com/opencloud-eu/opencloud/services/app-provider/pkg/command"
	appRegistry "github.com/opencloud-eu/opencloud/services/app-registry/pkg/command"
	audit "github.com/opencloud-eu/opencloud/services/audit/pkg/command"
	authapp "github.com/opencloud-eu/opencloud/services/auth-app/pkg/command"
	authbasic "github.com/opencloud-eu/opencloud/services/auth-basic/pkg/command"
	authmachine "github.com/opencloud-eu/opencloud/services/auth-machine/pkg/command"
	authservice "github.com/opencloud-eu/opencloud/services/auth-service/pkg/command"
	clientlog "github.com/opencloud-eu/opencloud/services/clientlog/pkg/command"
	collaboration "github.com/opencloud-eu/opencloud/services/collaboration/pkg/command"
	eventhistory "github.com/opencloud-eu/opencloud/services/eventhistory/pkg/command"
	frontend "github.com/opencloud-eu/opencloud/services/frontend/pkg/command"
	gateway "github.com/opencloud-eu/opencloud/services/gateway/pkg/command"
	graph "github.com/opencloud-eu/opencloud/services/graph/pkg/command"
	groups "github.com/opencloud-eu/opencloud/services/groups/pkg/command"
	idm "github.com/opencloud-eu/opencloud/services/idm/pkg/command"
	idp "github.com/opencloud-eu/opencloud/services/idp/pkg/command"
	invitations "github.com/opencloud-eu/opencloud/services/invitations/pkg/command"
	nats "github.com/opencloud-eu/opencloud/services/nats/pkg/command"
	notifications "github.com/opencloud-eu/opencloud/services/notifications/pkg/command"
	ocdav "github.com/opencloud-eu/opencloud/services/ocdav/pkg/command"
	ocm "github.com/opencloud-eu/opencloud/services/ocm/pkg/command"
	ocs "github.com/opencloud-eu/opencloud/services/ocs/pkg/command"
	policies "github.com/opencloud-eu/opencloud/services/policies/pkg/command"
	postprocessing "github.com/opencloud-eu/opencloud/services/postprocessing/pkg/command"
	proxy "github.com/opencloud-eu/opencloud/services/proxy/pkg/command"
	search "github.com/opencloud-eu/opencloud/services/search/pkg/command"
	settings "github.com/opencloud-eu/opencloud/services/settings/pkg/command"
	sharing "github.com/opencloud-eu/opencloud/services/sharing/pkg/command"
	sse "github.com/opencloud-eu/opencloud/services/sse/pkg/command"
	storagepublic "github.com/opencloud-eu/opencloud/services/storage-publiclink/pkg/command"
	storageshares "github.com/opencloud-eu/opencloud/services/storage-shares/pkg/command"
	storageSystem "github.com/opencloud-eu/opencloud/services/storage-system/pkg/command"
	storageusers "github.com/opencloud-eu/opencloud/services/storage-users/pkg/command"
	thumbnails "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/command"
	userlog "github.com/opencloud-eu/opencloud/services/userlog/pkg/command"
	users "github.com/opencloud-eu/opencloud/services/users/pkg/command"
	web "github.com/opencloud-eu/opencloud/services/web/pkg/command"
	webdav "github.com/opencloud-eu/opencloud/services/webdav/pkg/command"
	webfinger "github.com/opencloud-eu/opencloud/services/webfinger/pkg/command"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/opencloud-eu/reva/v2/pkg/logger"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/thejerf/suture/v4"
)

var (
	// runset keeps track of which services to start supervised.
	runset map[string]struct{}

	// wait funcs run after the service group has been started.
	_waitFuncs = []func(*occfg.Config) error{pingNats, pingGateway, nil, wait(time.Second), nil}

	// Use the runner.DefaultInterruptDuration as defaults for the individual service shutdown timeouts.
	_defaultShutdownTimeoutDuration = runner.DefaultInterruptDuration
	// Use the runner.DefaultGroupInterruptDuration as defaults for the server interruption timeout.
	_defaultInterruptTimeoutDuration = runner.DefaultGroupInterruptDuration
)

type serviceFuncMap map[string]func(*occfg.Config) suture.Service

// Service represents a RPC service.
type Service struct {
	Supervisor *suture.Supervisor
	Services   []serviceFuncMap
	Additional serviceFuncMap
	Log        log.Logger

	serviceToken map[string][]suture.ServiceToken
	cfg          *occfg.Config
}

// NewService returns a configured service with a controller and a default logger.
// When used as a library, flags are not parsed, and in order to avoid introducing a global state with init functions
// calls are done explicitly to loadFromEnv().
// Since this is the public constructor, options need to be added, at the moment only logging options
// are supported in order to match the running OpenCloud services structured log.
func NewService(ctx context.Context, options ...Option) (*Service, error) {
	opts := NewOptions()

	for _, f := range options {
		f(opts)
	}

	l := log.NewLogger(
		log.Color(opts.Config.Log.Color),
		log.Pretty(opts.Config.Log.Pretty),
		log.Level(opts.Config.Log.Level),
	)

	s := &Service{
		Services:   make([]serviceFuncMap, len(_waitFuncs)),
		Additional: make(serviceFuncMap),
		Log:        l,

		serviceToken: make(map[string][]suture.ServiceToken),
		cfg:          opts.Config,
	}

	// populate services
	reg := func(priority int, name string, exec func(context.Context, *occfg.Config) error) {
		if s.Services[priority] == nil {
			s.Services[priority] = make(serviceFuncMap)
		}
		s.Services[priority][name] = NewSutureServiceBuilder(exec)
	}

	// nats is in priority group 0. It needs to start before all other services
	reg(0, opts.Config.Nats.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Nats.Context = ctx
		cfg.Nats.Commons = cfg.Commons
		return nats.Execute(cfg.Nats)
	})

	// gateway is in priority group 1. It needs to start before the reva services
	reg(1, opts.Config.Gateway.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Gateway.Context = ctx
		cfg.Gateway.Commons = cfg.Commons
		return gateway.Execute(cfg.Gateway)
	})

	// priority group 2 is empty for now

	// most services are in priority group 3
	reg(3, opts.Config.Activitylog.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Activitylog.Context = ctx
		cfg.Activitylog.Commons = cfg.Commons
		return activitylog.Execute(cfg.Activitylog)
	})
	reg(3, opts.Config.AppProvider.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.AppProvider.Context = ctx
		cfg.AppProvider.Commons = cfg.Commons
		return appProvider.Execute(cfg.AppProvider)
	})
	reg(3, opts.Config.AppRegistry.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.AppRegistry.Context = ctx
		cfg.AppRegistry.Commons = cfg.Commons
		return appRegistry.Execute(cfg.AppRegistry)
	})
	reg(3, opts.Config.AuthApp.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.AuthApp.Context = ctx
		cfg.AuthApp.Commons = cfg.Commons
		return authapp.Execute(cfg.AuthApp)
	})
	reg(3, opts.Config.AuthBasic.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.AuthBasic.Context = ctx
		cfg.AuthBasic.Commons = cfg.Commons
		return authbasic.Execute(cfg.AuthBasic)
	})
	reg(3, opts.Config.AuthMachine.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.AuthMachine.Context = ctx
		cfg.AuthMachine.Commons = cfg.Commons
		return authmachine.Execute(cfg.AuthMachine)
	})
	reg(3, opts.Config.AuthService.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.AuthService.Context = ctx
		cfg.AuthService.Commons = cfg.Commons
		return authservice.Execute(cfg.AuthService)
	})
	reg(3, opts.Config.Clientlog.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Clientlog.Context = ctx
		cfg.Clientlog.Commons = cfg.Commons
		return clientlog.Execute(cfg.Clientlog)
	})
	reg(3, opts.Config.EventHistory.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.EventHistory.Context = ctx
		cfg.EventHistory.Commons = cfg.Commons
		return eventhistory.Execute(cfg.EventHistory)
	})
	reg(3, opts.Config.Graph.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Graph.Context = ctx
		cfg.Graph.Commons = cfg.Commons
		return graph.Execute(cfg.Graph)
	})
	reg(3, opts.Config.Groups.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Groups.Context = ctx
		cfg.Groups.Commons = cfg.Commons
		return groups.Execute(cfg.Groups)
	})
	reg(3, opts.Config.IDM.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.IDM.Context = ctx
		cfg.IDM.Commons = cfg.Commons
		return idm.Execute(cfg.IDM)
	})
	reg(3, opts.Config.OCDav.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.OCDav.Context = ctx
		cfg.OCDav.Commons = cfg.Commons
		return ocdav.Execute(cfg.OCDav)
	})
	reg(3, opts.Config.OCS.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.OCS.Context = ctx
		cfg.OCS.Commons = cfg.Commons
		return ocs.Execute(cfg.OCS)
	})
	reg(3, opts.Config.Postprocessing.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Postprocessing.Context = ctx
		cfg.Postprocessing.Commons = cfg.Commons
		return postprocessing.Execute(cfg.Postprocessing)
	})
	reg(3, opts.Config.Search.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Search.Context = ctx
		cfg.Search.Commons = cfg.Commons
		return search.Execute(cfg.Search)
	})
	reg(3, opts.Config.Settings.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Settings.Context = ctx
		cfg.Settings.Commons = cfg.Commons
		return settings.Execute(cfg.Settings)
	})
	reg(3, opts.Config.StoragePublicLink.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.StoragePublicLink.Context = ctx
		cfg.StoragePublicLink.Commons = cfg.Commons
		return storagepublic.Execute(cfg.StoragePublicLink)
	})
	reg(3, opts.Config.StorageShares.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.StorageShares.Context = ctx
		cfg.StorageShares.Commons = cfg.Commons
		return storageshares.Execute(cfg.StorageShares)
	})
	reg(3, opts.Config.StorageSystem.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.StorageSystem.Context = ctx
		cfg.StorageSystem.Commons = cfg.Commons
		return storageSystem.Execute(cfg.StorageSystem)
	})
	reg(3, opts.Config.StorageUsers.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.StorageUsers.Context = ctx
		cfg.StorageUsers.Commons = cfg.Commons
		return storageusers.Execute(cfg.StorageUsers)
	})
	reg(3, opts.Config.Thumbnails.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Thumbnails.Context = ctx
		cfg.Thumbnails.Commons = cfg.Commons
		return thumbnails.Execute(cfg.Thumbnails)
	})
	reg(3, opts.Config.Userlog.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Userlog.Context = ctx
		cfg.Userlog.Commons = cfg.Commons
		return userlog.Execute(cfg.Userlog)
	})
	reg(3, opts.Config.Users.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Users.Context = ctx
		cfg.Users.Commons = cfg.Commons
		return users.Execute(cfg.Users)
	})
	reg(3, opts.Config.Web.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Web.Context = ctx
		cfg.Web.Commons = cfg.Commons
		return web.Execute(cfg.Web)
	})
	reg(3, opts.Config.WebDAV.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.WebDAV.Context = ctx
		cfg.WebDAV.Commons = cfg.Commons
		return webdav.Execute(cfg.WebDAV)
	})
	reg(3, opts.Config.Webfinger.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Webfinger.Context = ctx
		cfg.Webfinger.Commons = cfg.Commons
		return webfinger.Execute(cfg.Webfinger)
	})
	reg(3, opts.Config.IDP.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.IDP.Context = ctx
		cfg.IDP.Commons = cfg.Commons
		return idp.Execute(cfg.IDP)
	})
	reg(3, opts.Config.Proxy.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Proxy.Context = ctx
		cfg.Proxy.Commons = cfg.Commons
		return proxy.Execute(cfg.Proxy)
	})
	reg(3, opts.Config.Sharing.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Sharing.Context = ctx
		cfg.Sharing.Commons = cfg.Commons
		return sharing.Execute(cfg.Sharing)
	})
	reg(3, opts.Config.SSE.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.SSE.Context = ctx
		cfg.SSE.Commons = cfg.Commons
		return sse.Execute(cfg.SSE)
	})
	reg(3, opts.Config.OCM.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.OCM.Context = ctx
		cfg.OCM.Commons = cfg.Commons
		return ocm.Execute(cfg.OCM)
	})

	// out of some unknown reason ci gets angry when frontend service starts in priority group 3
	// this is not reproducible locally, it can start when nats and gateway are already running
	// FIXME: find out why
	reg(4, opts.Config.Frontend.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Frontend.Context = ctx
		cfg.Frontend.Commons = cfg.Commons
		return frontend.Execute(cfg.Frontend)
	})

	// populate optional services
	areg := func(name string, exec func(context.Context, *occfg.Config) error) {
		s.Additional[name] = NewSutureServiceBuilder(exec)
	}
	areg(opts.Config.Antivirus.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Antivirus.Context = ctx
		// cfg.Antivirus.Commons = cfg.Commons // antivirus holds no Commons atm
		return antivirus.Execute(cfg.Antivirus)
	})
	areg(opts.Config.Audit.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Audit.Context = ctx
		cfg.Audit.Commons = cfg.Commons
		return audit.Execute(cfg.Audit)
	})
	areg(opts.Config.Collaboration.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Collaboration.Context = ctx
		cfg.Collaboration.Commons = cfg.Commons
		return collaboration.Execute(cfg.Collaboration)
	})
	areg(opts.Config.Policies.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Policies.Context = ctx
		cfg.Policies.Commons = cfg.Commons
		return policies.Execute(cfg.Policies)
	})
	areg(opts.Config.Invitations.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Invitations.Context = ctx
		cfg.Invitations.Commons = cfg.Commons
		return invitations.Execute(cfg.Invitations)
	})
	areg(opts.Config.Notifications.Service.Name, func(ctx context.Context, cfg *occfg.Config) error {
		cfg.Notifications.Context = ctx
		cfg.Notifications.Commons = cfg.Commons
		return notifications.Execute(cfg.Notifications)
	})

	return s, nil
}

// Start a rpc service. By default, the package scope Start will run all default services to provide with a working
// OpenCloud instance.
func Start(ctx context.Context, o ...Option) error {
	// Start the runtime. Most likely this was called ONLY by the `opencloud server` subcommand, but since we cannot protect
	// from the caller, the previous statement holds truth.

	// prepare a new rpc Service struct.
	s, err := NewService(ctx, o...)
	if err != nil {
		return err
	}

	// create a context that will be cancelled when too many backoff cycles on one of the services happens
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// tolerance controls backoff cycles from the supervisor.
	tolerance := 5
	totalBackoff := 0

	// Start creates its own supervisor. Running services under `opencloud server` will create its own supervision tree.
	s.Supervisor = suture.New("opencloud", suture.Spec{
		EventHook: func(e suture.Event) {
			if e.Type() == suture.EventTypeBackoff {
				totalBackoff++
				if totalBackoff == tolerance {
					cancel()
				}
			}
			s.Log.Info().Str("event", e.String()).Msg(fmt.Sprintf("supervisor: %v", e.Map()["supervisor_name"]))
		},
		FailureThreshold: 5,
		FailureBackoff:   3 * time.Second,
	})

	if s.cfg.Commons == nil {
		s.cfg.Commons = &shared.Commons{
			Log: &shared.Log{},
		}
	}

	if err = rpc.Register(s); err != nil {
		if s != nil {
			s.Log.Fatal().Err(err).Msg("could not register rpc service")
		}
	}
	rpc.HandleHTTP()

	l, err := net.Listen("tcp", net.JoinHostPort(s.cfg.Runtime.Host, s.cfg.Runtime.Port))
	if err != nil {
		s.Log.Fatal().Err(err).Msg("could not start listener")
	}
	srv := new(http.Server)

	// prepare the set of services to run
	s.generateRunSet(s.cfg)

	// There are reasons not to do this, but we have race conditions ourselves. Until we resolve them, mind the following disclaimer:
	// Calling ServeBackground will CORRECTLY start the supervisor running in a new goroutine. It is risky to directly run
	// go supervisor.Serve()
	// because that will briefly create a race condition as it starts up, if you try to .Add() services immediately afterward.
	// https://pkg.go.dev/github.com/thejerf/suture/v4@v4.0.0#Supervisor
	go s.Supervisor.ServeBackground(ctx)

	for i, service := range s.Services {
		scheduleServiceTokens(s, service)
		if _waitFuncs[i] != nil {
			if err := _waitFuncs[i](s.cfg); err != nil {
				s.Log.Fatal().Err(err).Msg("wait func failed")
			}
		}
	}

	// schedule services that are optional
	scheduleServiceTokens(s, s.Additional)

	go func() {
		if err = srv.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.Log.Fatal().Err(err).Msg("could not start rpc server")
		}
	}()

	// trapShutdownCtx will block on the context-done channel for interruptions.
	return trapShutdownCtx(s, srv, ctx)
}

// scheduleServiceTokens adds service tokens to the service supervisor.
func scheduleServiceTokens(s *Service, funcSet serviceFuncMap) {
	for name := range runset {
		if _, ok := funcSet[name]; !ok {
			continue
		}

		swap := deepcopy.Copy(s.cfg)
		s.serviceToken[name] = append(s.serviceToken[name], s.Supervisor.Add(funcSet[name](swap.(*occfg.Config))))
	}
}

// generateRunSet interprets the cfg.Runtime.Services config option to cherry-pick which services to start using
// the runtime.
func (s *Service) generateRunSet(cfg *occfg.Config) {
	runset = make(map[string]struct{})
	if cfg.Runtime.Services != nil {
		for _, name := range cfg.Runtime.Services {
			runset[name] = struct{}{}
		}
		return
	}

	for _, service := range s.Services {
		for name := range service {
			runset[name] = struct{}{}
		}
	}

	// add additional services if explicitly added by config
	for _, name := range cfg.Runtime.Additional {
		runset[name] = struct{}{}
	}

	// remove services if explicitly excluded by config
	for _, name := range cfg.Runtime.Disabled {
		delete(runset, name)
	}
}

// List running processes for the Service Controller.
func (s *Service) List(_ struct{}, reply *string) error {
	tableString := &strings.Builder{}
	table := tablewriter.NewTable(tableString)
	table.Header([]string{"Service"})

	names := []string{}
	for t := range s.serviceToken {
		if len(s.serviceToken[t]) > 0 {
			names = append(names, t)
		}
	}

	sort.Strings(names)

	for n := range names {
		table.Append([]string{names[n]})
	}

	table.Render()
	*reply = tableString.String()
	return nil
}

func trapShutdownCtx(s *Service, srv *http.Server, ctx context.Context) error {
	<-ctx.Done()
	s.Log.Info().Msg("starting graceful shutdown")
	start := time.Now()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), _defaultShutdownTimeoutDuration)
		defer cancel()
		s.Log.Debug().Msg("starting runtime listener shutdown")
		if err := srv.Shutdown(ctx); err != nil {
			s.Log.Error().Err(err).Msg("could not shutdown runtime listener")
			return
		}
		s.Log.Debug().Msg("runtime listener shutdown done")
	}()

	// shutdown services in the order defined in the config
	// any services not listed will be shutdown in parallel afterwards
	for _, sName := range s.cfg.Runtime.ShutdownOrder {
		if _, ok := s.serviceToken[sName]; !ok {
			s.Log.Warn().Str("service", sName).Msg("unknown service for ordered shutdown, skipping")
			continue
		}
		for i := range s.serviceToken[sName] {
			if err := s.Supervisor.RemoveAndWait(s.serviceToken[sName][i], _defaultShutdownTimeoutDuration); err != nil && !errors.Is(err, suture.ErrSupervisorNotRunning) {
				s.Log.Error().Err(err).Str("service", sName).Msg("could not shutdown service in order, skipping to next")
				// continue shutting down other services
				continue
			}
			s.Log.Debug().Str("service", sName).Msg("graceful ordered shutdown for service done")
		}
		delete(s.serviceToken, sName)
	}

	for sName := range s.serviceToken {
		for i := range s.serviceToken[sName] {
			wg.Add(1)
			go func() {
				s.Log.Debug().Str("service", sName).Msg("starting graceful shutdown for service")
				defer wg.Done()
				if err := s.Supervisor.RemoveAndWait(s.serviceToken[sName][i], _defaultShutdownTimeoutDuration); err != nil && !errors.Is(err, suture.ErrSupervisorNotRunning) {
					s.Log.Error().Err(err).Str("service", sName).Msg("could not shutdown service")
					return
				}
				s.Log.Debug().Str("service", sName).Msg("graceful shutdown for service done")
			}()
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-time.After(_defaultInterruptTimeoutDuration):
		s.Log.Error().Dur("timeoutDuration", _defaultInterruptTimeoutDuration).Msg("graceful shutdown timeout reached, terminating")
		return errors.New("graceful shutdown timeout reached, terminating")
	case <-done:
		duration := time.Since(start)
		s.Log.Info().Dur("duration", duration).Msg("graceful shutdown done")
		return nil
	}
}

// pingNats will attempt to connect to nats, blocking until a connection is established
func pingNats(cfg *occfg.Config) error {
	// We need to get a natsconfig from somewhere. We can use any one.
	evcfg := cfg.Postprocessing.Postprocessing.Events
	_, err := stream.NatsFromConfig("initial", true, stream.NatsConfig{
		Endpoint:             evcfg.Endpoint,
		Cluster:              evcfg.Cluster,
		EnableTLS:            evcfg.EnableTLS,
		TLSInsecure:          evcfg.TLSInsecure,
		TLSRootCACertificate: evcfg.TLSRootCACertificate,
		AuthUsername:         evcfg.AuthUsername,
		AuthPassword:         evcfg.AuthPassword,
	})
	return err
}

func pingGateway(cfg *occfg.Config) error {
	// init grpc connection
	_, err := ogrpc.NewClient()
	if err != nil {
		return err
	}

	b := backoff.NewExponentialBackOff()
	o := func() error {
		n := b.NextBackOff()
		_, err := pool.GetGatewayServiceClient(cfg.Reva.Address)
		if err != nil && n > time.Second {
			logger.New().Error().Err(err).Dur("backoff", n).Msg("can't connect to gateway service, retrying")
		}
		return err
	}

	err = backoff.Retry(o, b)
	return err
}

func wait(d time.Duration) func(cfg *occfg.Config) error {
	return func(cfg *occfg.Config) error {
		time.Sleep(d)
		return nil
	}
}
