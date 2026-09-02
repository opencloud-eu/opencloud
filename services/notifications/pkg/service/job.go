package service

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/l10n"
	ehmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/eventhistory/v0"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/channels"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/email"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/rs/zerolog"
)

func (s eventsNotifier) sendGroupedEmailsJob(sendEmailsEvent events.SendEmailsEvent, eventId string) {
	logger := s.logger.With().
		Str("event", "SendEmailsEvent").
		Str("eventId", eventId).
		Logger()

	if sendEmailsEvent.Interval != _intervalDaily && sendEmailsEvent.Interval != _intervalWeekly {
		logger.Error().Str("interval", sendEmailsEvent.Interval).Msg("unsupported email sending interval")
		return
	}

	keys, err := s.userEventStore.listKeys(sendEmailsEvent.Interval)
	if err != nil {
		logger.Error().Err(err).Msg("could not get list of keys")
		return
	}

	ctx := context.Background()

	jobs := make(chan string, 10)
	go func() {
		for _, key := range keys {
			jobs <- key
		}
		close(jobs)
	}()

	for job := range jobs {
		go s.createGroupedMail(ctx, logger, job)
	}
}

func (s eventsNotifier) createGroupedMail(ctx context.Context, logger zerolog.Logger, key string) {
	userEvents, err := s.userEventStore.pop(ctx, key)
	if err != nil {
		logger.Error().Err(err).Str("key", key).Msg("could not pop user events")
		return
	}

	var mts []email.MessageTemplate
	var mtsVars []map[string]string
	locale := l10n.MustGetUserLocale(ctx, userEvents.User.GetId().GetOpaqueId(), "", s.valueService)

	for _, e := range userEvents.Events {
		switch te := s.unwrapEvent(logger, e).(type) {
		case events.SpaceShared:
			logger := logger.With().
				Str("event", "SpaceShared").
				Str("eventId", te.ID.OpaqueId).
				Logger()

			executant, spaceName, shareLink, _, err := s.prepareSpaceShared(logger, te)
			if err != nil {
				logger.Error().Err(err).Msg("could not prepare vars for grouped email")
				continue
			}

			mts = append(mts, email.SharedSpace)
			mtsVars = append(mtsVars, map[string]string{
				"SpaceSharer": executant.GetDisplayName(),
				"SpaceName":   spaceName,
				"ShareLink":   shareLink,
			})
		case events.SpaceUnshared:
			logger := logger.With().
				Str("event", "SpaceUnshared").
				Str("eventId", te.ID.OpaqueId).
				Logger()

			executant, spaceName, shareLink, _, err := s.prepareSpaceUnshared(logger, te)
			if err != nil {
				logger.Error().Err(err).Msg("could not prepare vars for grouped email")
				continue
			}
			mts = append(mts, email.UnsharedSpace)
			mtsVars = append(mtsVars, map[string]string{
				"SpaceSharer": executant.GetDisplayName(),
				"SpaceName":   spaceName,
				"ShareLink":   shareLink,
			})
		case events.SpaceMembershipExpired:
			mts = append(mts, email.MembershipExpired)
			mtsVars = append(mtsVars, map[string]string{
				"SpaceName": te.SpaceName,
				"ExpiredAt": te.ExpiredAt.Format("2006-01-02 15:04:05"),
			})
		case events.ShareCreated:
			logger := logger.With().
				Str("event", "ShareCreated").
				Str("eventId", te.ItemID.OpaqueId).
				Logger()

			owner, shareFolder, shareLink, _, err := s.prepareShareCreated(logger, te)
			if err != nil {
				logger.Error().Err(err).Msg("could not prepare vars for grouped email")
				continue
			}
			mts = append(mts, email.ShareCreated)
			mtsVars = append(mtsVars, map[string]string{
				"ShareSharer": owner.GetDisplayName(),
				"ShareFolder": shareFolder,
				"ShareLink":   shareLink,
			})
		case events.ShareExpired:
			logger := logger.With().
				Str("event", "ShareCreated").
				Str("eventId", te.ItemID.OpaqueId).
				Logger()

			shareFolder, _, err := s.prepareShareExpired(logger, te)
			if err != nil {
				logger.Error().Err(err).Msg("could not prepare vars for grouped email")
				continue
			}
			mts = append(mts, email.ShareExpired)
			mtsVars = append(mtsVars, map[string]string{
				"ShareFolder": shareFolder,
				"ExpiredAt":   te.ExpiredAt.Format("2006-01-02 15:04:05"),
			})
		case events.ShareRemoved:
			logger := logger.With().
				Str("event", "ShareRemoved").
				Str("eventId", te.ItemID.OpaqueId).
				Logger()

			executant, shareFolder, _, err := s.prepareShareRemoved(logger, te)
			if err != nil {
				logger.Error().Err(err).Msg("could not prepare vars for grouped email")
				continue
			}

			mts = append(mts, email.ShareRemoved)
			mtsVars = append(mtsVars, map[string]string{
				"ShareSharer": executant.GetDisplayName(),
				"ShareFolder": shareFolder,
			})
		default:
			logger.Error().Str("eventType", e.Type).Msg("unsupported event type for grouped email")
		}
	}

	rendered, err := email.RenderGroupedEmailTemplate(email.Grouped, map[string]string{
		"DisplayName": userEvents.User.GetDisplayName(),
	}, locale, s.defaultLanguage, s.emailTemplatePath, s.translationPath, mts, mtsVars)
	if err != nil {
		logger.Error().Err(err).Msg("could not render template")
		return
	}
	rendered.Sender = s.defaultEmailSender
	rendered.Recipient = []string{userEvents.User.GetMail()}
	s.send(ctx, []*channels.Message{rendered})
}

func (s eventsNotifier) unwrapEvent(logger zerolog.Logger, e *ehmsg.Event) any {
	etype, ok := s.registeredEvents[e.GetType()]
	if !ok {
		logger.Error().Str("eventId", e.GetId()).Str("eventType", e.GetType()).Msg("event not registered")
		return nil
	}

	ue, err := etype.Unmarshal(e.GetEvent())
	if err != nil {
		logger.Error().Str("eventId", e.GetId()).Str("eventType", e.GetType()).Msg("failed to umarshal event")
		return nil
	}

	return ue
}
