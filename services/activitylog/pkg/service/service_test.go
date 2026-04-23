package service

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	nserver "github.com/nats-io/nats-server/v2/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/config"
	eventsmocks "github.com/opencloud-eu/reva/v2/pkg/events/mocks"
	"github.com/test-go/testify/mock"
	"go.opentelemetry.io/otel/trace/noop"
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
		alog                *ActivitylogService
		getResource         func(_ context.Context, ref *provider.Reference) (*provider.ResourceInfo, error)
		writebufferduration = 100 * time.Millisecond
	)

	JustBeforeEach(func() {
		var err error
		stream := &eventsmocks.Stream{}
		stream.EXPECT().Consume(mock.Anything, mock.Anything).Return(nil, nil)
		alog, err = New(
			Config(&config.Config{
				Service: config.Service{
					Name: "activitylog-test",
				},
				Store: config.Store{
					Store:    "nats-js-kv",
					Nodes:    []string{server.Addr().String()},
					Database: "activitylog-test-" + uuid.New().String(),
				},
				MaxActivities:       4,
				WriteBufferDuration: writebufferduration,
			}),
			Stream(stream),
			TraceProvider(noop.NewTracerProvider()),
			Mux(chi.NewMux()),
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
				Expected   map[string][]RawActivity
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
					Expected: map[string][]RawActivity{
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
					Expected: map[string][]RawActivity{
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
							err := alog.addActivity(context.Background(), reference(v), nil, k, time.Time{}, getResource)
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

				err := alog.addActivity(context.Background(), reference("base"), nil, "activity1", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())
				err = alog.addActivity(context.Background(), reference("base"), nil, "activity2", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func(g Gomega) {
					activities, err := alog.Activities(resourceID("base"))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(activities).To(ConsistOf(activitites("activity1", 0, "activity2", 0)))
				}).Should(Succeed())
			})

			It("adheres to the MaxActivities setting", func() {
				err := alog.addActivity(context.Background(), reference("base"), nil, "activity1", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())
				Eventually(func(g Gomega) {
					activities, err := alog.Activities(resourceID("base"))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(len(activities)).To(Equal(1))
				}).Should(Succeed())

				err = alog.addActivity(context.Background(), reference("base"), nil, "activity2", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())
				Eventually(func(g Gomega) {
					activities, err := alog.Activities(resourceID("base"))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(len(activities)).To(Equal(2))
				}).Should(Succeed())

				err = alog.addActivity(context.Background(), reference("base"), nil, "activity3", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())
				err = alog.addActivity(context.Background(), reference("base"), nil, "activity4", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())
				err = alog.addActivity(context.Background(), reference("base"), nil, "activity5", time.Time{}, getResource)
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

				err := alog.addActivity(context.Background(), reference("base"), nil, "activity1", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())
				err = alog.addActivity(context.Background(), reference("base"), nil, "activity2", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func(g Gomega) {
					activities, err := alog.Activities(resourceID("base"))
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(activities).To(ConsistOf(activitites("activity1", 0, "activity2", 0)))
				}).Should(Succeed())

				err = alog.addActivity(context.Background(), reference("base"), nil, "activity3", time.Time{}, getResource)
				Expect(err).NotTo(HaveOccurred())
				err = alog.addActivity(context.Background(), reference("base"), nil, "activity4", time.Time{}, getResource)
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

func activitites(acts ...any) []RawActivity {
	var activities []RawActivity
	act := RawActivity{}
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
