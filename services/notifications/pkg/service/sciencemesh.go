package service

import (
	"context"

	"github.com/opencloud-eu/opencloud/services/notifications/pkg/channels"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/email"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

func (s eventsNotifier) handleScienceMeshInviteTokenGenerated(e events.ScienceMeshInviteTokenGenerated) {
	logger := s.logger.With().
		Str("event", "ScienceMeshInviteTokenGenerated").
		Logger()

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		logger.Error().Err(err).Msg("could not select next gateway client")
		return
	}

	ctx, err := utils.GetServiceUserContextWithContext(context.Background(), gatewayClient, s.serviceAccountID, s.serviceAccountSecret)
	if err != nil {
		logger.Error().Err(err).Msg("Could not impersonate service user")
		return
	}

	owner, err := utils.GetUserNoGroups(ctx, e.Sharer, gatewayClient)
	if err != nil {
		logger.Error().Err(err).Msg("unable to get user")
		return
	}

	msgENV := map[string]string{
		"ShareSharer":     owner.GetDisplayName(),
		"ShareSharerMail": owner.GetMail(),
		"ShareLink":       e.InviteLink,
		"Token":           e.Token,
		"ProviderDomain":  owner.GetId().GetIdp(),
		"RecipientMail":   e.RecipientMail,
	}

	// validate the message, we only need recipient mail at the moment,
	// event that is optional when the event got triggered...
	// this means if we get a validation error, we can't send the message and skip it
	{
		validationEnv := make(map[string]interface{}, len(msgENV))
		for k, v := range msgENV {
			validationEnv[k] = v
		}
		if errs := validate.ValidateMap(validationEnv,
			map[string]interface{}{
				"RecipientMail": "required,email", // only recipient mail is required to send the message
			}); len(errs) > 0 {
			return // no mail, no message
		}
	}

	emailTpl := email.ScienceMeshInviteTokenGenerated
	if e.InviteLink == "" {
		emailTpl = email.ScienceMeshInviteTokenGeneratedWithoutShareLink
	}

	msg, err := email.RenderEmailTemplate(
		emailTpl,
		s.defaultLanguage, // fixMe: the recipient is unknown, should it be the defaultLocale?,
		s.defaultLanguage, // fixMe: the defaultLocale is not set by default, shouldn't it be?,
		s.emailTemplatePath,
		s.translationPath,
		msgENV,
	)
	if err != nil {
		logger.Error().Err(err).Msg("building the message has failed")
		return
	}

	msg.Sender = owner.GetDisplayName()
	msg.Recipient = []string{e.RecipientMail}

	s.send(ctx, []*channels.Message{msg})
}
