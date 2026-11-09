// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocEvents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
)

// handleShareCreated handles a share created event.
func (s *EventConsumer) handleShareCreated(ctx context.Context, ev events.ShareCreated) error {
	_, span := tracer.Start(ctx, "handleShareCreated")
	defer span.End()

	if ev.GranteeUserID == nil || ev.GranteeUserID.GetType() != user.UserType_USER_TYPE_GUEST {
		s.log.Debug().Msg("share created event is not for a guest, skipping")
		return nil
	}

	tok, err := s.authGuest.CreateToken(ctx, ev.ShareID.GetOpaqueId())
	if err != nil {
		return err
	}

	return events.Publish(ctx, s.stream, ocEvents.GuestTokenCreated{
		ShareID:      ev.ShareID,
		Sharer:       ev.Sharer,
		GranteeEmail: ev.GranteeUserID.GetOpaqueId(),
		ItemID:       ev.ItemID,
		ResourceName: ev.ResourceName,
		Token:        tok.String(),
		Timestamp:    time.Now(),
	})
}

// handleShareRemoved handles a share removed event.
func (s *EventConsumer) handleShareRemoved(ctx context.Context, ev events.ShareRemoved) error {
	_, span := tracer.Start(ctx, "handleShareRemoved")
	defer span.End()

	if ev.GranteeUserID == nil || ev.GranteeUserID.GetType() != user.UserType_USER_TYPE_GUEST {
		s.log.Debug().Msg("share removed event is not for a guest, skipping")
		return nil
	}

	s.log.Debug().Interface("event", ev).Msg("share removed event received")

	return s.authGuest.CleanupShare(ev.ShareID.GetOpaqueId())
}

// handleShareExpired handles a share expired event.
func (s *EventConsumer) handleShareExpired(ctx context.Context, ev events.ShareExpired) error {
	_, span := tracer.Start(ctx, "handleShareExpired")
	defer span.End()

	if ev.GranteeUserID == nil || ev.GranteeUserID.GetType() != user.UserType_USER_TYPE_GUEST {
		s.log.Debug().Msg("share expired event is not for a guest, skipping")
		return nil
	}

	s.log.Debug().Interface("event", ev).Msg("share expired event received")

	return s.authGuest.CleanupShare(ev.ShareID.GetOpaqueId())
}

// handleSpaceShared handles a space shared event.
func (s *EventConsumer) handleSpaceShared(ctx context.Context, ev events.SpaceShared) error {
	_, span := tracer.Start(ctx, "handleSpaceShared")
	defer span.End()

	if ev.GranteeUserID == nil || ev.GranteeUserID.GetType() != user.UserType_USER_TYPE_GUEST {
		s.log.Debug().Msg("space shared event is not for a guest, skipping")
		return nil
	}

	if ev.ID.GetOpaqueId() == "" {
		s.log.Warn().Msg("space shared event without space id, skipping")
		return nil
	}

	if ev.ShareID.GetOpaqueId() == "" {
		s.log.Warn().Msg("space shared event without share id, skipping")
		return nil
	}

	// The space root node id is the space id.
	itemID, err := storagespace.ParseID(ev.ID.GetOpaqueId())
	if err != nil {
		return err
	}
	itemID.OpaqueId = itemID.SpaceId

	tok, err := s.authGuest.CreateToken(ctx, ev.ShareID.GetOpaqueId())
	if err != nil {
		return err
	}

	return events.Publish(ctx, s.stream, ocEvents.GuestTokenCreated{
		ShareID:      ev.ShareID,
		Sharer:       ev.Creator,
		GranteeEmail: ev.GranteeUserID.GetOpaqueId(),
		ItemID:       &itemID,
		Token:        tok.String(),
		Timestamp:    time.Now(),
	})
}

// handleSpaceUnshared handles a space unshared event.
func (s *EventConsumer) handleSpaceUnshared(ctx context.Context, ev events.SpaceUnshared) error {
	_, span := tracer.Start(ctx, "handleSpaceUnshared")
	defer span.End()

	if ev.GranteeUserID == nil || ev.GranteeUserID.GetType() != user.UserType_USER_TYPE_GUEST {
		s.log.Debug().Msg("space unshared event is not for a guest, skipping")
		return nil
	}

	if ev.ShareID.GetOpaqueId() == "" {
		s.log.Warn().Msg("space unshared event without share id, skipping")
		return nil
	}

	s.log.Debug().Interface("event", ev).Msg("space unshared event received")

	return s.authGuest.CleanupShare(ev.ShareID.GetOpaqueId())
}

// handleSpaceMembershipExpired handles a space membership expired event.
func (s *EventConsumer) handleSpaceMembershipExpired(ctx context.Context, ev events.SpaceMembershipExpired) error {
	_, span := tracer.Start(ctx, "handleSpaceMembershipExpired")
	defer span.End()

	if ev.GranteeUserID == nil || ev.GranteeUserID.GetType() != user.UserType_USER_TYPE_GUEST {
		s.log.Debug().Msg("space membership expired event is not for a guest, skipping")
		return nil
	}

	if ev.ShareID.GetOpaqueId() == "" {
		s.log.Warn().Msg("space membership expired event without share id, skipping")
		return nil
	}

	s.log.Debug().Interface("event", ev).Msg("space membership expired event received")

	return s.authGuest.CleanupShare(ev.ShareID.GetOpaqueId())
}
