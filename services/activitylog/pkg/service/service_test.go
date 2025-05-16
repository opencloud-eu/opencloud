package service

import (
	"context"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/jellydator/ttlcache/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"go.opentelemetry.io/otel/trace/noop"
)

var _ = Describe("ActivitylogService", func() {
	var (
		alog        *ActivitylogService
		getResource func(ref *provider.Reference) (*provider.ResourceInfo, error)
	)

	Context("with a noop debouncer", func() {
		BeforeEach(func() {
			alog = &ActivitylogService{
				store:         store.Create(),
				tracer:        noop.NewTracerProvider().Tracer("test"),
				parentIdCache: ttlcache.NewCache(),
			}
			alog.debouncer = NewDebouncer(0, alog.storeActivity)
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
				tc := tc // capture range variable
				Context(tc.Name, func() {
					BeforeEach(func() {
						getResource = func(ref *provider.Reference) (*provider.ResourceInfo, error) {
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
			alog = &ActivitylogService{
				store:         store.Create(),
				tracer:        noop.NewTracerProvider().Tracer("test"),
				parentIdCache: ttlcache.NewCache(),
			}
			alog.debouncer = NewDebouncer(100*time.Millisecond, alog.storeActivity)
		})

		It("should debounce activities", func() {
			getResource = func(ref *provider.Reference) (*provider.ResourceInfo, error) {
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
		})
	})
})

func activitites(acts ...interface{}) []RawActivity {
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
