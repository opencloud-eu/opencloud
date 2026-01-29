package service

import (
	"context"
	"net/url"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/webfinger"
)

// NewLogging returns a service that logs messages.
func NewLogging(next Service, logger log.Logger) Service {
	return logging{
		next:   next,
		logger: logger,
	}
}

type logging struct {
	next   Service
	logger log.Logger
}

// Webfinger implements the Service interface.
func (l logging) Webfinger(ctx context.Context, queryTarget *url.URL, rels []string, platform string) (webfinger.JSONResourceDescriptor, error) {
	l.logger.Debug().
		Str("query_target", queryTarget.String()).
		Strs("rel", rels).
		Msg("Webfinger")

	return l.next.Webfinger(ctx, queryTarget, rels, platform)
}
