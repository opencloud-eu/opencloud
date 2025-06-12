package service

import (
	"context"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/email"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/store/defaults"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/rs/zerolog"
)

func (s eventsNotifier) handleSpaceShared(e events.SpaceShared, eventId string) {
	logger := s.logger.With().
		Str("event", "SpaceShared").
		Str("itemid", e.ID.OpaqueId).
		Logger()
	executant, spaceName, shareLink, ctx, err := s.prepareSpaceShared(logger, e)
	if err != nil {
		logger.Error().Err(err).Msg("could not prepare vars for email")
		return
	}

	granteeList := s.ensureGranteeList(ctx, executant.GetId(), e.GranteeUserID, e.GranteeGroupID)
	filteredGrantees := s.filter.execute(ctx, granteeList, defaults.SettingUUIDProfileEventSpaceShared)

	recipientsInstant, recipientsDaily, recipientsInstantWeekly := s.splitter.execute(ctx, filteredGrantees)
	recipientsInstant = append(recipientsInstant, s.userEventStore.persist(_intervalDaily, eventId, recipientsDaily)...)
	recipientsInstant = append(recipientsInstant, s.userEventStore.persist(_intervalWeekly, eventId, recipientsInstantWeekly)...)
	if recipientsInstant == nil {
		return
	}

	sharerDisplayName := executant.GetDisplayName()
	emails, err := s.render(ctx, email.SharedSpace,
		"SpaceGrantee",
		map[string]string{
			"SpaceSharer": sharerDisplayName,
			"SpaceName":   spaceName,
			"ShareLink":   shareLink,
		}, recipientsInstant, sharerDisplayName)
	if err != nil {
		logger.Error().Err(err).Msg("could not get render the email")
		return
	}
	s.send(ctx, emails)
}

func (s eventsNotifier) prepareSpaceShared(logger zerolog.Logger, e events.SpaceShared) (executant *user.User, spaceName, shareLink string, ctx context.Context, err error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		logger.Error().Err(err).Msg("could not select next gateway client")
		return executant, spaceName, shareLink, ctx, err
	}

	ctx, err = utils.GetServiceUserContextWithContext(context.Background(), gatewayClient, s.serviceAccountID, s.serviceAccountSecret)
	if err != nil {
		logger.Error().Err(err).Msg("could not get service user context")
		return executant, spaceName, shareLink, ctx, err
	}

	executant, err = utils.GetUserNoGroups(ctx, e.Executant, gatewayClient)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("could not get user")
		return executant, spaceName, shareLink, ctx, err
	}

	resourceID, err := storagespace.ParseID(e.ID.OpaqueId)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("could not parse resourceid from ItemID ")
		return executant, spaceName, shareLink, ctx, err
	}

	resourceInfo, err := s.getResourceInfo(ctx, &resourceID, nil)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("could not get space info")
		return executant, spaceName, shareLink, ctx, err
	}
	spaceName = resourceInfo.GetSpace().GetName()

	shareLink, err = urlJoinPath(s.openCloudURL, "f", e.ID.OpaqueId)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("could not create link to the share")
		return executant, spaceName, shareLink, ctx, err
	}
	return executant, spaceName, shareLink, ctx, err
}

func (s eventsNotifier) handleSpaceUnshared(e events.SpaceUnshared, eventId string) {
	logger := s.logger.With().
		Str("event", "SpaceUnshared").
		Str("itemid", e.ID.OpaqueId).
		Logger()

	executant, spaceName, shareLink, ctx, err := s.prepareSpaceUnshared(logger, e)
	if err != nil {
		logger.Error().Err(err).Msg("could not prepare vars for email")
		return
	}

	granteeList := s.ensureGranteeList(ctx, executant.GetId(), e.GranteeUserID, e.GranteeGroupID)
	filteredGrantees := s.filter.execute(ctx, granteeList, defaults.SettingUUIDProfileEventSpaceUnshared)

	recipientsInstant, recipientsDaily, recipientsInstantWeekly := s.splitter.execute(ctx, filteredGrantees)
	recipientsInstant = append(recipientsInstant, s.userEventStore.persist(_intervalDaily, eventId, recipientsDaily)...)
	recipientsInstant = append(recipientsInstant, s.userEventStore.persist(_intervalWeekly, eventId, recipientsInstantWeekly)...)
	if recipientsInstant == nil {
		return
	}

	sharerDisplayName := executant.GetDisplayName()
	emails, err := s.render(ctx, email.UnsharedSpace,
		"SpaceGrantee",
		map[string]string{
			"SpaceSharer": sharerDisplayName,
			"SpaceName":   spaceName,
			"ShareLink":   shareLink,
		}, recipientsInstant, sharerDisplayName)
	if err != nil {
		logger.Error().Err(err).Msg("Could not get render the email")
		return
	}
	s.send(ctx, emails)
}

