package service

import (
	"context"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/email"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/store/defaults"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func (s eventsNotifier) handleShareCreated(e events.ShareCreated, eventId string) {
	logger := s.logger.With().
		Str("event", "ShareCreated").
		Str("itemid", e.ItemID.OpaqueId).
		Logger()

	owner, shareFolder, shareLink, ctx, err := s.prepareShareCreated(logger, e)
	if err != nil {
		logger.Error().Err(err).Msg("could not prepare vars for email")
		return
	}

	granteeList := s.ensureGranteeList(ctx, owner.GetId(), e.GranteeUserID, e.GranteeGroupID)
	filteredGrantees := s.filter.execute(ctx, granteeList, defaults.SettingUUIDProfileEventShareCreated)

	recipientsInstant, recipientsDaily, recipientsInstantWeekly := s.splitter.execute(ctx, filteredGrantees)
	recipientsInstant = append(recipientsInstant, s.userEventStore.persist(_intervalDaily, eventId, recipientsDaily)...)
	recipientsInstant = append(recipientsInstant, s.userEventStore.persist(_intervalWeekly, eventId, recipientsInstantWeekly)...)
	if recipientsInstant == nil {
		return
	}

	sharerDisplayName := owner.GetDisplayName()
	emails, err := s.render(ctx, email.ShareCreated,
		"ShareGrantee",
		map[string]string{
			"ShareSharer": sharerDisplayName,
			"ShareFolder": shareFolder,
			"ShareLink":   shareLink,
		}, recipientsInstant, sharerDisplayName)
	if err != nil {
		logger.Error().Err(err).Msg("could not get render the email")
		return
	}
	s.send(ctx, emails)
}

func (s eventsNotifier) prepareShareCreated(logger zerolog.Logger, e events.ShareCreated) (owner *user.User, shareFolder, shareLink string, ctx context.Context, err error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		logger.Error().Err(err).Msg("could not select next gateway client")
		return owner, shareFolder, shareLink, ctx, err
	}

	ctx, err = utils.GetServiceUserContextWithContext(context.Background(), gatewayClient, s.serviceAccountID, s.serviceAccountSecret)
	if err != nil {
		logger.Error().Err(err).Msg("could not get service user context")
		return owner, shareFolder, shareLink, ctx, err
	}

	resourceInfo, err := s.getResourceInfo(ctx, e.ItemID, &fieldmaskpb.FieldMask{Paths: []string{"name"}})
	if err != nil {
		logger.Error().
			Err(err).
			Msg("could not stat resource")
		return owner, shareFolder, shareLink, ctx, err
	}
	shareFolder = resourceInfo.Name

	shareLink, err = urlJoinPath(s.openCloudURL, "files/shares/with-me")
	if err != nil {
		logger.Error().
			Err(err).
			Msg("could not create link to the share")
		return owner, shareFolder, shareLink, ctx, err
	}

	owner, err = utils.GetUserNoGroups(ctx, e.Sharer, gatewayClient)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("could not get user")
		return owner, shareFolder, shareLink, ctx, err
	}

	return owner, shareFolder, shareLink, ctx, err
}

func (s eventsNotifier) handleShareExpired(e events.ShareExpired, eventId string) {
	logger := s.logger.With().
		Str("event", "ShareExpired").
		Str("itemid", e.ItemID.GetOpaqueId()).
		Logger()

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		logger.Error().Err(err).Msg("could not select next gateway client")
		return
	}

	shareFolder, ctx, err := s.prepareShareExpired(logger, e)
	if err != nil {
		logger.Error().Err(err).Msg("could not prepare vars for email")
		return
	}

	owner, err := utils.GetUserNoGroups(ctx, e.ShareOwner, gatewayClient)
	if err != nil {
		logger.Error().Err(err).Msg("Could not get user")
		return
	}

	granteeList := s.ensureGranteeList(ctx, owner.GetId(), e.GranteeUserID, e.GranteeGroupID)
	filteredGrantees := s.filter.execute(ctx, granteeList, defaults.SettingUUIDProfileEventShareExpired)

	recipientsInstant, recipientsDaily, recipientsInstantWeekly := s.splitter.execute(ctx, filteredGrantees)
	recipientsInstant = append(recipientsInstant, s.userEventStore.persist(_intervalDaily, eventId, recipientsDaily)...)
	recipientsInstant = append(recipientsInstant, s.userEventStore.persist(_intervalWeekly, eventId, recipientsInstantWeekly)...)
	if recipientsInstant == nil {
		return
	}

	emails, err := s.render(ctx, email.ShareExpired,
		"ShareGrantee",
		map[string]string{
			"ShareFolder": shareFolder,
			"ExpiredAt":   e.ExpiredAt.Format("2006-01-02 15:04:05"),
		}, recipientsInstant, owner.GetDisplayName())
	if err != nil {
		logger.Error().Err(err).Msg("could not get render the email")
		return
	}
	s.send(ctx, emails)
}

func (s eventsNotifier) prepareShareExpired(logger zerolog.Logger, e events.ShareExpired) (shareFolder string, ctx context.Context, err error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		logger.Error().Err(err).Msg("could not select next gateway client")
		return shareFolder, ctx, err
	}

	ctx, err = utils.GetServiceUserContextWithContext(context.Background(), gatewayClient, s.serviceAccountID, s.serviceAccountSecret)
	if err != nil {
		logger.Error().Err(err).Msg("could not get service user context")
		return shareFolder, ctx, err
	}

	resourceInfo, err := s.getResourceInfo(ctx, e.ItemID, &fieldmaskpb.FieldMask{Paths: []string{"name"}})
	if err != nil {
		logger.Error().
			Err(err).
			Msg("could not stat resource")
		return shareFolder, ctx, err
	}
	shareFolder = resourceInfo.GetName()

	return shareFolder, ctx, err
}
