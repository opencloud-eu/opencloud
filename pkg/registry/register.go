package registry

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	mRegistry "go-micro.dev/v4/registry"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

// RegisterService publishes an arbitrary endpoint to the service-registry. This allows querying nodes of
// non-micro services like reva. No health-checks are done, thus the caller is responsible for canceling.
func RegisterService(ctx context.Context, logger log.Logger, service *mRegistry.Service, debugAddr string) error {
	registry := GetRegistry()
	node := service.Nodes[0]

	logger.Info().Msgf("registering external service %v@%v", node.Id, node.Address)

	rOpts := []mRegistry.RegisterOption{mRegistry.RegisterTTL(GetRegisterTTL())}
	if err := registry.Register(service, rOpts...); err != nil {
		logger.Fatal().Err(err).Strs("registry_addresses", redactAddrs(registry.Options().Addrs)).Msgf("Registration error for external service %v", service.Name)
	}

	t := time.NewTicker(GetRegisterInterval())

	go func() {
		// check if the service is ready
		delay := 500 * time.Millisecond
		for {
			resp, err := http.DefaultClient.Get("http://" + debugAddr + "/readyz")
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				break
			}
			time.Sleep(delay)
			delay *= 2
		}
		for {
			select {
			case <-t.C:
				logger.Debug().Interface("service", service).Msg("refreshing external service-registration")
				err := registry.Register(service, rOpts...)
				if err != nil {
					logger.Error().Err(err).Msgf("registration error for external service %v", service.Name)
				}
			case <-ctx.Done():
				logger.Debug().Interface("service", service).Msg("unregistering")
				t.Stop()
				err := registry.Deregister(service)
				if err != nil {
					logger.Err(err).Msgf("Error unregistering external service %v", service.Name)
				}
				return
			}
		}
	}()

	return nil
}

// redactAddrs masks passwords in registry addresses like nats://user:pass@host:9233
func redactAddrs(addrs []string) []string {
	redacted := make([]string, 0, len(addrs))
	for _, a := range addrs {
		redacted = append(redacted, redactAddr(a))
	}
	return redacted
}

func redactAddr(addr string) string {
	// nats adds the scheme to addresses without one, so user:pass@host:9233 is valid
	prefix := ""
	if !strings.Contains(addr, "://") {
		prefix = "nats://"
	}

	u, err := url.Parse(prefix + addr)
	if err != nil {
		// we can't tell where the password is, so hide everything before the host
		if i := strings.LastIndex(addr, "@"); i >= 0 {
			return "xxxxx" + addr[i:]
		}
		return addr
	}
	if u.User == nil {
		return addr
	}
	return strings.TrimPrefix(u.Redacted(), prefix)
}
