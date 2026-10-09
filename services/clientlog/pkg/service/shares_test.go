// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"errors"

	group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

var _ = Describe("share recipients", func() {
	const (
		storageID = "storage"
		spaceID   = "space"
	)

	var (
		ctx    context.Context
		gwc    *cs3mocks.GatewayAPIClient
		logger log.Logger

		rootID   = &provider.ResourceId{StorageId: storageID, SpaceId: spaceID, OpaqueId: spaceID}
		folderA  = &provider.ResourceId{StorageId: storageID, SpaceId: spaceID, OpaqueId: "a"}
		fileID   = &provider.ResourceId{StorageId: storageID, SpaceId: spaceID, OpaqueId: "file"}
		viewer   = &provider.ResourcePermissions{Stat: true, ListContainer: true}
		denied   = &provider.ResourcePermissions{}
		okStatus = &rpc.Status{Code: rpc.Code_CODE_OK}
	)

	userGrant := func(userID *user.UserId, perms *provider.ResourcePermissions) *provider.Grant {
		return &provider.Grant{
			Grantee:     &provider.Grantee{Type: provider.GranteeType_GRANTEE_TYPE_USER, Id: &provider.Grantee_UserId{UserId: userID}},
			Permissions: perms,
		}
	}

	groupGrant := func(groupID string, perms *provider.ResourcePermissions) *provider.Grant {
		return &provider.Grant{
			Grantee:     &provider.Grantee{Type: provider.GranteeType_GRANTEE_TYPE_GROUP, Id: &provider.Grantee_GroupId{GroupId: &group.GroupId{OpaqueId: groupID}}},
			Permissions: perms,
		}
	}

	// resourceInfo returns a resource info as stat'ed with the ancestor grants
	resourceInfo := func(id *provider.ResourceId, grants ...*provider.Grant) *provider.ResourceInfo {
		return &provider.ResourceInfo{
			Id:     id,
			Opaque: utils.AppendGrantsToOpaque(nil, ancestorGrantsKey, grants),
		}
	}

	expectGroupMembers := func(members ...string) {
		ids := make([]*user.UserId, 0, len(members))
		for _, m := range members {
			ids = append(ids, &user.UserId{OpaqueId: m})
		}
		gwc.EXPECT().GetGroup(mock.Anything, mock.Anything).Return(&group.GetGroupResponse{
			Status: okStatus,
			Group:  &group.Group{Members: ids},
		}, nil)
	}

	BeforeEach(func() {
		ctx = context.Background()
		gwc = cs3mocks.NewGatewayAPIClient(GinkgoT())
		logger = log.NopLogger()
	})

	Describe("getShareRecipients", func() {
		It("returns the grantees of the ancestor grants", func() {
			expectGroupMembers("bob", "carol")

			recipients, denials, err := getShareRecipients(ctx, gwc, resourceInfo(fileID,
				userGrant(&user.UserId{OpaqueId: "bob"}, viewer),
				userGrant(&user.UserId{OpaqueId: "guest@example.com", Type: user.UserType_USER_TYPE_GUEST}, viewer),
				groupGrant("group", viewer),
			))
			Expect(err).ToNot(HaveOccurred())
			Expect(recipients).To(ConsistOf("bob", "guest@example.com", "carol"))
			Expect(denials).To(BeEmpty())
		})

		It("returns the users that were denied access", func() {
			expectGroupMembers("carol", "dave")

			recipients, denials, err := getShareRecipients(ctx, gwc, resourceInfo(fileID,
				userGrant(&user.UserId{OpaqueId: "bob"}, viewer),
				groupGrant("group", denied),
			))
			Expect(err).ToNot(HaveOccurred())
			Expect(recipients).To(ConsistOf("bob"))
			Expect(denials).To(ConsistOf("carol", "dave"))
		})

		It("returns nothing when the storage didn't return the ancestor grants", func() {
			recipients, denials, err := getShareRecipients(ctx, gwc, &provider.ResourceInfo{Id: fileID})
			Expect(err).ToNot(HaveOccurred())
			Expect(recipients).To(BeEmpty())
			Expect(denials).To(BeEmpty())
		})

		It("fails when the ancestor grants can't be decoded", func() {
			info := &provider.ResourceInfo{Id: fileID, Opaque: &types.Opaque{Map: map[string]*types.OpaqueEntry{
				ancestorGrantsKey: {Decoder: "json", Value: []byte("not json")},
			}}}
			_, _, err := getShareRecipients(ctx, gwc, info)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("addShareRecipients", func() {
		It("adds the share recipients and removes denied users", func() {
			users := addShareRecipients(ctx, gwc, logger, []string{"alice", "carol"}, resourceInfo(fileID,
				userGrant(&user.UserId{OpaqueId: "bob"}, viewer),
				userGrant(&user.UserId{OpaqueId: "alice"}, viewer),
				userGrant(&user.UserId{OpaqueId: "carol"}, denied),
			))
			Expect(users).To(ConsistOf("alice", "bob"))
		})

		It("keeps the users when a group can't be resolved", func() {
			gwc.EXPECT().GetGroup(mock.Anything, mock.Anything).Return(nil, errors.New("unavailable"))

			users := addShareRecipients(ctx, gwc, logger, []string{"alice"}, resourceInfo(fileID, groupGrant("group", denied)))
			Expect(users).To(ConsistOf("alice"))
		})
	})

	Describe("addParentShareRecipients", func() {
		It("stats the parent with its ancestor grants", func() {
			gwc.EXPECT().Stat(mock.Anything, mock.MatchedBy(func(req *provider.StatRequest) bool {
				return utils.ResourceIDEqual(req.GetRef().GetResourceId(), rootID) &&
					req.GetRef().GetPath() == "./a" &&
					req.GetFieldMask().GetPaths()[0] == ancestorGrantsKey
			})).Return(&provider.StatResponse{
				Status: okStatus,
				Info:   resourceInfo(folderA, userGrant(&user.UserId{OpaqueId: "bob"}, viewer)),
			}, nil)

			users := addParentShareRecipients(ctx, gwc, logger, []string{"alice"}, &provider.Reference{ResourceId: rootID, Path: "./a/file"})
			Expect(users).To(ConsistOf("alice", "bob"))
		})

		It("keeps the users when the parent can't be stat'ed", func() {
			gwc.EXPECT().Stat(mock.Anything, mock.Anything).Return(&provider.StatResponse{
				Status: &rpc.Status{Code: rpc.Code_CODE_NOT_FOUND},
			}, nil)

			users := addParentShareRecipients(ctx, gwc, logger, []string{"alice"}, &provider.Reference{ResourceId: rootID, Path: "./a/file"})
			Expect(users).To(ConsistOf("alice"))
		})

		It("does nothing when the parent can't be derived from the reference", func() {
			users := addParentShareRecipients(ctx, gwc, logger, []string{"alice"}, &provider.Reference{ResourceId: fileID})
			Expect(users).To(ConsistOf("alice"))
		})
	})

	Describe("processFileEvent", func() {
		It("notifies the space members and the share recipients with a single stat", func() {
			info := resourceInfo(fileID,
				userGrant(&user.UserId{OpaqueId: "bob"}, viewer),
				userGrant(&user.UserId{OpaqueId: "carol"}, denied),
			)
			info.ParentId = folderA
			info.Space = &provider.StorageSpace{Id: &provider.StorageSpaceId{OpaqueId: storageID + "$" + spaceID}}
			gwc.EXPECT().Stat(mock.Anything, mock.MatchedBy(func(req *provider.StatRequest) bool {
				return req.GetFieldMask().GetPaths()[1] == ancestorGrantsKey
			})).Return(&provider.StatResponse{Status: okStatus, Info: info}, nil).Once()

			members := utils.AppendJSONToOpaque(nil, "grants", map[string]*provider.ResourcePermissions{
				"alice": viewer,
				"carol": viewer,
			})
			gwc.EXPECT().ListStorageSpaces(mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
				Status:        okStatus,
				StorageSpaces: []*provider.StorageSpace{{Opaque: members, SpaceType: "project"}},
			}, nil)

			users, data, err := processFileEvent(ctx, &provider.Reference{ResourceId: fileID}, gwc, logger, "initiator")
			Expect(err).ToNot(HaveOccurred())
			Expect(users).To(ConsistOf("alice", "bob"))
			Expect(data.ItemID).To(Equal("storage$space!file"))
		})
	})

	DescribeTable("parentReference",
		func(ref *provider.Reference, expectedPath string, expectedOK bool) {
			parent, ok := parentReference(ref)
			Expect(ok).To(Equal(expectedOK))
			if ok {
				Expect(parent.GetResourceId()).To(Equal(ref.GetResourceId()))
				Expect(parent.GetPath()).To(Equal(expectedPath))
			}
		},
		Entry("item in the referenced folder", &provider.Reference{ResourceId: rootID, Path: "./file"}, ".", true),
		Entry("nested item", &provider.Reference{ResourceId: rootID, Path: "./a/b/file"}, "./a/b", true),
		Entry("trailing slash", &provider.Reference{ResourceId: rootID, Path: "./a/b/"}, "./a", true),
		Entry("reference by id", &provider.Reference{ResourceId: fileID}, "", false),
		Entry("reference to the resource itself", &provider.Reference{ResourceId: fileID, Path: "."}, "", false),
		Entry("reference without resource id", &provider.Reference{Path: "/a/file"}, "", false),
	)
})
