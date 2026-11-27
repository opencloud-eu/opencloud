package activitylog_test

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/uuid"
	nserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/command"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/config"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/activitylog"
)

var (
	server *nserver.Server
	tmpdir string
)

func getFreeLocalhostPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return -1, err
	}

	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close() // Close the listener immediately to free the port
	return port, nil
}

// Spawn a nats server and a JetStream instance for the duration of the test suite.
// The different tests need to make sure to use different databases to avoid conflicts.
var _ = SynchronizedBeforeSuite(func() {
	port, err := getFreeLocalhostPort()
	server, err = nserver.NewServer(&nserver.Options{
		Port: port,
	})
	Expect(err).ToNot(HaveOccurred())

	tmpdir, err = os.MkdirTemp("", "activitylog-test")
	natsdir := filepath.Join(tmpdir, "nats-js")
	jsConf := &nserver.JetStreamConfig{
		StoreDir: natsdir,
	}
	// first start NATS
	go server.Start()
	time.Sleep(time.Second)

	// second start JetStream
	err = server.EnableJetStream(jsConf)
	Expect(err).ToNot(HaveOccurred())
}, func() {})

var _ = SynchronizedAfterSuite(func() {
	server.Shutdown()
	_ = os.RemoveAll(tmpdir)
}, func() {})

