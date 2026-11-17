package memory

import (
	"time"

	"go-micro.dev/v4/store"
)

func toStoreRecord(src *store.Record, options store.WriteOptions) *storeRecord {
	newRecord := &storeRecord{}
	newRecord.Key = src.Key
	newRecord.Value = make([]byte, len(src.Value))
	copy(newRecord.Value, src.Value)

	// set base ttl duration and expiration time based on the record
	newRecord.Expiry = src.Expiry
	if src.Expiry != 0 {
		newRecord.ExpiresAt = time.Now().Add(src.Expiry)
	}

	// overwrite ttl duration and expiration time based on options
	if !options.Expiry.IsZero() {
		// options.Expiry is a time.Time, newRecord.Expiry is a time.Duration
		newRecord.Expiry = time.Until(options.Expiry)
		newRecord.ExpiresAt = options.Expiry
	}

	// TTL option takes precedence over expiration time
	if options.TTL != 0 {
		newRecord.Expiry = options.TTL
		newRecord.ExpiresAt = time.Now().Add(options.TTL)
	}

	newRecord.Metadata = make(map[string]interface{})
	for k, v := range src.Metadata {
		newRecord.Metadata[k] = v
	}
	return newRecord
}

func fromStoreRecord(src *storeRecord) *store.Record {
	newRecord := &store.Record{}
	newRecord.Key = src.Key
	newRecord.Value = make([]byte, len(src.Value))
	copy(newRecord.Value, src.Value)
	if src.Expiry != 0 {
		newRecord.Expiry = time.Until(src.ExpiresAt)
	}

	newRecord.Metadata = make(map[string]interface{})
	for k, v := range src.Metadata {
		newRecord.Metadata[k] = v
	}
	return newRecord
}

func reverseString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}
