package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocEvents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/pkg/l10n"
	"github.com/opencloud-eu/opencloud/pkg/log"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/config"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/service"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// Service consumes events and stores them for the users
type Service struct {
	ctx              context.Context
	log              log.Logger
	cfg              *config.Config
	userlog          *service.UserlogService
	stream           events.Stream
	gatewaySelector  pool.Selectable[gateway.GatewayAPIClient]
	valueClient      settingssvc.ValueService
	registeredEvents []events.Unmarshaller
	filter           *userlogFilter
	stopCh           chan struct{}
	stopped          atomic.Bool
}

// Run to fulfil Runner interface
func (s *Service) Run() error {
	ch, err := events.Consume(s.stream, "userlog", s.registeredEvents...)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	for i := 0; i < s.cfg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-ch:
					if !ok {
						return
					}
					go s.processEvent(event)
				}
			}
		}()
	}

	<-s.stopCh
	cancel()
	wg.Wait()

	return nil
}

// Close will make the service to stop processing, so the `Run`
// method can finish.
func (s *Service) Close() {
	if s.stopped.CompareAndSwap(false, true) {
		close(s.stopCh)
	}
}

func (s *Service) processEvent(event events.Event) {
	// for each event we need to:
	// I) find users eligible to receive the event
	var (
		users     []string
		executant *user.UserId
		err       error
	)

	gwc, err := s.gatewaySelector.Next()
	if err != nil {
		s.log.Error().Err(err).Msg("cannot get gateway client")
		return
	}

	ctx, err := utils.GetServiceUserContext(s.cfg.ServiceAccount.ServiceAccountID, gwc, s.cfg.ServiceAccount.ServiceAccountSecret)
	if err != nil {
		s.log.Error().Err(err).Msg("cannot get service account")
		return
	}

	gwc, err = s.gatewaySelector.Next()
	if err != nil {
		s.log.Error().Err(err).Msg("cannot get gateway client")
		return
	}
	switch e := event.Event.(type) {
	default:
		err = errors.New("unhandled event")
	// file related
	case events.PostprocessingStepFinished:
		switch e.FinishedStep {
		case events.PPStepAntivirus:
			result := e.Result.(events.VirusscanResult)
			if !result.Infected {
				return
			}

			// TODO: should space mangers also be informed?
			users = append(users, e.ExecutingUser.GetId().GetOpaqueId())
		case events.PPStepPolicies:
			if e.Outcome == events.PPOutcomeContinue {
				return
			}
			users = append(users, e.ExecutingUser.GetId().GetOpaqueId())
		default:
			return
		}

	// space related // TODO: how to find spaceadmins?
	case events.SpaceDisabled:
		executant = e.Executant
		users, err = utils.GetSpaceMembers(ctx, e.ID.GetOpaqueId(), gwc, utils.ViewerRole)
	case events.SpaceDeleted:
		executant = e.Executant
		for u := range e.FinalMembers {
			users = append(users, u)
		}
	case events.SpaceShared:
		executant = e.Executant
		users, err = utils.ResolveID(ctx, e.GranteeUserID, e.GranteeGroupID, gwc)
	case ocEvents.ResourceMention:
		executant = e.Executant
		for _, userID := range e.UserIDs {
			users = append(users, userID.GetOpaqueId())
		}
	case events.SpaceUnshared:
		executant = e.Executant
		users, err = utils.ResolveID(ctx, e.GranteeUserID, e.GranteeGroupID, gwc)
	case events.SpaceMembershipExpired:
		users, err = utils.ResolveID(ctx, e.GranteeUserID, e.GranteeGroupID, gwc)

	// share related
	case events.ShareCreated:
		executant = e.Executant
		users, err = utils.ResolveID(ctx, e.GranteeUserID, e.GranteeGroupID, gwc)
	case events.ShareRemoved:
		executant = e.Executant
		users, err = utils.ResolveID(ctx, e.GranteeUserID, e.GranteeGroupID, gwc)
	case events.ShareExpired:
		users, err = utils.ResolveID(ctx, e.GranteeUserID, e.GranteeGroupID, gwc)
	}

	if err != nil {
		// TODO: Find out why this errors on ci pipeline
		s.log.Debug().Err(err).Interface("event", event).Msg("error gathering members for event")
		return
	}

	// II) filter users who want to receive the event
	users = s.filter.execute(ctx, event, executant, users)

	// III) store the eventID for each user
	for _, id := range users {
		if err := s.userlog.AddEventToUser(id, event); err != nil {
			s.log.Error().Err(err).Str("userID", id).Str("eventid", event.ID).Msg("failed to store event for user")
			return
		}
	}

	// IV) send sses
	if !s.cfg.DisableSSE {
		if err := s.sendSSE(ctx, users, event); err != nil {
			s.log.Error().Err(err).Interface("userid", users).Str("eventid", event.ID).Msg("cannot create sse event")
		}
	}
}

func (s *Service) sendSSE(ctx context.Context, userIDs []string, event events.Event) error {
	m := make(map[string]events.SendSSE)

	for _, userid := range userIDs {
		loc := l10n.MustGetUserLocale(ctx, userid, "", s.valueClient)
		if ev, ok := m[loc]; ok {
			ev.UserIDs = append(m[loc].UserIDs, userid)
			m[loc] = ev
			continue
		}

		ev, err := service.NewConverter(ctx, loc, s.gatewaySelector, s.cfg.Service.Name, s.cfg.TranslationPath, s.cfg.DefaultLanguage).ConvertEvent(event.ID, event.Event)
		if err != nil {
			if utils.IsErrNotFound(err) || utils.IsErrPermissionDenied(err) {
				// the resource was not found, we assume it is deleted
				continue
			}
			return err
		}

		b, err := json.Marshal(ev)
		if err != nil {
			return err
		}

		m[loc] = events.SendSSE{
			UserIDs: []string{userid},
			Type:    "userlog-notification",
			Message: b,
		}

	}

	for _, ev := range m {
		if err := events.Publish(ctx, s.stream, ev); err != nil {
			return err
		}
	}

	return nil
}