var _ = Describe("ActivitylogService", func() {
	var (
		alog                *activitylog.ActivityLog
		getResource         func(_ context.Context, ref *provider.Reference) (*provider.ResourceInfo, error)
		writebufferduration = 100 * time.Millisecond
	)

	JustBeforeEach(func() {
		var err error
		db := "activitylog-test-" + uuid.New().String()

		kv, js, err := command.ConnectNatsKV(config.Store{
			Nodes:    []string{server.Addr().String()},
			Database: db,
		}, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())

		alog, err = activitylog.New(
			kv,
			activitylog.JetStream(js),
			activitylog.MaxActivities(4),
			activitylog.WriteBufferDuration(writebufferduration),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("with a noop debouncer", func() {
		BeforeEach(func() {
			writebufferduration = 0
		})

		Describe("AddActivity", func() {
			type testCase struct {
				Name       string
				Tree       map[string]*provider.ResourceInfo
				Activities map[string]string
				Expected   map[string][]activitylog.RawActivity
			}

			testCases := []testCase{
				{
					Name: "simple",
					Tree: map[string]*provider.ResourceInfo{
						"base":    resourceInfo("base", "parent"),
						"parent":  resourceInfo("parent", "spaceid"),
						"spaceid": resourceInfo("spaceid", "spaceid"),
					},
					Activities: map[string]string{
						"activity": "base",
					},
					Expected: map[string][]activitylog.RawActivity{
						"base":    activitites("activity", 0),
						"parent":  activitites("activity", 1),
						"spaceid": activitites("activity", 2),
					},
				},
				{
					Name: "two activities on same resource",
					Tree: map[string]*provider.ResourceInfo{
						"base":    resourceInfo("base", "parent"),
						"parent":  resourceInfo("parent", "spaceid"),
						"spaceid": resourceInfo("spaceid", "spaceid"),
					},
					Activities: map[string]string{
						"activity1": "base",
						"activity2": "base",
					},
					Expected: map[string][]activitylog.RawActivity{
						"base":    activitites("activity1", 0, "activity2", 0),
						"parent":  activitites("activity1", 1, "activity2", 1),
						"spaceid": activitites("activity1", 2, "activity2", 2),
					},
				},
				// Add other test cases here...
			}

			for _, tc := range testCases {
				Context(tc.Name, func() {
					JustBeforeEach(func() {
						getResource = func(_ context.Context, ref *provider.Reference) (*provider.ResourceInfo, error) {
							return tc.Tree[ref.GetResourceId().GetOpaqueId()], nil
						}

						for k, v := range tc.Activities {
							err := alog.AddActivity(context.Background(), reference(v), nil, k, time.Time{}, getResource)
							Expect(err).NotTo(HaveOccurred())
						}
					})

					It("should match the expected activities", func() {
						for id, acts := range tc.Expected {
							activities, err := alog.Activities(resourceID(id))
							Expect(err).NotTo(HaveOccurred(), tc.Name+":"+id)
							Expect(activities).To(ConsistOf(acts), tc.Name+":"+id)
						}
					})
				})
			}
		})
	})

	Context("with a debouncing debouncer", func() {
		var (
			tree = map[string]*provider.ResourceInfo{
				"base":    resourceInfo("base", "parent"),
				"parent":  resourceInfo("parent", "spaceid"),
				"spaceid": resourceInfo("spaceid", "spaceid"),
			}
		)

		BeforeEach(func() {
			writebufferduration = 100 * time.Millisecond
		})

		Describe("addActivity", func() {
			var (
				getResource = func(_ context.Context, ref *provider.Reference) (*provider.ResourceInfo, error) {
					return tree[ref.GetResourceId().GetOpaqueId()], nil
				}
			)

			It("debounces activities", func() {

				err := alog.AddActivity(context.Background(), reference("base"), nil, "activity1", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())
				err = alog.AddActivity(context.Background(), reference("base"), nil, "activity2", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func(g Gomega) {
					activities, err := alog.Activities(resourceID("base"))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(activities).To(ConsistOf(activitites("activity1", 0, "activity2", 0)))
				}).Should(Succeed())
			})

			It("adheres to the MaxActivities setting", func() {
				err := alog.AddActivity(context.Background(), reference("base"), nil, "activity1", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())
				Eventually(func(g Gomega) {
					activities, err := alog.Activities(resourceID("base"))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(len(activities)).To(Equal(1))
				}).Should(Succeed())

				err = alog.AddActivity(context.Background(), reference("base"), nil, "activity2", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())
				Eventually(func(g Gomega) {
					activities, err := alog.Activities(resourceID("base"))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(len(activities)).To(Equal(2))
				}).Should(Succeed())

				err = alog.AddActivity(context.Background(), reference("base"), nil, "activity3", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())
				err = alog.AddActivity(context.Background(), reference("base"), nil, "activity4", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())
				err = alog.AddActivity(context.Background(), reference("base"), nil, "activity5", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func(g Gomega) {
					activities, err := alog.Activities(resourceID("base"))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(activities).To(ConsistOf(activitites("activity2", 0, "activity3", 0, "activity4", 0, "activity5", 0)))
				}).Should(Succeed())
			})
		})

		Describe("Activities", func() {
			It("combines multiple batches", func() {
				getResource = func(_ context.Context, ref *provider.Reference) (*provider.ResourceInfo, error) {
					return tree[ref.GetResourceId().GetOpaqueId()], nil
				}

				err := alog.AddActivity(context.Background(), reference("base"), nil, "activity1", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())
				err = alog.AddActivity(context.Background(), reference("base"), nil, "activity2", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func(g Gomega) {
					activities, err := alog.Activities(resourceID("base"))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(activities).To(ConsistOf(activitites("activity1", 0, "activity2", 0)))
				}).Should(Succeed())

				err = alog.AddActivity(context.Background(), reference("base"), nil, "activity3", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())
				err = alog.AddActivity(context.Background(), reference("base"), nil, "activity4", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func(g Gomega) {
					activities, err := alog.Activities(resourceID("base"))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(activities).To(ConsistOf(activitites("activity1", 0, "activity2", 0, "activity3", 0, "activity4", 0)))
				}).Should(Succeed())
			})
		})
	})
})

var _ = Describe("ConnectNatsKV", func() {
	var (
		db  string
		cfg config.Store
		js  nats.JetStreamContext
	)

	maxAge := func() time.Duration {
		info, err := js.StreamInfo("KV_" + db)
		Expect(err).ToNot(HaveOccurred())
		return info.Config.MaxAge
	}

	metadata := func() map[string]string {
		info, err := js.StreamInfo("KV_" + db)
		Expect(err).ToNot(HaveOccurred())
		return info.Config.Metadata
	}

	BeforeEach(func() {
		db = "activitylog-ttl-test-" + uuid.New().String()
		cfg = config.Store{
			Nodes:    []string{server.Addr().String()},
			Database: db,
		}

		conn, err := nats.Connect(server.Addr().String())
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(conn.Close)
		js, err = conn.JetStream()
		Expect(err).ToNot(HaveOccurred())
	})

	It("creates a new bucket without expiry by default", func() {
		_, _, err := command.ConnectNatsKV(cfg, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(maxAge()).To(Equal(time.Duration(0)))
	})

	It("applies the TTL to a new bucket", func() {
		cfg.TTL = time.Hour
		_, _, err := command.ConnectNatsKV(cfg, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(maxAge()).To(Equal(time.Hour))
	})

	It("updates the TTL of an existing bucket", func() {
		_, _, err := command.ConnectNatsKV(cfg, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(maxAge()).To(Equal(time.Duration(0)))

		cfg.TTL = time.Hour
		_, _, err = command.ConnectNatsKV(cfg, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(maxAge()).To(Equal(time.Hour))

		cfg.TTL = time.Second
		_, _, err = command.ConnectNatsKV(cfg, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(maxAge()).To(Equal(time.Second))

		cfg.TTL = 0
		_, _, err = command.ConnectNatsKV(cfg, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(maxAge()).To(Equal(time.Duration(0)))
	})

	It("falls back to no expiry for an invalid TTL on a new bucket", func() {
		for _, ttl := range []time.Duration{-time.Hour, 50 * time.Millisecond} {
			db = "activitylog-ttl-test-" + uuid.New().String()
			cfg.Database = db
			cfg.TTL = ttl
			_, _, err := command.ConnectNatsKV(cfg, log.NopLogger())
			Expect(err).ToNot(HaveOccurred())
			Expect(maxAge()).To(Equal(time.Duration(0)))
		}
	})

	It("falls back to no expiry for an invalid TTL on an existing bucket", func() {
		for _, ttl := range []time.Duration{-time.Hour, 50 * time.Millisecond} {
			db = "activitylog-ttl-test-" + uuid.New().String()
			cfg.Database = db
			cfg.TTL = time.Hour
			_, _, err := command.ConnectNatsKV(cfg, log.NopLogger())
			Expect(err).ToNot(HaveOccurred())
			Expect(maxAge()).To(Equal(time.Hour))

			cfg.TTL = ttl
			_, _, err = command.ConnectNatsKV(cfg, log.NopLogger())
			Expect(err).ToNot(HaveOccurred())
			Expect(maxAge()).To(Equal(time.Duration(0)))
		}
	})

	It("copies the version key to the stream metadata of an existing bucket", func() {
		kv, _, err := command.ConnectNatsKV(cfg, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		_, err = kv.PutString(activitylog.ActivityLogVersionKey, "1")
		Expect(err).ToNot(HaveOccurred())

		_, _, err = command.ConnectNatsKV(cfg, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(metadata()).To(HaveKeyWithValue(activitylog.ActivityLogVersionMetadataKey, "1"))
		Expect(maxAge()).To(Equal(time.Duration(0)))

		entry, err := kv.Get(activitylog.ActivityLogVersionKey)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(entry.Value())).To(Equal("1"))
	})

	It("copies the version key to the stream metadata while applying a TTL", func() {
		kv, _, err := command.ConnectNatsKV(cfg, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		_, err = kv.PutString(activitylog.ActivityLogVersionKey, "1")
		Expect(err).ToNot(HaveOccurred())

		cfg.TTL = time.Hour
		_, _, err = command.ConnectNatsKV(cfg, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(metadata()).To(HaveKeyWithValue(activitylog.ActivityLogVersionMetadataKey, "1"))
		Expect(maxAge()).To(Equal(time.Hour))
	})

	It("keeps the version stream metadata when the version key is missing", func() {
		_, _, err := command.ConnectNatsKV(cfg, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		info, err := js.StreamInfo("KV_" + db)
		Expect(err).ToNot(HaveOccurred())
		streamCfg := info.Config
		streamCfg.Metadata = map[string]string{activitylog.ActivityLogVersionMetadataKey: "1"}
		_, err = js.UpdateStream(&streamCfg)
		Expect(err).ToNot(HaveOccurred())

		cfg.TTL = time.Hour
		_, _, err = command.ConnectNatsKV(cfg, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(metadata()).To(HaveKeyWithValue(activitylog.ActivityLogVersionMetadataKey, "1"))
		Expect(maxAge()).To(Equal(time.Hour))
	})

	It("does not set the version stream metadata when there is no version", func() {
		_, _, err := command.ConnectNatsKV(cfg, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())

		_, _, err = command.ConnectNatsKV(cfg, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(metadata()).ToNot(HaveKey(activitylog.ActivityLogVersionMetadataKey))
	})
})

var _ = Describe("migrations", func() {
	const legacyKey = "legacy-key"

	var (
		kv nats.KeyValue
		js nats.JetStreamContext
		db string
	)

	setStreamVersion := func(version string) {
		info, err := js.StreamInfo("KV_" + db)
		Expect(err).ToNot(HaveOccurred())
		streamCfg := info.Config
		streamCfg.Metadata = map[string]string{activitylog.ActivityLogVersionMetadataKey: version}
		_, err = js.UpdateStream(&streamCfg)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		var err error
		db = "activitylog-migration-test-" + uuid.New().String()
		kv, js, err = command.ConnectNatsKV(config.Store{
			Nodes:    []string{server.Addr().String()},
			Database: db,
		}, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())

		data, err := json.Marshal([]activitylog.RawActivity{{EventID: "activity", Depth: 0}})
		Expect(err).ToNot(HaveOccurred())
		envelope, err := json.Marshal(map[string]any{
			"key":      "resource",
			"data":     data,
			"metadata": map[string]any{},
		})
		Expect(err).ToNot(HaveOccurred())
		_, err = kv.Put(legacyKey, envelope)
		Expect(err).ToNot(HaveOccurred())
	})

	It("does not migrate when the stream metadata has the current version", func() {
		setStreamVersion("1")

		_, err := activitylog.New(kv, activitylog.JetStream(js), activitylog.Logger(log.NopLogger()))
		Expect(err).ToNot(HaveOccurred())

		_, err = kv.Get(legacyKey)
		Expect(err).ToNot(HaveOccurred())
		_, err = kv.Get(activitylog.ActivityLogVersionKey)
		Expect(err).To(MatchError(nats.ErrKeyNotFound))
	})

	It("does not migrate when only the version key has the current version", func() {
		_, err := kv.PutString(activitylog.ActivityLogVersionKey, "1")
		Expect(err).ToNot(HaveOccurred())

		_, err = activitylog.New(kv, activitylog.JetStream(js), activitylog.Logger(log.NopLogger()))
		Expect(err).ToNot(HaveOccurred())

		_, err = kv.Get(legacyKey)
		Expect(err).ToNot(HaveOccurred())
	})

	It("migrates when there is no version", func() {
		_, err := activitylog.New(kv, activitylog.JetStream(js), activitylog.Logger(log.NopLogger()))
		Expect(err).ToNot(HaveOccurred())

		_, err = kv.Get(legacyKey)
		Expect(err).To(MatchError(nats.ErrKeyNotFound))
		info, err := js.StreamInfo("KV_" + db)
		Expect(err).ToNot(HaveOccurred())
		Expect(info.Config.Metadata).To(HaveKeyWithValue(activitylog.ActivityLogVersionMetadataKey, "1"))
		_, err = kv.Get(activitylog.ActivityLogVersionKey)
		Expect(err).To(MatchError(nats.ErrKeyNotFound))
	})

	It("stores the version key when no JetStream context is configured", func() {
		_, err := activitylog.New(kv, activitylog.Logger(log.NopLogger()))
		Expect(err).ToNot(HaveOccurred())

		_, err = kv.Get(legacyKey)
		Expect(err).To(MatchError(nats.ErrKeyNotFound))
		entry, err := kv.Get(activitylog.ActivityLogVersionKey)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(entry.Value())).To(Equal("1"))
	})

	It("does not migrate again after the migration stored the version in the stream metadata", func() {
		_, err := activitylog.New(kv, activitylog.JetStream(js), activitylog.Logger(log.NopLogger()))
		Expect(err).ToNot(HaveOccurred())

		_, err = kv.Put("second-legacy-key", []byte("not migrated"))
		Expect(err).ToNot(HaveOccurred())

		_, err = activitylog.New(kv, activitylog.JetStream(js), activitylog.Logger(log.NopLogger()))
		Expect(err).ToNot(HaveOccurred())

		entry, err := kv.Get("second-legacy-key")
		Expect(err).ToNot(HaveOccurred())
		Expect(string(entry.Value())).To(Equal("not migrated"))
	})

	It("fails when the stream metadata has an unexpected version", func() {
		setStreamVersion("2")

		_, err := activitylog.New(kv, activitylog.JetStream(js), activitylog.Logger(log.NopLogger()))
		Expect(err).To(MatchError(ContainSubstring("unexpected activitylog version")))
	})
})

func activitites(acts ...any) []activitylog.RawActivity {
	var activities []activitylog.RawActivity
	act := activitylog.RawActivity{}
	for _, a := range acts {
		switch v := a.(type) {
		case string:
			act.EventID = v
		case int:
			act.Depth = v
			activities = append(activities, act)
		}
	}
	return activities
}

func resourceID(id string) *provider.ResourceId {
	return &provider.ResourceId{
		StorageId: "storageid",
		OpaqueId:  id,
		SpaceId:   "spaceid",
	}
}

func reference(id string) *provider.Reference {
	return &provider.Reference{ResourceId: resourceID(id)}
}

func resourceInfo(id, parentID string) *provider.ResourceInfo {
	return &provider.ResourceInfo{
		Id:       resourceID(id),
		ParentId: resourceID(parentID),
		Space: &provider.StorageSpace{
			Root: resourceID("spaceid"),
		},
	}
}
