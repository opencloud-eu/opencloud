// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"

	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	ocevents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/channels"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/email"
)

// handleGuestTokenCreated sends the guest link to the guest's email address.
// Guests have no account, settings or locale, so the mail is always sent
// instantly (the token is short-lived) in the default language.
func (s eventsNotifier) handleGuestTokenCreated(e ocevents.GuestTokenCreated) {
	logger := s.logger.With().
		Str("event", "GuestTokenCreated").
		Str("itemid", e.ItemID.GetOpaqueId()).
		Logger()

	if err := validate.Var(e.GranteeEmail, "required,email"); err != nil {
		logger.Error().Err(err).Msg("invalid guest email address")
		return
	}
	if e.Token == "" {
		logger.Error().Msg("guest token is empty")
		return
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		logger.Error().Err(err).Msg("could not select next gateway client")
		return
	}

	ctx, err := utils.GetServiceUserContextWithContext(context.Background(), gatewayClient, s.serviceAccountID, s.serviceAccountSecret)
	if err != nil {
		logger.Error().Err(err).Msg("could not get service user context")
		return
	}

	owner, err := utils.GetUserNoGroups(ctx, e.Sharer, gatewayClient)
	if err != nil {
		logger.Error().Err(err).Msg("could not get user")
		return
	}

	shareFolder := e.ResourceName
	if shareFolder == "" {
		resourceInfo, err := s.getResourceInfo(ctx, e.ItemID, &fieldmaskpb.FieldMask{Paths: []string{"name"}})
		if err != nil {
			logger.Error().Err(err).Msg("could not stat resource")
			return
		}
		shareFolder = resourceInfo.GetName()
	}

	shareLink, err := urlJoinPath(s.openCloudURL, "g", e.Token)
	if err != nil {
		logger.Error().Err(err).Msg("could not create guest link")
		return
	}

	msg, err := email.RenderEmailTemplate(email.GuestLinkShareCreated, s.defaultLanguage, s.defaultLanguage,
		s.emailTemplatePath, s.translationPath,
		map[string]string{
			"ShareSharer": owner.GetDisplayName(),
			"ShareFolder": shareFolder,
			"ShareLink":   shareLink,
		})
	if err != nil {
		logger.Error().Err(err).Msg("could not render the email")
		return
	}
	msg.Sender = owner.GetDisplayName()
	msg.Recipient = []string{e.GranteeEmail}

	s.send(ctx, []*channels.Message{msg})
}
