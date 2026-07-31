package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack/v5"
)

const activitylogVersionKey = "activitylog.version"
const currentMigrationVersion = "1"

// RunMigrations checks the activitylog data version and runs migrations if necessary.
// It should be called during service startup, after the NATS KeyValue store is initialized.
func (a *ActivitylogService) runMigrations(ctx context.Context, kv nats.KeyValue) error {
	entry, err := kv.Get(activitylogVersionKey)
	if err == nats.ErrKeyNotFound {
		a.log.Info().Msg("activitylog version key not found. Running migration to V1...")
		return a.migrateToV1(ctx, kv)
	} else if err != nil {
		return fmt.Errorf("failed to get activitylog version from NATS KV store: %w", err)
	}

	version := string(entry.Value())
	if version == currentMigrationVersion {
		a.log.Debug().Str("currentVersion", version).Msg("No migration needed")
		return nil
	}

	// If version is something else, it might indicate a future version or an unexpected state.
	// Add logic here if more complex version handling is needed.
	return fmt.Errorf("unexpected activitylog version: %s, expected %s or older", version, currentMigrationVersion)
}

// migrateToV1 performs the data migration to version 1.
// It iterates over all keys, expecting their values to be JSON arrays of strings.
// For each such key, it creates a new key in the format "originalKey.count.timestamp"
// and stores the original list of strings (re-marshalled to messagepack) as its value.
// Finally, it sets the activitylog.version key to "1".
func (a *ActivitylogService) migrateToV1(_ context.Context, kv nats.KeyValue) error {
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
		if key == activitylogVersionKey {
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
	if _, err := kv.PutString(activitylogVersionKey, currentMigrationVersion); err != nil {
		return fmt.Errorf("migrateToV1: failed to set activitylog version key to '%s' in NATS KV store: %w", currentMigrationVersion, err)
	}

	a.log.Info().Int("migrated", migratedCount).Int("skipped", skippedCount).Msg("Migration to V1 complete")
	return nil
}
