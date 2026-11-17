package etcd

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"go-micro.dev/v4/store"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/namespace"
)

const (
	prefixNS = ".prefix"
	suffixNS = ".suffix"
)

// Store is a store implementation which uses etcd to store the data
type Store struct {
	options store.Options
	client  *clientv3.Client
}

// NewStore creates a new go-micro store backed by etcd
func NewStore(opts ...store.Option) store.Store {
	es := &Store{}
	_ = es.Init(opts...)
	return es
}

func (es *Store) getCtx() (context.Context, context.CancelFunc) {
	currentCtx := es.options.Context
	if currentCtx == nil {
		currentCtx = context.TODO()
	}
	ctx, cancel := context.WithTimeout(currentCtx, 10*time.Second)
	return ctx, cancel
}

// Setup the etcd client based on the current options. The old client (if any)
// will be closed.
// Currently, only the etcd nodes are configurable. If no node is provided,
// it will use the "127.0.0.1:2379" node.
// Context timeout is setup to 10 seconds, and dial timeout to 2 seconds
func (es *Store) setupClient() {
	if es.client != nil {
		es.client.Close()
	}

	endpoints := []string{"127.0.0.1:2379"}
	if len(es.options.Nodes) > 0 {
		endpoints = es.options.Nodes
	}

	cli, _ := clientv3.New(clientv3.Config{
		DialTimeout: 2 * time.Second,
		Endpoints:   endpoints,
	})

	es.client = cli
}

// Init initializes the go-micro store implementation.
// Currently, only the nodes are configurable, the rest of the options
// will be ignored.
func (es *Store) Init(opts ...store.Option) error {
	optList := store.Options{}
	for _, opt := range opts {
		opt(&optList)
	}

	es.options = optList
	es.setupClient()
	return nil
}

// Options returns the store options
func (es *Store) Options() store.Options {
	return es.options
}

// Get the effective TTL, as int64 number of seconds. It will prioritize
// the TTL set in the options, then the expiry time in the options, and
// finally the one set as part of the record
func getEffectiveTTL(r *store.Record, opts store.WriteOptions) int64 {
	// set base ttl duration and expiration time based on the record
	duration := r.Expiry

	// overwrite ttl duration and expiration time based on options
	if !opts.Expiry.IsZero() {
		// options.Expiry is a time.Time, newRecord.Expiry is a time.Duration
		duration = time.Until(opts.Expiry)
	}

	// TTL option takes precedence over expiration time
	if opts.TTL != 0 {
		duration = opts.TTL
	}

	// use milliseconds because it returns an int64 instead of a float64
	return duration.Milliseconds() / 1000
}

// Write the record into the etcd. The record will be duplicated in order to
// find it by prefix or by suffix. This means that it will take double space.
// Note that this is an implementation detail and it will be handled
// transparently.
//
// Database and Table options will be used to provide a different prefix to
// the key. Each service using this store should use a different database+table
// combination in order to prevent key collisions.
//
// Due to how TTLs are implemented in etcd, the minimum valid TTL seems to
// be 2 secs. Using lower values or even negative values will force the etcd
// server to use the minimum value instead.
// In addition, getting a lease for the TTL and attach it to the target key
// are 2 different operations that can't be sent as part of a transaction.
// This means that it's possible to get a lease and have that lease expire
// before attaching it to the key. Errors are expected to happen if this is
// the case, and no key will be inserted.
// According to etcd documentation, the key is guaranteed to be available
// AT LEAST the TTL duration. This means that the key might be available for
// a longer period of time in special circumstances.
//
// It's recommended to use a minimum TTL of 10 secs or higher (or not to use
// TTL) in order to prevent problematic scenarios.
func (es *Store) Write(r *store.Record, opts ...store.WriteOption) error {
	wopts := store.WriteOptions{}
	for _, opt := range opts {
		opt(&wopts)
	}

	prefix := buildPrefix(wopts.Database, wopts.Table, prefixNS)
	suffix := buildPrefix(wopts.Database, wopts.Table, suffixNS)

	kv := es.client.KV

	jsonRecord, err := json.Marshal(r)
	if err != nil {
		return err
	}
	jsonStringRecord := string(jsonRecord)

	effectiveTTL := getEffectiveTTL(r, wopts)
	var opOpts []clientv3.OpOption

	if effectiveTTL != 0 {
		lease := es.client.Lease
		ctx, cancel := es.getCtx()
		gResp, gErr := lease.Grant(ctx, getEffectiveTTL(r, wopts))
		cancel()
		if gErr != nil {
			return gErr
		}
		opOpts = []clientv3.OpOption{clientv3.WithLease(gResp.ID)}
	} else {
		opOpts = []clientv3.OpOption{clientv3.WithLease(0)}
	}

	ctx, cancel := es.getCtx()
	_, err = kv.Txn(ctx).Then(
		clientv3.OpPut(prefix+r.Key, jsonStringRecord, opOpts...),
		clientv3.OpPut(suffix+reverseString(r.Key), jsonStringRecord, opOpts...),
	).Commit()
	cancel()

	return err
}

