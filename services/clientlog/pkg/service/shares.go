// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"errors"
	"path"
	"slices"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/grants"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

// ancestorGrantsKey makes a stat return the active grants on the resource and its ancestors,
// excluding the space root. The storage only returns them to service accounts.
const ancestorGrantsKey = "ancestor-grants"

var (
	// resourceFieldMask returns the complete resource info including the ancestor grants
	resourceFieldMask = &fieldmaskpb.FieldMask{Paths: []string{"*", ancestorGrantsKey}}
	// ancestorGrantsFieldMask returns the basic resource info and the ancestor grants
	ancestorGrantsFieldMask = &fieldmaskpb.FieldMask{Paths: []string{ancestorGrantsKey}}
)

// addShareRecipients adds the grantees of shares on the resource or on one of its ancestors
// to users and removes the users that were denied access. The resource info must have been
// stat'ed with the ancestor grants. Errors are logged, users is returned unchanged in that case.
func addShareRecipients(ctx context.Context, gwc gateway.GatewayAPIClient, logger log.Logger, users []string, info *provider.ResourceInfo) []string {
	recipients, denied, err := getShareRecipients(ctx, gwc, info)
	if err != nil {
		logger.Error().Err(err).Str("itemid", storagespace.FormatResourceID(info.GetId())).Msg("error gathering share recipients")
		return users
	}

	for _, u := range recipients {
		users = appendUnique(users, u)
	}
	return slices.DeleteFunc(users, func(u string) bool {
		return slices.Contains(denied, u)
	})
}

// addParentShareRecipients is like addShareRecipients but starts at the
// parent of ref. It is used for resources that can no longer be stat'ed at
// ref, e.g. because they were trashed or moved away.
func addParentShareRecipients(ctx context.Context, gwc gateway.GatewayAPIClient, logger log.Logger, users []string, ref *provider.Reference) []string {
	parentRef, ok := parentReference(ref)
	if !ok {
		return users
	}

	info, err := stat(ctx, gwc, parentRef, ancestorGrantsFieldMask)
	if err != nil {
		logger.Error().Err(err).Interface("ref", parentRef).Msg("error getting parent for share recipients")
		return users
	}
	return addShareRecipients(ctx, gwc, logger, users, info)
}

// getShareRecipients returns the grantees of the ancestor grants of the resource and the users
// that were denied access on the resource or one of its ancestors.
func getShareRecipients(ctx context.Context, gwc gateway.GatewayAPIClient, info *provider.ResourceInfo) (recipients, denied []string, err error) {
	if !utils.ExistsInOpaque(info.GetOpaque(), ancestorGrantsKey) {
		return nil, nil, nil
	}

	gs, err := utils.ReadGrantsFromOpaque(info.GetOpaque(), ancestorGrantsKey)
	if err != nil {
		return nil, nil, err
	}

	for _, g := range gs {
		uid, gid := g.GetGrantee().GetUserId(), g.GetGrantee().GetGroupId()
		if uid == nil && gid == nil {
			continue
		}

		us, err := resolveID(ctx, gwc, uid, gid)
		if err != nil {
			return nil, nil, err
		}

		// denials are grants without any permissions, they apply to the whole subtree
		if grants.PermissionsEqual(g.GetPermissions(), &provider.ResourcePermissions{}) {
			denied = append(denied, us...)
			continue
		}
		for _, u := range us {
			recipients = appendUnique(recipients, u)
		}
	}
	return recipients, denied, nil
}

// stat stats the resource, only returning the fields of the field mask
func stat(ctx context.Context, gwc gateway.GatewayAPIClient, ref *provider.Reference, fieldMask *fieldmaskpb.FieldMask) (*provider.ResourceInfo, error) {
	res, err := gwc.Stat(ctx, &provider.StatRequest{Ref: ref, FieldMask: fieldMask})
	if err != nil {
		return nil, err
	}
	if res.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return nil, errors.New("error stating resource: " + res.GetStatus().GetMessage())
	}
	return res.GetInfo(), nil
}

// parentReference returns a reference to the parent of a relative reference.
// It returns false if ref does not carry a path below its resource id, in
// which case the parent can't be derived from it.
func parentReference(ref *provider.Reference) (*provider.Reference, bool) {
	p := path.Clean(ref.GetPath())
	if ref.GetResourceId() == nil || p == "." || p == "/" {
		return nil, false
	}
	return &provider.Reference{
		ResourceId: ref.GetResourceId(),
		Path:       utils.MakeRelativePath(path.Dir(p)),
	}, true
}
