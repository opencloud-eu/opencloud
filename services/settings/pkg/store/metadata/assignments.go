// Package store implements the go-micro store interface
package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/settings"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/store/defaults"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
)

// ListRoleAssignments loads and returns all role assignments matching the given assignment identifier.
func (s *Store) ListRoleAssignments(accountUUID string) ([]*settingsmsg.UserRoleAssignment, error) {
	// shortcut for service accounts
	for _, serviceAccountID := range s.cfg.ServiceAccountIDs {
		if accountUUID == serviceAccountID {
			return []*settingsmsg.UserRoleAssignment{
				{
					Id:          uuid.NewString(), // should we hardcode this id too?
					AccountUuid: accountUUID,
					RoleId:      defaults.BundleUUIDServiceAccount,
				},
			}, nil
		}
	}
	s.Init()
	ctx := context.TODO()
	assIDs, err := s.mdc.ReadDir(ctx, accountPath(accountUUID))
	switch err.(type) {
	case nil:
		// continue
	case errtypes.NotFound:
		return make([]*settingsmsg.UserRoleAssignment, 0), nil
	default:
		return nil, err
	}

	ass := make([]*settingsmsg.UserRoleAssignment, 0, len(assIDs))
	for _, assID := range assIDs {
		b, err := s.mdc.SimpleDownload(ctx, assignmentPath(accountUUID, assID))
		switch err.(type) {
		case nil:
			// continue
		case errtypes.NotFound:
			continue
		default:
			return nil, err
		}

		a := &settingsmsg.UserRoleAssignment{}
		err = json.Unmarshal(b, a)
		if err != nil {
			return nil, err
		}

		ass = append(ass, a)
	}
	return ass, nil
}

// ListRoleAssignmentsByRole returns all role assignmentes matching the give roleID
func (s *Store) ListRoleAssignmentsByRole(roleID string) ([]*settingsmsg.UserRoleAssignment, error) {
	s.Init()
	ctx := context.TODO()
	accountIDs, err := s.mdc.ReadDir(ctx, accountsFolderLocation)
	switch err.(type) {
	case nil:
		// continue
	case errtypes.NotFound:
		return make([]*settingsmsg.UserRoleAssignment, 0), nil
	default:
		return nil, err
	}
	assignments := make([]*settingsmsg.UserRoleAssignment, 0, len(accountIDs))

	// This is very inefficient, with the current layout we need to iterated through all
	// account folders and read each assignment file in there to check if that contains
	// the give role ID.
	for _, account := range accountIDs {
		assignmentIDs, err := s.mdc.ReadDir(ctx, accountPath(account))
		switch err.(type) {
		case nil:
			// continue
		case errtypes.NotFound:
			return make([]*settingsmsg.UserRoleAssignment, 0), nil
		default:
			return nil, err
		}

		for _, assignmentID := range assignmentIDs {
			b, err := s.mdc.SimpleDownload(ctx, assignmentPath(account, assignmentID))
			switch err.(type) {
			case nil:
				// continue
			case errtypes.NotFound:
				continue
			default:
				return nil, err
			}

			a := &settingsmsg.UserRoleAssignment{}
			err = json.Unmarshal(b, a)
			if err != nil {
				return nil, err
			}
			if a.GetRoleId() == roleID {
				assignments = append(assignments, a)
			}
		}
	}
	return assignments, nil
}

// WriteRoleAssignment appends the given role assignment to the existing assignments of the respective account.
func (s *Store) WriteRoleAssignment(accountUUID, roleID string) (*settingsmsg.UserRoleAssignment, error) {
	s.Init()
	ctx := context.TODO()
	// as per https://github.com/owncloud/product/issues/103 "Each user can have exactly one role"
	err := s.mdc.Delete(ctx, accountPath(accountUUID))
	switch err.(type) {
	case nil:
		// continue
	case errtypes.NotFound:
		// already gone, continue
	default:
		return nil, err
	}

	err = s.mdc.MakeDirIfNotExist(ctx, accountPath(accountUUID))
	if err != nil {
		return nil, err
	}

	ass := &settingsmsg.UserRoleAssignment{
		Id:          uuid.NewString(),
		AccountUuid: accountUUID,
		RoleId:      roleID,
	}
	b, err := json.Marshal(ass)
	if err != nil {
		return nil, err
	}
	return ass, s.mdc.SimpleUpload(ctx, assignmentPath(accountUUID, ass.Id), b)
}

// RemoveRoleAssignment deletes the given role assignment from the existing assignments of the respective account.
func (s *Store) RemoveRoleAssignment(assignmentID string) error {
	s.Init()
	ctx := context.TODO()
	accounts, err := s.mdc.ReadDir(ctx, accountsFolderLocation)
	switch err.(type) {
	case nil:
		// continue
	case errtypes.NotFound:
		return fmt.Errorf("assignmentID '%s' %w", assignmentID, settings.ErrNotFound)
	default:
		return err
	}

	// TODO: use indexer to avoid spamming Metadata service
	for _, accID := range accounts {
		assIDs, err := s.mdc.ReadDir(ctx, accountPath(accID))
		if err != nil {
			// TODO: error?
			continue
		}

		for _, assID := range assIDs {
			if assID == assignmentID {
				// as per https://github.com/owncloud/product/issues/103 "Each user can have exactly one role"
				// we also have to delete the cached dir listing
				return s.mdc.Delete(ctx, accountPath(accID))
			}
		}
	}
	return fmt.Errorf("assignmentID '%s' %w", assignmentID, settings.ErrNotFound)
}

func accountPath(accountUUID string) string {
	return fmt.Sprintf("%s/%s", accountsFolderLocation, accountUUID)
}

func assignmentPath(accountUUID string, assignmentID string) string {
	return fmt.Sprintf("%s/%s/%s", accountsFolderLocation, accountUUID, assignmentID)
}