// Process a Get response taking into account the provided offset
func processGetResponse(resp *clientv3.GetResponse, offset int64) ([]*store.Record, error) {
	result := make([]*store.Record, 0, len(resp.Kvs))
	for index, kvs := range resp.Kvs {
		if int64(index) < offset {
			// skip entries before the offset
			continue
		}

		value := &store.Record{}
		err := json.Unmarshal(kvs.Value, value)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// Process a List response taking into account the provided offset.
// The reverse flag will be used to reverse the keys found. For example,
// "zyxw" will be reversed to "wxyz". This is used for suffix searches,
// where the keys are stored reversed and need to be changed
func processListResponse(resp *clientv3.GetResponse, offset int64, reverse bool) ([]string, error) {
	result := make([]string, 0, len(resp.Kvs))
	for index, kvs := range resp.Kvs {
		if int64(index) < offset {
			// skip entries before the offset
			continue
		}

		targetKey := string(kvs.Key)
		if reverse {
			targetKey = reverseString(targetKey)
		}
		result = append(result, targetKey)
	}
	return result, nil
}

// Perform an exact key read and return the result
func (es *Store) directRead(kv clientv3.KV, key string) ([]*store.Record, error) {
	ctx, cancel := es.getCtx()
	resp, err := kv.Get(ctx, key)
	cancel()
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, store.ErrNotFound
	}

	return processGetResponse(resp, 0)
}

// Perform a prefix read with limit and offset. A limit of 0 will return all
// results. Usage of offset isn't recommended because those results must still
// be fethed from the server in order to be discarded.
func (es *Store) prefixRead(kv clientv3.KV, key string, limit, offset int64) ([]*store.Record, error) {
	getOptions := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}
	if limit > 0 {
		getOptions = append(getOptions, clientv3.WithLimit(limit+offset))
	}

	ctx, cancel := es.getCtx()
	resp, err := kv.Get(ctx, key, getOptions...)
	cancel()
	if err != nil {
		return nil, err
	}
	return processGetResponse(resp, offset)
}

// Perform a prefix + suffix read with limit and offset. A limit of 0 will
// return all results found. Usage of this function is discouraged because
// we'll have to request a prefix search and match the suffix manually. This
// means that even with a limit = 3 and offset = 0, there is no guarantee
// we'll find all the results we need within that range, and we'll likely
// need to request more data from the server. The number of requests we need
// to perform is unknown and might cause load.
func (es *Store) prefixSuffixRead(kv clientv3.KV, prefix, suffix string, limit, offset int64) ([]*store.Record, error) {
	firstKeyOut := firstKeyOutOfPrefixString(prefix)
	getOptions := []clientv3.OpOption{
		clientv3.WithRange(firstKeyOut),
	}

	if limit > 0 {
		// unlikely to find all the entries we need within offset + limit
		getOptions = append(getOptions, clientv3.WithLimit((limit+offset)*2))
	}

	var currentRecordOffset int64
	result := []*store.Record{}
	initialKey := prefix

	keepGoing := true
	for keepGoing {
		ctx, cancel := es.getCtx()
		resp, respErr := kv.Get(ctx, initialKey, getOptions...)
		cancel()
		if respErr != nil {
			return nil, respErr
		}

		records, err := processGetResponse(resp, 0)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			if !strings.HasSuffix(record.Key, suffix) {
				continue
			}

			if currentRecordOffset < offset {
				currentRecordOffset++
				continue
			}

			if !shouldFinish(int64(len(result)), limit) {
				result = append(result, record)
				if shouldFinish(int64(len(result)), limit) {
					break
				}
			}
		}
		if !resp.More || shouldFinish(int64(len(result)), limit) {
			keepGoing = false
		} else {
			initialKey = string(append(resp.Kvs[len(resp.Kvs)-1].Key, 0)) // append byte 0 (nul char) to the last key
		}
	}
	return result, nil
}