func (s eventsNotifier) prepareSpaceUnshared(logger zerolog.Logger, e events.SpaceUnshared) (executant *user.User, spaceName, shareLink string, ctx context.Context, err error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		logger.Error().Err(err).Msg("could not select next gateway client")
		return executant, spaceName, shareLink, ctx, err
	}

	ctx, err = utils.GetServiceUserContextWithContext(context.Background(), gatewayClient, s.serviceAccountID, s.serviceAccountSecret)
	if err != nil {
		logger.Error().Err(err).Msg("could not get service user context")
		return executant, spaceName, shareLink, ctx, err
	}

	executant, err = utils.GetUserNoGroups(ctx, e.Executant, gatewayClient)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("could not get user")
		return executant, spaceName, shareLink, ctx, err
	}

	resourceID, err := storagespace.ParseID(e.ID.OpaqueId)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("could not parse resourceid from ItemID ")
		return executant, spaceName, shareLink, ctx, err
	}

	resourceInfo, err := s.getResourceInfo(ctx, &resourceID, nil)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("could not get space info")
		return executant, spaceName, shareLink, ctx, err
	}
	spaceName = resourceInfo.GetSpace().GetName()

	shareLink, err = urlJoinPath(s.openCloudURL, "f", e.ID.OpaqueId)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("could not create link to the share")
		return executant, spaceName, shareLink, ctx, err
	}
	return executant, spaceName, shareLink, ctx, err
}

func (s eventsNotifier) handleSpaceMembershipExpired(e events.SpaceMembershipExpired, eventId string) {
	logger := s.logger.With().
		Str("event", "SpaceMembershipExpired").
		Str("itemid", e.SpaceID.GetOpaqueId()).
		Logger()

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		logger.Error().Err(err).Msg("could not select next gateway client")
		return
	}

	ctx, err := utils.GetServiceUserContext(s.serviceAccountID, gatewayClient, s.serviceAccountSecret)
	if err != nil {
		logger.Error().Err(err).Msg("Could not impersonate sharer")
		return
	}

	owner, err := utils.GetUserNoGroups(ctx, e.SpaceOwner, gatewayClient)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("could not get user")
		return
	}

	granteeList := s.ensureGranteeList(ctx, owner.GetId(), e.GranteeUserID, e.GranteeGroupID)
	if granteeList == nil {
		return
	}
	filteredGrantees := s.filter.execute(ctx, granteeList, defaults.SettingUUIDProfileEventSpaceMembershipExpired)

	recipientsInstant, recipientsDaily, recipientsInstantWeekly := s.splitter.execute(ctx, filteredGrantees)
	recipientsInstant = append(recipientsInstant, s.userEventStore.persist(_intervalDaily, eventId, recipientsDaily)...)
	recipientsInstant = append(recipientsInstant, s.userEventStore.persist(_intervalWeekly, eventId, recipientsInstantWeekly)...)
	if recipientsInstant == nil {
		return
	}

	emails, err := s.render(ctx, email.MembershipExpired,
		"SpaceGrantee",
		map[string]string{
			"SpaceName": e.SpaceName,
			"ExpiredAt": e.ExpiredAt.Format("2006-01-02 15:04:05"),
		}, recipientsInstant, owner.GetDisplayName())
	if err != nil {
		logger.Error().Err(err).Msg("could not get render the email")
		return
	}
	s.send(ctx, emails)
}
