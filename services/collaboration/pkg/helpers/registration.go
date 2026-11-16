package helpers

import (
	"context"
	"errors"
	"time"

	registryv1beta1 "github.com/cs3org/go-cs3apis/cs3/app/registry/v1beta1"
	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
)

// RegisterOpenCloudService will register this service.
// There are no explicit requirements for the context, and it will be passed
// without changes to the underlying RegisterService method.
func RegisterOpenCloudService(ctx context.Context, cfg *config.Config, logger log.Logger) error {
	svc := registry.BuildGRPCService(cfg.GRPC.Namespace+"."+cfg.Service.Name, cfg.GRPC.Protocol, cfg.GRPC.Addr, version.GetString())
	return registry.RegisterService(ctx, logger, svc, cfg.Debug.Addr)
}

// RegisterAppProvider will register this service as app provider in REVA.
// The GatewayAPIClient is expected to be provided via `helpers.GetCS3apiClient`.
// The appUrls are expected to be provided via `helpers.GetAppURLs`
//
// Note that this method doesn't provide a re-registration mechanism, so it
// will register the service once
func RegisterAppProvider(
	ctx context.Context,
	cfg *config.Config,
	logger log.Logger,
	gws pool.Selectable[gatewayv1beta1.GatewayAPIClient],
	appUrls *AppURLs,
) error {
	mimeTypes := appUrls.GetMimeTypes()

	logger.Debug().
		Str("AppName", cfg.App.Name).
		Strs("Mimetypes", mimeTypes).
		Msg("Registering mimetypes in the app provider")

	// TODO: an added app provider shouldn't last forever. Instead the registry should use a TTL
	// and delete providers that didn't register again. If an app provider dies or get's disconnected,
	// the users will be no longer available to choose to open a file with it (currently, opening a file just fails)
	req := &registryv1beta1.AddAppProviderRequest{
		Provider: &registryv1beta1.ProviderInfo{
			Name:        cfg.App.Name,
			Description: cfg.App.Description,
			Icon:        cfg.App.Icon,
			Address:     cfg.GRPC.Namespace + "." + cfg.Service.Name,
			MimeTypes:   mimeTypes,
			ProductName: cfg.App.Product,
		},
	}
	gwc, err := gws.Next()
	if err != nil {
		return err
	}
	resp, err := gwc.AddAppProvider(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("AddAppProvider failed")
		return err
	}

	if resp.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		logger.Error().Str("status_code", resp.GetStatus().GetCode().String()).Msg("AddAppProvider failed")
		return errors.New("status code != CODE_OK")
	}

	return nil
}

// firstRetryDelay is how long RefreshAppProvider waits before it fetches the
// WOPI discovery again after a failed fetch. The delay doubles with every
// further failure, up to the registration interval. It's a variable only so
// that tests can shorten it.
var firstRetryDelay = 1 * time.Second

// RefreshAppProvider keeps the app URLs and the registration of this service
// as app provider up to date, until ctx is done: it fetches the app URLs from
// the WOPI app's discovery endpoint, stores them in appURLs and registers the
// app provider with their mimetypes, right away and then every
// cfg.CS3Api.APPRegistrationInterval.
//
// When a fetch fails, the fetch is retried sooner than the interval, and
// appURLs keeps the URLs of the last successful fetch for a while: the WOPI
// app may only be restarting, and they will most likely work again shortly.
// Once the retries have backed off to the full interval, i.e. the WOPI app
// has been unreachable for about one interval, the app URLs are cleared, so
// that the app isn't offered for opening documents while it is gone. The next
// successful fetch brings them back.
func RefreshAppProvider(
	ctx context.Context,
	cfg *config.Config,
	logger log.Logger,
	gws pool.Selectable[gatewayv1beta1.GatewayAPIClient],
	appURLs *AppURLs,
) {
	interval := cfg.CS3Api.APPRegistrationInterval
	retryDelay := firstRetryDelay
	cleared := false

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		next := interval
		if v, err := GetAppURLs(cfg, logger); err != nil {
			next = min(retryDelay, interval)
			retryDelay = min(2*retryDelay, interval)
			logger.Warn().Err(err).
				Bool("fetched_before", appURLs.Fetched()).
				Dur("retry_in", next).
				Msg("Failed to get app URLs")

			if next == interval && appURLs.Fetched() && !cleared {
				logger.Warn().Msg("The WOPI app has been unreachable for a while, clearing its app URLs until it answers again")
				appURLs.Store(map[string]map[string]string{})
				cleared = true
			}
		} else {
			appURLs.Store(v)
			retryDelay = firstRetryDelay
			cleared = false
		}

		if err := RegisterAppProvider(ctx, cfg, logger, gws, appURLs); err != nil {
			logger.Warn().Err(err).Msg("Failed to register app provider")
		}

		timer.Reset(next)
	}
}