// Read records from the etcd server based in the key. Database and Table
// options are highly recommended, otherwise we'll use a default one (which
// might not have the requested keys)
//
// If no prefix or suffix option is provided, we'll read the record matching
// the provided key. Note that a list of records will be provided anyway,
// likely with only one record (the one requested)
//
// Prefix and suffix options are supported and should perform fine even with
// a large amount of data. Note that the limit option should also be included
// in order to limit the amount of records we need to fetch.
//
// Note that using both prefix and suffix options at the same time is possible
// but discouraged. A prefix search will be send to the etcd server, and from
// there we'll manually pick the records matching the suffix. This might become
// very inefficient since we might need to request more data to the etcd
// multiple times in order to provide the results asked.
// Usage of the offset option is also discouraged because we'll have to request
// records that we'll have to skip manually on our side.
//
// Don't rely on any particular order of the keys. The records are expected to
// be sorted by key except if the suffix option (suffix without prefix) is
// used. In this case, the keys will be sorted based on the reversed key
func (es *Store) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	ropts := store.ReadOptions{}
	for _, opt := range opts {
		opt(&ropts)
	}

	prefix := buildPrefix(ropts.Database, ropts.Table, prefixNS)
	suffix := buildPrefix(ropts.Database, ropts.Table, suffixNS)

	kv := es.client.KV
	preKv := namespace.NewKV(kv, prefix)
	sufKv := namespace.NewKV(kv, suffix)

	if ropts.Prefix && ropts.Suffix {
		return es.prefixSuffixRead(preKv, key, key, int64(ropts.Limit), int64(ropts.Offset))
	}

	if ropts.Prefix {
		return es.prefixRead(preKv, key, int64(ropts.Limit), int64(ropts.Offset))
	}

	if ropts.Suffix {
		return es.prefixRead(sufKv, reverseString(key), int64(ropts.Limit), int64(ropts.Offset))
	}

	return es.directRead(preKv, key)
}

// Delete the record containing the key provided. Database and Table
// options are highly recommended, otherwise we'll use a default one (which
// might not have the requested keys)
//
// Since the Write method inserts 2 entries for a given key, those both
// entries will also be removed using the same key. This is handled
// transparently.
func (es *Store) Delete(key string, opts ...store.DeleteOption) error {
	dopts := store.DeleteOptions{}
	for _, opt := range opts {
		opt(&dopts)
	}

	prefix := buildPrefix(dopts.Database, dopts.Table, prefixNS)
	suffix := buildPrefix(dopts.Database, dopts.Table, suffixNS)

	kv := es.client.KV

	ctx, cancel := es.getCtx()
	_, err := kv.Txn(ctx).Then(
		clientv3.OpDelete(prefix+key),
		clientv3.OpDelete(suffix+reverseString(key)),
	).Commit()
	cancel()

	return err
}

// List the keys based on the provided prefix. Use the empty string (and no
// limit nor offset) to list all keys available.
// Limit and offset options are available to limit the keys we need to return.
// The reverse option will reverse the keys before returning them. Use it when
// listing the keys from the suffix KV.
//
// Note that values for the keys won't be requested to the etcd server, that's
// why the reverse option is important
func (es *Store) listKeys(kv clientv3.KV, prefixKey string, limit, offset int64, reverse bool) ([]string, error) {
	getOptions := []clientv3.OpOption{
		clientv3.WithKeysOnly(),
		clientv3.WithPrefix(),
	}
	if limit > 0 {
		getOptions = append(getOptions, clientv3.WithLimit(limit+offset))
	}

	ctx, cancel := es.getCtx()
	resp, err := kv.Get(ctx, prefixKey, getOptions...)
	cancel()
	if err != nil {
		return nil, err
	}

	return processListResponse(resp, offset, reverse)
}

