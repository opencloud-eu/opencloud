package service

import (
	"context"
	"net/http"
	"time"

	"github.com/tmaxmax/go-sse"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/sse/pkg/config"
)

const (
	SSETopicAllUsers = "all"
)

// SSEHandler defines implements the business logic for Service.
type SSEHandler struct {
	conf    *config.Config
	logger  log.Logger
	server  *sse.Server
	channel <-chan events.Event
}

// NewSSEHandler returns a service implementation for Service.
func NewSSEHandler(ctx context.Context, conf *config.Config, logger log.Logger, ch <-chan events.Event) (SSEHandler, error) {
	handler := SSEHandler{
		conf:    conf,
		logger:  logger,
		channel: ch,
	}

	handler.server = &sse.Server{
		OnSession: func(_ http.ResponseWriter, r *http.Request) (topics []string, allowed bool) {
			return handler.topics(r)
		},
	}

	go func() {
		select {
		case <-ctx.Done():
			if err := handler.server.Shutdown(ctx); err != nil {
				logger.Error().Err(err).Msg("failed to shutdown SSE handler")
			}
			return
		}
	}()

	go handler.listen()

	return handler, nil
}

// ServeHTTP fulfills Handler interface
func (h SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	topics, ok := h.topics(r)
	if !ok {
		h.logger.Error().Msg("sse: failed to get topics")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if h.conf.KeepAliveInterval != 0 {
		ticker := time.NewTicker(h.conf.KeepAliveInterval)
		defer ticker.Stop()
		go func() {
			for range ticker.C {
				m := &sse.Message{}
				m.AppendData("keep-alive")
				if err := h.server.Publish(m, topics...); err != nil {
					h.logger.Error().Err(err).Msg("sse: failed to publish message")
				}
			}
		}()
	}

	h.server.ServeHTTP(w, r)
}

// ListenForEvents listens for events
func (h SSEHandler) listen() {
	for e := range h.channel {
		switch ev := e.Event.(type) {
		default:
			h.logger.Error().Interface("event", ev).Msg("unhandled event")
		case events.SendSSE:
			m := &sse.Message{
				Type: sse.Type(ev.Type),
			}
			m.AppendData(string(ev.Message))
			if err := h.server.Publish(m, ev.UserIDs...); err != nil {
				h.logger.Error().Err(err).Msg("sse: failed to publish message")
			}
		}
	}
}

func (h SSEHandler) topics(r *http.Request) ([]string, bool) {
	u, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		return nil, false
	}

	uid := u.GetId().GetOpaqueId()
	if uid == "" {
		return nil, false
	}

	return append([]string{SSETopicAllUsers}, uid), true
}
