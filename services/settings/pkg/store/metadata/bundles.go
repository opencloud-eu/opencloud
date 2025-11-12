// Package store implements the go-micro store interface
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/settings"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/store/defaults"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
)

// ListBundles returns all bundles in the dataPath folder that match the given type.
func (s *Store) ListBundles(bundleType settingsmsg.Bundle_Type, bundleIDs []string) ([]*settingsmsg.Bundle, error) {
	s.Init()
	ctx := context.TODO()

	if len(bundleIDs) == 0 {
		bIDs, err := s.mdc.ReadDir(ctx, bundleFolderLocation)
		switch err.(type) {
		case nil:
			// continue
		case errtypes.NotFound:
			return make([]*settingsmsg.Bundle, 0), nil
		default:
			return nil, err
		}

		bundleIDs = bIDs
	}
	var bundles []*settingsmsg.Bundle
	for _, id := range bundleIDs {
		if id == defaults.BundleUUIDServiceAccount {
			bundles = append(bundles, defaults.ServiceAccountBundle())
			continue
		}
		b, err := s.mdc.SimpleDownload(ctx, bundlePath(id))
		switch err.(type) {
		case nil:
			// continue
		case errtypes.NotFound:
			continue
		default:
			return nil, err
		}

		bundle := &settingsmsg.Bundle{}
		err = json.Unmarshal(b, bundle)
		if err != nil {
			return nil, err
		}

		if bundle.Type == bundleType {
			bundles = append(bundles, bundle)
		}

	}
	return bundles, nil
}

// ReadBundle tries to find a bundle by the given id from the metadata service
func (s *Store) ReadBundle(bundleID string) (*settingsmsg.Bundle, error) {
	// shortcut for service accounts
	if bundleID == defaults.BundleUUIDServiceAccount {
		return defaults.ServiceAccountBundle(), nil
	}

	s.Init()
	ctx := context.TODO()
	b, err := s.mdc.SimpleDownload(ctx, bundlePath(bundleID))
	switch err.(type) {
	case nil:
		// continue
	case errtypes.NotFound:
		return nil, fmt.Errorf("bundleID '%s' %w", bundleID, settings.ErrNotFound)
	default:
		return nil, err
	}

	bundle := &settingsmsg.Bundle{}
	return bundle, json.Unmarshal(b, bundle)
}

// ReadSetting tries to find a setting by the given id from the metadata service
func (s *Store) ReadSetting(settingID string) (*settingsmsg.Setting, error) {
	s.Init()
	ctx := context.TODO()

	ids, err := s.mdc.ReadDir(ctx, bundleFolderLocation)
	switch err.(type) {
	case nil:
		// continue
	case errtypes.NotFound:
		return nil, fmt.Errorf("settingID '%s' %w", settingID, settings.ErrNotFound)
	default:
		return nil, err
	}

	// TODO: avoid spamming metadata service
	for _, id := range ids {
		b, err := s.ReadBundle(id)
		if err != nil {
			if errors.Is(err, settings.ErrNotFound) {
				continue
			}
			return nil, err
		}

		for _, setting := range b.Settings {
			if setting.Id == settingID {
				return setting, nil
			}
		}

	}
	return nil, fmt.Errorf("settingID '%s' %w", settingID, settings.ErrNotFound)
}

// WriteBundle sends the givens record to the metadataclient. returns `record` for legacy reasons
func (s *Store) WriteBundle(record *settingsmsg.Bundle) (*settingsmsg.Bundle, error) {
	s.Init()
	ctx := context.TODO()

	b, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	return record, s.mdc.SimpleUpload(ctx, bundlePath(record.Id), b)
}

// AddSettingToBundle adds the given setting to the bundle with the given bundleID.
func (s *Store) AddSettingToBundle(bundleID string, setting *settingsmsg.Setting) (*settingsmsg.Setting, error) {
	s.Init()
	b, err := s.ReadBundle(bundleID)
	if err != nil {
		if !errors.Is(err, settings.ErrNotFound) {
			return nil, err
		}
		b = new(settingsmsg.Bundle)
		b.Id = bundleID
		b.Type = settingsmsg.Bundle_TYPE_DEFAULT
	}

	if setting.Id == "" {
		setting.Id = uuid.NewString()
	}

	b.Settings = append(b.Settings, setting)
	_, err = s.WriteBundle(b)
	return setting, err
}

// RemoveSettingFromBundle removes the setting from the bundle with the given ids.
func (s *Store) RemoveSettingFromBundle(bundleID string, settingID string) error {
	fmt.Println("RemoveSettingFromBundle not implemented")
	return errors.New("not implemented")
}

func bundlePath(id string) string {
	return fmt.Sprintf("%s/%s", bundleFolderLocation, id)
}
