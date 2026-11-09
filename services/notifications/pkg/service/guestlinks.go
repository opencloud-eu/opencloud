// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/rs/zerolog"
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

	s.sendGuestLinkMail(logger, email.GuestLinkShareCreated, e.Sharer, e.GranteeEmail, e.ItemID, e.ResourceName, e.Token, nil)
}

// handleGuestTokenRenewed sends the renewed guest link to the guest's email
// address, like handleGuestTokenCreated. The event only carries a PIN when the
// renewal was authorized by a guest session; otherwise just the link is sent.
func (s eventsNotifier) handleGuestTokenRenewed(e ocevents.GuestTokenRenewed) {
	logger := s.logger.With().
		Str("event", "GuestTokenRenewed").
		Str("itemid", e.ItemID.GetOpaqueId()).
		Logger()

	if e.Pin == "" {
		s.sendGuestLinkMail(logger, email.GuestLinkShareRenewedWithoutPin, e.Sharer, e.GranteeEmail, e.ItemID, "", e.Token, nil)
		return
	}

	s.sendGuestLinkMail(logger, email.GuestLinkShareRenewed, e.Sharer, e.GranteeEmail, e.ItemID, "", e.Token,
		map[string]string{"SharePin": e.Pin})
}

// sendGuestLinkMail renders the guest link template and sends it to the guest.
// The resource is stat'ed when resourceName is empty. vars are passed to the
// template in addition to ShareSharer, ShareFolder and ShareLink.
func (s eventsNotifier) sendGuestLinkMail(logger zerolog.Logger, template email.MessageTemplate, sharer *user.UserId,
	granteeEmail string, itemID *provider.ResourceId, resourceName, token string, vars map[string]string) {
	if err := validate.Var(granteeEmail, "required,email"); err != nil {
		logger.Error().Err(err).Msg("invalid guest email address")
		return
	}
	if token == "" {
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

	owner, err := utils.GetUserNoGroups(ctx, sharer, gatewayClient)
	if err != nil {
		logger.Error().Err(err).Msg("could not get user")
		return
	}

	shareFolder := resourceName
	if shareFolder == "" {
		resourceInfo, err := s.getResourceInfo(ctx, itemID, &fieldmaskpb.FieldMask{Paths: []string{"name"}})
		if err != nil {
			logger.Error().Err(err).Msg("could not stat resource")
			return
		}
		shareFolder = resourceInfo.GetName()
	}

	shareLink, err := urlJoinPath(s.openCloudURL, "g", token)
	if err != nil {
		logger.Error().Err(err).Msg("could not create guest link")
		return
	}

	templateVars := map[string]string{
		"ShareSharer": owner.GetDisplayName(),
		"ShareFolder": shareFolder,
		"ShareLink":   shareLink,
	}
	for k, v := range vars {
		templateVars[k] = v
	}

	msg, err := email.RenderEmailTemplate(template, s.defaultLanguage, s.defaultLanguage,
		s.emailTemplatePath, s.translationPath, templateVars)
	if err != nil {
		logger.Error().Err(err).Msg("could not render the email")
		return
	}
	msg.Sender = owner.GetDisplayName()
	msg.Recipient = []string{granteeEmail}

	s.send(ctx, []*channels.Message{msg})
}
