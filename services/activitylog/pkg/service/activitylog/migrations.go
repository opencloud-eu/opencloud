package activitylog

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack/v5"
)

// ActivityLogVersionKey is the KV key holding the activitylog data version.
const ActivityLogVersionKey = "activitylog.version"

// ActivityLogVersionMetadataKey is the stream metadata key holding the activitylog data version.
const ActivityLogVersionMetadataKey = "opencloud.activitylog.version"
const currentMigrationVersion = "1"

// RunMigrations checks the activitylog data version and runs migrations if necessary.
// It should be called during service startup, after the NATS KeyValue store is initialized.
func (a *ActivityLog) runMigrations(ctx context.Context, kv nats.KeyValue) error {
	version, err := a.storedVersion(kv)
	if err != nil {
		return err
	}
	if version == "" {
		a.log.Info().Msg("activitylog version key not found. Running migration to V1...")
		return a.migrateToV1(ctx, kv)
	}

	if version == currentMigrationVersion {
		a.log.Debug().Str("currentVersion", version).Msg("No migration needed")
		return nil
	}

	// If version is something else, it might indicate a future version or an unexpected state.
	// Add logic here if more complex version handling is needed.
	return fmt.Errorf("unexpected activitylog version: %s, expected %s or older", version, currentMigrationVersion)
}

// storedVersion returns the activitylog data version, or an empty string if none is stored.
// The stream metadata is preferred, the version key of the KV store is the fallback.
func (a *ActivityLog) storedVersion(kv nats.KeyValue) (string, error) {
	if a.js != nil {
		info, err := a.js.StreamInfo("KV_" + kv.Bucket())
		if err != nil {
			return "", fmt.Errorf("failed to get activitylog version from NATS stream info: %w", err)
		}
		if version := info.Config.Metadata[ActivityLogVersionMetadataKey]; version != "" {
			return version, nil
		}
	}

	entry, err := kv.Get(ActivityLogVersionKey)
	if err == nats.ErrKeyNotFound {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("failed to get activitylog version from NATS KV store: %w", err)
	}
	return string(entry.Value()), nil
}

// migrateToV1 performs the data migration to version 1.
// It iterates over all keys, expecting their values to be JSON arrays of strings.
// For each such key, it creates a new key in the format "originalKey.count.timestamp"
// and stores the original list of strings (re-marshalled to messagepack) as its value.
// Finally, it sets the activitylog version to "1" in the NATS stream metadata
// (or in the activitylog.version key if no JetStream context is configured).
func (a *ActivityLog) migrateToV1(_ context.Context, kv nats.KeyValue) error {
	lister, err := kv.ListKeys()
	if err != nil {
		return fmt.Errorf("migrateToV1: failed to list keys from NATS KV store: %w", err)
	}

	migratedCount := 0
	skippedCount := 0

	keyChan := lister.Keys()
	defer lister.Stop()

	// keyValueEnvelope is the data structure used by the go micro plugin which was used previously.
	type keyValueEnvelope struct {
		Key      string         `json:"key"`
		Data     []byte         `json:"data"`
		Metadata map[string]any `json:"metadata"`
	}

	for key := range keyChan {
		if key == ActivityLogVersionKey {
			skippedCount++
			continue // Skip the version key itself
		}

		// Get the original value
		entry, err := kv.Get(key)
		if err != nil {
			a.log.Error().Err(err).Str("key", key).Msg("migrateToV1: Failed to get value for key. Skipping.")
			skippedCount++
			continue
		}
		valBytes := entry.Value()

		val := keyValueEnvelope{}
		// Unmarshal the value into the keyValueEnvelope structure
		if err := json.Unmarshal(valBytes, &val); err != nil {
			a.log.Error().Err(err).Str("key", key).Msg("migrateToV1: Value for key ss not a keyValueEnvelope. Skipping.")
			skippedCount++
			continue
		}

		// Unmarshal value into a list of strings
		var activities []RawActivity
		if err := msgpack.Unmarshal(val.Data, &activities); err != nil {
			if err := json.Unmarshal(val.Data, &activities); err != nil {
				// This key's value is not a JSON array of strings. Skip it.
				a.log.Error().Err(err).Str("key", key).Msg("migrateToV1: Value for key is not a msgback or JSON array of strings. Skipping.")
				skippedCount++
				continue
			}
		}

		// Construct the new key
		newKey := natsKey(val.Key, len(activities))
		newValue, err := msgpack.Marshal(activities)
		if err != nil {
			a.log.Error().Err(err).Str("key", key).Msg("migrateToV1: Failed to marshal activities. Skipping.")
			skippedCount++
			continue
		}

		// Write the value (the list of strings, marshalled as messagepack) under the new key
		if _, err := kv.Put(newKey, newValue); err != nil {
			a.log.Error().Err(err).Str("newKey", newKey).Str("key", key).Msg("migrateToV1: Failed to put new key. Skipping.")
			skippedCount++
			continue
		}

		// delete old key, it's no longer needed
		if err := kv.Delete(key); err != nil {
			log.Printf("migrateToV1: Failed to delete old key '%s' after migration: %v. Skipping deletion.", key, err)
			skippedCount++
			continue
		}

		migratedCount++
	}

	// Set the activitylog version to "1" after migration
	if err := a.storeVersion(kv, currentMigrationVersion); err != nil {
		return err
	}

	a.log.Info().Int("migrated", migratedCount).Int("skipped", skippedCount).Msg("Migration to V1 complete")
	return nil
}

// storeVersion stores the activitylog data version in the stream metadata, which is not affected by the
// stream MaxAge. Without JetStream it falls back to the version key of the KV store.
func (a *ActivityLog) storeVersion(kv nats.KeyValue, version string) error {
	if a.js == nil {
		if _, err := kv.PutString(ActivityLogVersionKey, version); err != nil {
			return fmt.Errorf("migrateToV1: failed to set activitylog version key to '%s' in NATS KV store: %w", version, err)
		}
		return nil
	}

	info, err := a.js.StreamInfo("KV_" + kv.Bucket())
	if err != nil {
		return fmt.Errorf("migrateToV1: failed to get NATS stream info: %w", err)
	}
	cfg := info.Config
	if cfg.Metadata == nil {
		cfg.Metadata = map[string]string{}
	}
	cfg.Metadata[ActivityLogVersionMetadataKey] = version
	if _, err := a.js.UpdateStream(&cfg); err != nil {
		return fmt.Errorf("migrateToV1: failed to set activitylog version to '%s' in the NATS stream metadata: %w", version, err)
	}
	return nil
}
