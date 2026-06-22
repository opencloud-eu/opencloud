package service

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/r3labs/sse/v2"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/sse/pkg/config"
)

// SSE defines implements the business logic for Service.
type SSE struct {
	c         *config.Config
	l         log.Logger
	m         *chi.Mux
	sse       *sse.Server
	evChannel <-chan events.Event
}

// NewSSE returns a service implementation for Service.
func NewSSE(c *config.Config, l log.Logger, ch <-chan events.Event, mux *chi.Mux) (SSE, error) {
	s := SSE{
		c:         c,
		l:         l,
		m:         mux,
		sse:       sse.New(),
		evChannel: ch,
	}
	mux.Route("/ocs/v2.php/apps/notifications/api/v1/notifications", func(r chi.Router) {
		r.Get("/sse", s.HandleSSE)
	})

	go s.ListenForEvents()

	return s, nil
}

// ServeHTTP fulfills Handler interface
func (s SSE) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.ServeHTTP(w, r)
}

// ListenForEvents listens for events
func (s SSE) ListenForEvents() {
	for e := range s.evChannel {
		go func() {
			switch ev := e.Event.(type) {
			default:
				s.l.Error().Interface("event", ev).Msg("unhandled event")
			case events.SendSSE:
				for _, uid := range ev.UserIDs {
					s.sse.Publish(uid, &sse.Event{
						Event: []byte(ev.Type),
						Data:  ev.Message,
					})
				}
			}
		}()
	}
}

// HandleSSE is the GET handler for events
func (s SSE) HandleSSE(w http.ResponseWriter, r *http.Request) {
	u, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		s.l.Error().Msg("sse: no user in context")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	uid := u.GetId().GetOpaqueId()
	if uid == "" {
		s.l.Error().Msg("sse: user in context is broken")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	stream := s.sse.CreateStream(uid)
	stream.AutoReplay = false

	if s.c.KeepAliveInterval != 0 {
		ticker := time.NewTicker(s.c.KeepAliveInterval)
		defer ticker.Stop()
		go func() {
			for range ticker.C {
				s.sse.Publish(uid, &sse.Event{
					Comment: []byte("keepalive"),
				})
			}
		}()
	}

	// add stream to URL
	q := r.URL.Query()
	q.Set("stream", uid)
	r.URL.RawQuery = q.Encode()

	s.sse.ServeHTTP(w, r)
}