// List the keys matching both prefix and suffix, with the provided limit and
// offset. Usage of this function is discouraged because we'll have to match
// the suffix manually on our side, which means we'll likely need to perform
// additional requests to the etcd server to get more results matching all the
// requirements.
func (es *Store) prefixSuffixList(kv clientv3.KV, prefix, suffix string, limit, offset int64) ([]string, error) {
	firstKeyOut := firstKeyOutOfPrefixString(prefix)
	getOptions := []clientv3.OpOption{
		clientv3.WithKeysOnly(),
		clientv3.WithRange(firstKeyOut),
	}
	if firstKeyOut == "" {
		// could happen of all bytes are "\xff"
		getOptions = getOptions[:1] // remove the WithRange option
	}

	if limit > 0 {
		// unlikely to find all the entries we need within offset + limit
		getOptions = append(getOptions, clientv3.WithLimit((limit+offset)*2))
	}

	var currentRecordOffset int64
	result := []string{}
	initialKey := prefix

	keepGoing := true
	for keepGoing {
		ctx, cancel := es.getCtx()
		resp, respErr := kv.Get(ctx, initialKey, getOptions...)
		cancel()
		if respErr != nil {
			return nil, respErr
		}

		keys, err := processListResponse(resp, 0, false)
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			if !strings.HasSuffix(key, suffix) {
				continue
			}

			if currentRecordOffset < offset {
				currentRecordOffset++
				continue
			}

			if !shouldFinish(int64(len(result)), limit) {
				result = append(result, key)
				if shouldFinish(int64(len(result)), limit) {
					break
				}
			}
		}
		if !resp.More || shouldFinish(int64(len(result)), limit) {
			keepGoing = false
		} else {
			initialKey = string(append(resp.Kvs[len(resp.Kvs)-1].Key, 0)) // append byte 0 (nul char) to the last key
		}
	}
	return result, nil
}

// List the keys available in the etcd server. Database and Table
// options are highly recommended, otherwise we'll use a default one (which
// might not have the requested keys)
//
// With the Database and Table options, all the keys returned will be within
// that database and table. Each service is expected to use a different
// database + table, so using those options will list only the keys used by
// that particular service.
//
// Prefix and suffix options are available along with the limit and offset
// ones.
//
// Using prefix and suffix options at the same time is discourage because
// the suffix matching will be done on our side, and we'll likely need to
// perform multiple requests to get the requested results. Note that using
// just the suffix option is fine.
// In addition, using the offset option is also discouraged because we'll
// need to request additional keys that will be skipped on our side.
func (es *Store) List(opts ...store.ListOption) ([]string, error) {
	lopts := store.ListOptions{}
	for _, opt := range opts {
		opt(&lopts)
	}

	prefix := buildPrefix(lopts.Database, lopts.Table, prefixNS)
	suffix := buildPrefix(lopts.Database, lopts.Table, suffixNS)

	kv := es.client.KV
	preKv := namespace.NewKV(kv, prefix)
	sufKv := namespace.NewKV(kv, suffix)

	if lopts.Prefix != "" && lopts.Suffix != "" {
		return es.prefixSuffixList(preKv, lopts.Prefix, lopts.Suffix, int64(lopts.Limit), int64(lopts.Offset))
	}

	if lopts.Prefix != "" {
		return es.listKeys(preKv, lopts.Prefix, int64(lopts.Limit), int64(lopts.Offset), false)
	}

	if lopts.Suffix != "" {
		return es.listKeys(sufKv, reverseString(lopts.Suffix), int64(lopts.Limit), int64(lopts.Offset), true)
	}

	return es.listKeys(preKv, "", int64(lopts.Limit), int64(lopts.Offset), false)
}

// Close the client
func (es *Store) Close() error {
	return es.client.Close()
}

// Return the service name
func (es *Store) String() string {
	return "Etcd"
}
