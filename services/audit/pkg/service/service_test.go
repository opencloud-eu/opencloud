package svc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/audit/pkg/types"
	"github.com/opencloud-eu/reva/v3/pkg/events"

	group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	rtypes "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

var testCases = []struct {
	Alias           string
	SystemEvent     events.Event
	CheckAuditEvent func(*testing.T, []byte)
}{
	{
		Alias: "ShareCreated - user",
		SystemEvent: events.Event{
			Event: events.ShareCreated{
				Sharer:         userID("sharing-userid"),
				GranteeUserID:  userID("beshared-userid"),
				GranteeGroupID: nil,
				ItemID:         resourceID("provider-1", "storage-1", "itemid-1"),
				CTime:          timestamp(0),
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventShareCreated{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "sharing-userid", "1970-01-01T00:00:00Z", "user 'sharing-userid' shared file 'itemid-1' with 'beshared-userid'", "file_shared")
			// AuditEventSharing fields
			checkSharingAuditEvent(t, ev.AuditEventSharing, "itemid-1", "sharing-userid", "")
			// AuditEventShareCreated fields
			require.Equal(t, "", ev.ItemType)
			require.Equal(t, "", ev.ExpirationDate)
			require.Equal(t, false, ev.SharePass)
			//require.Equal(t, "stat:true ", ev.Permissions) // TODO: BUG! Should work
			require.Equal(t, "user", ev.ShareType)
			require.Equal(t, "beshared-userid", ev.ShareWith)
			require.Equal(t, "sharing-userid", ev.ShareOwner)
			require.Equal(t, "", ev.ShareToken)
		},
	}, {
		Alias: "ShareCreated - group",
		SystemEvent: events.Event{
			Event: events.ShareCreated{
				Sharer:         userID("sharing-userid"),
				GranteeUserID:  nil,
				GranteeGroupID: groupID("beshared-groupid"),
				ItemID:         resourceID("provider-1", "storage-1", "itemid-1"),
				CTime:          timestamp(10e8),
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventShareCreated{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "sharing-userid", "2001-09-09T01:46:40Z", "user 'sharing-userid' shared file 'itemid-1' with 'beshared-groupid'", "file_shared")
			// AuditEventSharing fields
			checkSharingAuditEvent(t, ev.AuditEventSharing, "itemid-1", "sharing-userid", "")
			// AuditEventShareCreated fields
			require.Equal(t, "", ev.ItemType)
			require.Equal(t, "", ev.ExpirationDate)
			require.Equal(t, false, ev.SharePass)
			//require.Equal(t, "stat:true ", ev.Permissions) // TODO: BUG! Should work
			require.Equal(t, "group", ev.ShareType)
			require.Equal(t, "beshared-groupid", ev.ShareWith)
			require.Equal(t, "sharing-userid", ev.ShareOwner)
			require.Equal(t, "", ev.ShareToken)

		},
	}, {
		Alias: "ShareUpdated",
		SystemEvent: events.Event{
			Event: events.ShareUpdated{
				ShareID:        shareID("shareid"),
				Sharer:         userID("sharing-userid"),
				GranteeUserID:  nil,
				GranteeGroupID: groupID("beshared-groupid"),
				ItemID:         resourceID("provider-1", "storage-1", "itemid-1"),
				Permissions:    sharePermissions("stat", "get_quota"),
				MTime:          timestamp(10e8),
				Updated:        "permissions",
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventShareUpdated{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "sharing-userid", "2001-09-09T01:46:40Z", "user 'sharing-userid' updated field 'permissions' of share 'shareid'", "share_permission_updated")
			// AuditEventSharing fields
			checkSharingAuditEvent(t, ev.AuditEventSharing, "itemid-1", "sharing-userid", "shareid")
			// AuditEventShareUpdated fields
			require.Equal(t, "", ev.ItemType)       // not implemented atm
			require.Equal(t, "", ev.ExpirationDate) // no expiration for shares
			require.Equal(t, false, ev.SharePass)
			require.Equal(t, "get_quota:true stat:true", ev.Permissions)
			require.Equal(t, "group", ev.ShareType)
			require.Equal(t, "beshared-groupid", ev.ShareWith)
			require.Equal(t, "sharing-userid", ev.ShareOwner)
			require.Equal(t, "", ev.ShareToken) // token not filled for shares
		},
	}, {
		Alias: "LinkUpdated - permissions",
		SystemEvent: events.Event{
			Event: events.LinkUpdated{
				ShareID:           linkID("shareid"),
				Sharer:            userID("sharing-userid"),
				ItemID:            resourceID("provider-1", "storage-1", "itemid-1"),
				Permissions:       linkPermissions("stat"),
				MTime:             timestamp(10e8),
				DisplayName:       "link",
				Expiration:        timestamp(10e8 + 10e5),
				PasswordProtected: true,
				Token:             "token-123",
				FieldUpdated:      "permissions",
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventShareUpdated{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "sharing-userid", "2001-09-09T01:46:40Z", "user 'sharing-userid' updated field 'permissions' of public link 'shareid'", "share_permission_updated")
			// AuditEventSharing fields
			checkSharingAuditEvent(t, ev.AuditEventSharing, "itemid-1", "sharing-userid", "shareid")
			// AuditEventShareUpdated fields
			require.Equal(t, "", ev.ItemType) // not implemented atm
			require.Equal(t, "2001-09-20T15:33:20Z", ev.ExpirationDate)
			require.Equal(t, true, ev.SharePass)
			require.Equal(t, "stat:true", ev.Permissions)
			require.Equal(t, "link", ev.ShareType)
			require.Equal(t, "", ev.ShareWith) // not filled on links
			require.Equal(t, "sharing-userid", ev.ShareOwner)
			require.Equal(t, "token-123", ev.ShareToken)
		},
	}, {
		Alias: "ShareRemoved",
		SystemEvent: events.Event{
			Event: events.ShareRemoved{
				ShareID:  shareID("shareid"),
				ShareKey: nil,
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventShareRemoved{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "", "0001-01-01T00:00:00Z", "share id:'shareid' uid:'' item-id:'' was removed", "file_unshared")
			// AuditEventSharing fields
			checkSharingAuditEvent(t, ev.AuditEventSharing, "", "", "shareid")
			// AuditEventShareUpdated fields
			require.Equal(t, "", ev.ItemType) // not implemented atm
			require.Equal(t, "", ev.ShareType)
			require.Equal(t, "", ev.ShareWith) // not filled on links
		},
	}, {
		Alias: "LinkRemoved - id",
		SystemEvent: events.Event{
			Event: events.LinkRemoved{
				Executant:  userID("sharing-userid"),
				ShareID:    linkID("shareid"),
				ShareToken: "",
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventShareRemoved{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "sharing-userid", "", "user 'sharing-userid' removed public link with id:'shareid'", "file_unshared")
			// AuditEventSharing fields
			checkSharingAuditEvent(t, ev.AuditEventSharing, "", "sharing-userid", "shareid")
			// AuditEventShareUpdated fields
			require.Equal(t, "", ev.ItemType) // not implemented atm
			require.Equal(t, "link", ev.ShareType)
			require.Equal(t, "", ev.ShareWith) // not filled on links
		},
	}, {
		Alias: "LinkRemoved - token",
		SystemEvent: events.Event{
			Event: events.LinkRemoved{
				Executant:  userID("sharing-userid"),
				ShareID:    nil,
				ShareToken: "token-123",
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventShareRemoved{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "sharing-userid", "", "user 'sharing-userid' removed public link with id:'token-123'", "file_unshared")
			// AuditEventSharing fields
			checkSharingAuditEvent(t, ev.AuditEventSharing, "", "sharing-userid", "token-123")
			// AuditEventShareUpdated fields
			require.Equal(t, "", ev.ItemType) // not implemented atm
			require.Equal(t, "link", ev.ShareType)
			require.Equal(t, "", ev.ShareWith) // not filled on links
		},
	}, {
		Alias: "Share accepted",
		SystemEvent: events.Event{
			Event: events.ReceivedShareUpdated{
				ShareID:        shareID("shareid"),
				ItemID:         resourceID("provider-1", "storageid-1", "itemid-1"),
				Permissions:    sharePermissions("get_quota"),
				GranteeUserID:  userID("beshared-userid"),
				GranteeGroupID: nil,
				Sharer:         userID("sharing-userid"),
				MTime:          timestamp(10e8),
				State:          "SHARE_STATE_ACCEPTED",
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventReceivedShareUpdated{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "beshared-userid", "2001-09-09T01:46:40Z", "user 'beshared-userid' accepted share 'shareid' from user 'sharing-userid'", "share_accepted")
			// AuditEventSharing fields
			checkSharingAuditEvent(t, ev.AuditEventSharing, "itemid-1", "sharing-userid", "shareid")
			// AuditEventShareUpdated fields
			require.Equal(t, "", ev.ItemType)
			require.Equal(t, "user", ev.ShareType)
			require.Equal(t, "beshared-userid", ev.ShareWith)
		},
	}, {
		Alias: "Share declined",
		SystemEvent: events.Event{
			Event: events.ReceivedShareUpdated{
				ShareID:        shareID("shareid"),
				ItemID:         resourceID("provider-1", "storageid-1", "itemid-1"),
				Permissions:    sharePermissions("get_quota"),
				GranteeUserID:  userID("beshared-userid"),
				GranteeGroupID: nil,
				Sharer:         userID("sharing-userid"),
				MTime:          timestamp(10e8),
				State:          "SHARE_STATE_DECLINED",
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventReceivedShareUpdated{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "beshared-userid", "2001-09-09T01:46:40Z", "user 'beshared-userid' declined share 'shareid' from user 'sharing-userid'", "share_declined")
			// AuditEventSharing fields
			checkSharingAuditEvent(t, ev.AuditEventSharing, "itemid-1", "sharing-userid", "shareid")
			// AuditEventShareUpdated fields
			require.Equal(t, "", ev.ItemType)
			require.Equal(t, "user", ev.ShareType)
			require.Equal(t, "beshared-userid", ev.ShareWith)
		},
	}, {
		Alias: "Link accessed - success",
		SystemEvent: events.Event{
			Event: events.LinkAccessed{
				ShareID:           linkID("shareid"),
				Sharer:            userID("sharing-userid"),
				ItemID:            resourceID("provider-1", "storage-1", "itemid-1"),
				Permissions:       linkPermissions("stat"),
				DisplayName:       "link",
				Expiration:        timestamp(10e8 + 10e5),
				PasswordProtected: true,
				CTime:             timestamp(10e8),
				Token:             "token-123",
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventLinkAccessed{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "sharing-userid", "2001-09-09T01:46:40Z", "link with token 'token-123' was accessed. Success: true", "public_link_accessed")
			// AuditEventSharing fields
			checkSharingAuditEvent(t, ev.AuditEventSharing, "itemid-1", "sharing-userid", "shareid")
			// AuditEventShareUpdated fields
			require.Equal(t, "", ev.ItemType) // not implemented atm
			require.Equal(t, "token-123", ev.ShareToken)
			require.Equal(t, true, ev.Success)
		},
	}, {
		Alias: "Link accessed - failure",
		SystemEvent: events.Event{
			Event: events.LinkAccessFailed{
				ShareID: linkID("shareid"),
				Token:   "token-123",
				Status:  8,
				Message: "access denied",
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventLinkAccessed{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "", "", "link with token 'token-123' was accessed. Success: false", "public_link_accessed")
			// AuditEventSharing fields
			checkSharingAuditEvent(t, ev.AuditEventSharing, "", "", "shareid")
			// AuditEventShareUpdated fields
			require.Equal(t, "", ev.ItemType) // not implemented atm
			require.Equal(t, "token-123", ev.ShareToken)
			require.Equal(t, false, ev.Success)
		},
	}, {
		Alias: "File created",
		SystemEvent: events.Event{
			Event: events.FileUploaded{
				Executant: userID("uid-123"),
				Ref:       reference("pro-1", "sto-123", "iid-123", "./item"),
				Owner:     userID("uid-123"), // NOTE: owner not yet implemented in reva
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventFileCreated{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "uid-123", "", "user 'uid-123' created file 'pro-1$sto-123!iid-123/item'", "file_create")
			// AuditEventSharing fields
			checkFilesAuditEvent(t, ev.AuditEventFiles, "pro-1$sto-123!iid-123/item", "uid-123", "./item")
		},
	}, {
		Alias: "File read",
		SystemEvent: events.Event{
			Event: events.FileDownloaded{
				Executant: userID("uid-123"),
				Ref:       reference("pro-1", "sto-123", "iid-123", "./item"),
				Owner:     userID("uid-123"), // NOTE: owner not yet implemented in reva
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventFileRead{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "uid-123", "", "user 'uid-123' read file 'pro-1$sto-123!iid-123/item'", "file_read")
			// AuditEventSharing fields
			checkFilesAuditEvent(t, ev.AuditEventFiles, "pro-1$sto-123!iid-123/item", "uid-123", "./item")
		},
	}, {
		Alias: "File trashed",
		SystemEvent: events.Event{
			Event: events.ItemTrashed{
				Executant: userID("uid-123"),
				Ref:       reference("pro-1", "sto-123", "iid-123", "./item"),
				Owner:     userID("uid-123"), // NOTE: owner not yet implemented in reva
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventFileDeleted{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "uid-123", "", "user 'uid-123' trashed file 'pro-1$sto-123!iid-123/item'", "file_delete")
			// AuditEventSharing fields
			checkFilesAuditEvent(t, ev.AuditEventFiles, "pro-1$sto-123!iid-123/item", "uid-123", "./item")
		},
	}, {
		Alias: "File renamed",
		SystemEvent: events.Event{
			Event: events.ItemMoved{
				Executant:    userID("uid-123"),
				Ref:          reference("pro-1", "sto-123", "iid-123", "./item"),
				OldReference: reference("pro-1", "sto-123", "iid-123", "./anotheritem"),
				Owner:        userID("uid-123"), // NOTE: owner not yet implemented in reva
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventFileRenamed{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "uid-123", "", "user 'uid-123' moved file 'pro-1$sto-123!iid-123/item' from './anotheritem' to './item'", "file_rename")
			// AuditEventSharing fields
			checkFilesAuditEvent(t, ev.AuditEventFiles, "pro-1$sto-123!iid-123/item", "uid-123", "./item")
			// AuditEventFileRenamed fields
			require.Equal(t, "./anotheritem", ev.OldPath)

		},
	}, {
		Alias: "File purged",
		SystemEvent: events.Event{
			Event: events.ItemPurged{
				Executant: userID("uid-123"),
				Ref:       reference("pro-1", "sto-123", "iid-123", "./item"),
				Owner:     userID("uid-123"), // NOTE: owner not yet implemented in reva
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventFilePurged{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "uid-123", "", "user 'uid-123' removed file 'pro-1$sto-123!iid-123/item' from trashbin", "file_trash_delete")
			// AuditEventSharing fields
			checkFilesAuditEvent(t, ev.AuditEventFiles, "pro-1$sto-123!iid-123/item", "uid-123", "./item")
		},
	}, {
		Alias: "File restored",
		SystemEvent: events.Event{
			Event: events.ItemRestored{
				Executant:    userID("uid-123"),
				Ref:          reference("pro-1", "sto-123", "iid-123", "./item"),
				Owner:        userID("uid-123"), // NOTE: owner not yet implemented in reva
				OldReference: reference("pro-1", "sto-123", "sto-123!iid-123/item", "./oldpath"),
				Key:          "",
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventFileRestored{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "uid-123", "", "user 'uid-123' restored file 'pro-1$sto-123!iid-123/item' from trashbin to './item'", "file_trash_restore")
			// AuditEventSharing fields
			checkFilesAuditEvent(t, ev.AuditEventFiles, "pro-1$sto-123!iid-123/item", "uid-123", "./item")
			// AuditEventFileRestored fields
			require.Equal(t, "./oldpath", ev.OldPath)

		},
	}, {
		Alias: "File version restored",
		SystemEvent: events.Event{
			Event: events.FileVersionRestored{
				Executant: userID("uid-123"),
				Ref:       reference("pro-1", "sto-123", "iid-123", "./item"),
				Owner:     userID("uid-123"), // NOTE: owner not yet implemented in reva
				Key:       "v1",
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventFileVersionRestored{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "uid-123", "", "user 'uid-123' restored file 'pro-1$sto-123!iid-123/item' in version 'v1'", "file_version_restore")
			// AuditEventSharing fields
			checkFilesAuditEvent(t, ev.AuditEventFiles, "pro-1$sto-123!iid-123/item", "uid-123", "./item")
			// AuditEventFileRestored fields
			require.Equal(t, "v1", ev.Key)

		},
	}, {
		Alias: "Space created",
		SystemEvent: events.Event{
			Event: events.SpaceCreated{
				Executant: userID("uid-123"),
				ID:        &provider.StorageSpaceId{OpaqueId: "storage-1$space-123"},
				Owner:     userID("uid-123"),
				Root:      resourceID("pro-1", "sto-123", "iid-123"),
				Name:      "test-space",
				Type:      "project",
				Quota:     nil, // Quota not interesting atm
				MTime:     timestamp(10e9),
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventSpaceCreated{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "", "2286-11-20T17:46:40Z", "user 'uid-123' created a space 'space-123' with name 'test-space' (storage: 'storage-1')", "space_created")
			// AuditEventSpaces fields
			checkSpacesAuditEvent(t, ev.AuditEventSpaces, "storage-1$space-123")
			// AuditEventFileRestored fields
			require.Equal(t, "uid-123", ev.Owner)
			require.Equal(t, "pro-1$sto-123!iid-123", ev.RootItem)
			require.Equal(t, "test-space", ev.Name)
			require.Equal(t, "project", ev.Type)
		},
	}, {
		Alias: "Space renamed",
		SystemEvent: events.Event{
			Event: events.SpaceRenamed{
				Executant: userID("uid-123"),
				ID:        &provider.StorageSpaceId{OpaqueId: "storage-1$space-123"},
				Owner:     userID("uid-123"),
				Name:      "new-name",
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventSpaceRenamed{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "", "", "user 'uid-123' renamed space 'space-123' to 'new-name' (storage: 'storage-1')", "space_renamed")
			// AuditEventSpaces fields
			checkSpacesAuditEvent(t, ev.AuditEventSpaces, "storage-1$space-123")
			// AuditEventSpaceRenamed fields
			require.Equal(t, "new-name", ev.NewName)
		},
	}, {
		Alias: "Space disabled",
		SystemEvent: events.Event{
			Event: events.SpaceDisabled{
				Executant: userID("uid-123"),
				ID:        &provider.StorageSpaceId{OpaqueId: "storage-1$space-123"},
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventSpaceDisabled{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "", "0001-01-01T00:00:00Z", "user 'uid-123' disabled the space 'space-123' (storage: 'storage-1')", "space_disabled")
			// AuditEventSpaces fields
			checkSpacesAuditEvent(t, ev.AuditEventSpaces, "storage-1$space-123")
		},
	}, {
		Alias: "Space enabled",
		SystemEvent: events.Event{
			Event: events.SpaceEnabled{
				Executant: userID("uid-123"),
				ID:        &provider.StorageSpaceId{OpaqueId: "storage-1$space-123"},
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventSpaceEnabled{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "", "", "user 'uid-123' (re-) enabled the space 'space-123' (storage: 'storage-1')", "space_enabled")
			// AuditEventSpaces fields
			checkSpacesAuditEvent(t, ev.AuditEventSpaces, "storage-1$space-123")
		},
	}, {
		Alias: "Space deleted",
		SystemEvent: events.Event{
			Event: events.SpaceDeleted{
				Executant: userID("uid-123"),
				ID:        &provider.StorageSpaceId{OpaqueId: "storage-1$space-123"},
				SpaceName: "test-space",
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventSpaceDeleted{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "", "0001-01-01T00:00:00Z", "user 'uid-123' deleted the space 'space-123' with name 'test-space' (storage: 'storage-1')", "space_deleted")
			// AuditEventSpaces fields
			checkSpacesAuditEvent(t, ev.AuditEventSpaces, "storage-1$space-123")
		},
	}, {
		Alias: "Space deleted - unicode name",
		SystemEvent: events.Event{
			Event: events.SpaceDeleted{
				Executant: userID("uid-123"),
				ID:        &provider.StorageSpaceId{OpaqueId: "storage-1$space-123"},
				SpaceName: "Prüfung 📁 空间",
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventSpaceDeleted{}
			require.NoError(t, json.Unmarshal(b, &ev))

			checkBaseAuditEvent(t, ev.AuditEvent, "", "0001-01-01T00:00:00Z", "user 'uid-123' deleted the space 'space-123' with name 'Prüfung 📁 空间' (storage: 'storage-1')", "space_deleted")
			checkSpacesAuditEvent(t, ev.AuditEventSpaces, "storage-1$space-123")
		},
	}, {
		Alias: "Space deleted - empty name",
		SystemEvent: events.Event{
			Event: events.SpaceDeleted{
				Executant: userID("uid-123"),
				ID:        &provider.StorageSpaceId{OpaqueId: "storage-1$space-123"},
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventSpaceDeleted{}
			require.NoError(t, json.Unmarshal(b, &ev))

			checkBaseAuditEvent(t, ev.AuditEvent, "", "0001-01-01T00:00:00Z", "user 'uid-123' deleted the space 'space-123' with name '' (storage: 'storage-1')", "space_deleted")
			checkSpacesAuditEvent(t, ev.AuditEventSpaces, "storage-1$space-123")
		},
	},
	{
		Alias: "ScienceMesh - InviteTokenGenerated",
		SystemEvent: events.Event{
			Event: events.ScienceMeshInviteTokenGenerated{
				Sharer:        userID("sharer-user-id"),
				RecipientMail: "mail@opencloud.test",
				Token:         "token-123",
				Description:   "some-description",
				Expiration:    uint64(10e8),
				InviteLink:    "http://opencloud.test/invite",
				Timestamp:     timestamp(10e8),
			},
		},
		CheckAuditEvent: func(t *testing.T, b []byte) {
			ev := types.AuditEventScienceMeshInviteTokenGenerated{}
			require.NoError(t, json.Unmarshal(b, &ev))

			// AuditEvent fields
			checkBaseAuditEvent(t, ev.AuditEvent, "sharer-user-id", "2001-09-09T01:46:40Z", "user 'sharer-user-id' generated a ScienceMesh invite with token 'token-123'", "science_mesh_invite_token_generated")
			// AuditEventScienceMeshInviteTokenGenerated fields
			require.Equal(t, "mail@opencloud.test", ev.RecipientMail)
			require.Equal(t, "token-123", ev.Token)
			require.Equal(t, "some-description", ev.Description)
			require.Equal(t, uint64(10e8), ev.Expiration)
			require.Equal(t, "http://opencloud.test/invite", ev.InviteLink)
		},
	},
}

func TestAuditLogging(t *testing.T) {
	l := log.NewLogger()

	inch := make(chan events.Event)
	defer close(inch)

	outch := make(chan []byte)
	defer close(outch)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	go StartAuditLogger(ctx, inch, l, Marshal("json", l), func(b []byte) {
		outch <- b
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Alias, func(t *testing.T) {
			inch <- tc.SystemEvent
			tc.CheckAuditEvent(t, <-outch)
		})
	}
}

func checkBaseAuditEvent(t *testing.T, ev types.AuditEvent, user string, time string, message string, action string) {
	require.Equal(t, "", ev.RemoteAddr) // not implemented atm
	require.Equal(t, user, ev.User)
	require.Equal(t, "", ev.URL)       // not implemented atm
	require.Equal(t, "", ev.Method)    // not implemented atm
	require.Equal(t, "", ev.UserAgent) // not implemented atm
	require.Equal(t, time, ev.Time)
	require.Equal(t, "admin_audit", ev.App)
	require.Equal(t, message, ev.Message)
	require.Equal(t, action, ev.Action)
	require.Equal(t, false, ev.CLI) // not implemented atm
	require.Equal(t, 1, ev.Level)
}

func checkSharingAuditEvent(t *testing.T, ev types.AuditEventSharing, itemID string, owner string, shareID string) {
	require.Equal(t, itemID, ev.FileID)
	require.Equal(t, owner, ev.Owner)
	require.Equal(t, "", ev.Path) // not implemented atm
	require.Equal(t, shareID, ev.ShareID)
}

func checkFilesAuditEvent(t *testing.T, ev types.AuditEventFiles, itemID string, owner string, path string) {
	require.Equal(t, itemID, ev.FileID)
	require.Equal(t, owner, ev.Owner)
	require.Equal(t, path, ev.Path)
}

func checkSpacesAuditEvent(t *testing.T, ev types.AuditEventSpaces, spaceID string) {
	require.Equal(t, spaceID, ev.SpaceID)
}
func shareID(id string) *collaboration.ShareId {
	return &collaboration.ShareId{
		OpaqueId: id,
	}
}

func linkID(id string) *link.PublicShareId {
	return &link.PublicShareId{
		OpaqueId: id,
	}
}

func userID(id string) *user.UserId {
	return &user.UserId{
		OpaqueId: id,
		Idp:      "idp",
	}
}

func groupID(id string) *group.GroupId {
	return &group.GroupId{
		OpaqueId: id,
		Idp:      "idp",
	}
}

func resourceID(sid, spid, oid string) *provider.ResourceId {
	return &provider.ResourceId{
		StorageId: sid,
		SpaceId:   spid,
		OpaqueId:  oid,
	}
}

func reference(sid, spid, oid, path string) *provider.Reference {
	return &provider.Reference{
		ResourceId: resourceID(sid, spid, oid),
		Path:       path,
	}
}

func timestamp(seconds uint64) *rtypes.Timestamp {
	return &rtypes.Timestamp{
		Seconds: seconds,
		Nanos:   0,
	}
}

func sharePermissions(perms ...string) *collaboration.SharePermissions {
	return &collaboration.SharePermissions{
		Permissions: permissions(perms...),
	}
}

func linkPermissions(perms ...string) *link.PublicSharePermissions {
	return &link.PublicSharePermissions{
		Permissions: permissions(perms...),
	}
}

func permissions(permissions ...string) *provider.ResourcePermissions {
	perms := &provider.ResourcePermissions{}

	for _, p := range permissions {
		switch p {
		case "stat":
			perms.Stat = true
		case "get_path":
			perms.GetPath = true
		case "list_container":
			perms.ListContainer = true
		case "get_quota":
			perms.GetQuota = true

		}
	}

	return perms
}
