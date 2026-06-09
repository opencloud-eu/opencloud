package service

import (
	"context"
	"errors"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	micrometadata "go-micro.dev/v4/metadata"

	ocEvents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/store/defaults"
)

type userlogFilter struct {
	log         log.Logger
	valueClient settingssvc.ValueService
}

func newUserlogFilter(l log.Logger, vc settingssvc.ValueService) *userlogFilter {
	return &userlogFilter{log: l, valueClient: vc}
}

// execute removes users who should not receive an in-app notifications for the event
func (ulf userlogFilter) execute(ctx context.Context, event events.Event, executant *user.UserId, users []string) []string {
	filteredUsers := ulf.filterExecutant(users, executant)
	return ulf.filterUsersBySettings(ctx, filteredUsers, event)
}

// filterExecutant removes the executant
func (ulf userlogFilter) filterExecutant(users []string, executant *user.UserId) []string {
	var filteredUsers []string
	for _, u := range users {
		if u != executant.GetOpaqueId() {
			filteredUsers = append(filteredUsers, u)
		}
	}
	return filteredUsers
}

// filterUsersBySettings removes users who have disabled in-app notifications for the event
func (ulf userlogFilter) filterUsersBySettings(ctx context.Context, users []string, event events.Event) []string {
	var filteredUsers []string
	var settingId string
	// map type to settings key
	switch event.Event.(type) {
	case events.ShareCreated:
		settingId = defaults.SettingUUIDProfileEventShareCreated
	case events.ShareRemoved:
		settingId = defaults.SettingUUIDProfileEventShareRemoved
	case events.ShareExpired:
		settingId = defaults.SettingUUIDProfileEventShareExpired
	case events.SpaceShared:
		settingId = defaults.SettingUUIDProfileEventSpaceShared
	case events.SpaceUnshared:
		settingId = defaults.SettingUUIDProfileEventSpaceUnshared
	case events.SpaceMembershipExpired:
		settingId = defaults.SettingUUIDProfileEventSpaceMembershipExpired
	case events.SpaceDisabled:
		settingId = defaults.SettingUUIDProfileEventSpaceDisabled
	case events.SpaceDeleted:
		settingId = defaults.SettingUUIDProfileEventSpaceDeleted
	case ocEvents.ResourceMention:
		settingId = defaults.SettingUUIDProfileEventResourceMention
	default:
		// event that cannot be disabled
		return users
	}

	for _, u := range users {
		enabled, err := getSetting(ctx, ulf.valueClient, u, settingId)
		if err != nil {
			ulf.log.Error().Err(err).Str("userId", u).Str("settingId", settingId).Msg("cannot get user event setting")
			continue
		}
		if enabled {
			filteredUsers = append(filteredUsers, u)
		}
	}

	return filteredUsers
}

func getSetting(ctx context.Context, vc settingssvc.ValueService, userId string, settingId string) (bool, error) {
	resp, err := vc.GetValueByUniqueIdentifiers(
		micrometadata.Set(ctx, middleware.AccountID, userId),
		&settingssvc.GetValueByUniqueIdentifiersRequest{
			AccountUuid: userId,
			SettingId:   settingId,
		},
	)

	if err != nil {
		return false, err
	}

	val := resp.GetValue().GetValue().GetCollectionValue().GetValues()
	for _, option := range val {
		if option.GetKey() == "in-app" {
			return option.GetBoolValue(), nil
		}
	}
	return false, errors.New("no setting found")
}
