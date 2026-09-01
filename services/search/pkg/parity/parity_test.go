package parity

import (
	"context"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

	"github.com/opencloud-eu/reva/v2/pkg/errtypes"

	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

type queryCase struct {
	id             int
	query          string
	want           []string
	limit          int32
	wantCount      *int
	ref            *searchMessage.Reference
	wantBadRequest bool

	// engineOverrides holds, per engine, what that engine answers today where
	// it still differs from the expectation above. The spec asserts the
	// override and the README marks the row as a known divergence. Once the
	// engine answers as expected the override fails and has to go.
	engineOverrides map[string]override
}

// override is one engine's current answer to a case, in the expectation's
// own terms.
type override struct {
	want           []string
	wantCount      *int
	wantBadRequest bool
}

func (o override) matcher() types.GomegaMatcher {
	switch {
	case o.wantBadRequest:
		return ConsistOf("bad request")
	case o.wantCount != nil:
		return HaveLen(*o.wantCount)
	default:
		return matchNames(o.want)
	}
}

// rendered spells the override the way the README spells an expectation.
func (o override) rendered() string {
	switch {
	case o.wantBadRequest:
		return "bad request"
	case o.wantCount != nil:
		return fmt.Sprintf("%d items", *o.wantCount)
	default:
		return matrixNames(o.want)
	}
}

func renderOverrides(overrides map[string]override) map[string]string {
	if len(overrides) == 0 {
		return nil
	}

	out := make(map[string]string, len(overrides))
	for engine, o := range overrides {
		out[engine] = o.rendered()
	}

	return out
}

func (c queryCase) label(group string) string {
	return fmt.Sprintf("%s-%02d", strings.ToUpper(group), c.id)
}

func (c queryCase) name(group string) string {
	return c.label(group) + " " + c.query
}

type queryGroup struct {
	name     string
	fixtures []search.Resource
	cases    []queryCase
}

func queryGroups() []queryGroup {
	return []queryGroup{
		nameGroup(),
		extensionGroup(),
		tagsGroup(),
		titleGroup(),
		contentGroup(),
		cjkGroup(),
		favoritesGroup(),
		mediatypeGroup(),
		pathGroup(),
		fieldsGroup(),
		deletedGroup(),
		visibilityGroup(),
		booleanGroup(),
		samenameGroup(),
		stressGroup(),
		everythingGroup(),
		rangeGroup(),
		scopeGroup(),
		invalidGroup(),
	}
}

type responseCase struct {
	id    int
	do    func(e search.Engine) error
	after string
	query string
	reads string
	read  func(*searchService.SearchIndexResponse) []string
	want  []string

	engineOverrides map[string]override
}

func (c responseCase) label(group string) string {
	return fmt.Sprintf("%s-%02d", strings.ToUpper(group), c.id)
}

func (c responseCase) name(group string) string {
	return c.label(group) + " " + c.reads
}

type responseGroup struct {
	name     string
	fixtures []search.Resource
	cases    []responseCase
}

func responseGroups() []responseGroup {
	return []responseGroup{
		entityGroup(),
		metadataGroup(),
	}
}

type lifecycleCase struct {
	id      int
	title   string
	do      func(e search.Engine) error
	expect  []expectation
	wantErr bool

	fixtures []search.Resource

	wantDocCount *uint64

	// engineOverrides is the lifecycle spelling of queryCase.engineOverrides:
	// a case has several expectations, so an override answers them by query.
	engineOverrides map[string]lifecycleOverride
}

// lifecycleOverride is one engine's current answers to a lifecycle case.
type lifecycleOverride struct {
	expect       map[string][]string
	wantDocCount *uint64
}

// overridesFor picks one expectation's overrides out of the case's.
func (c lifecycleCase) overridesFor(query string) map[string]override {
	var overrides map[string]override
	for engine, o := range c.engineOverrides {
		if answer, ok := o.expect[query]; ok {
			if overrides == nil {
				overrides = map[string]override{}
			}

			overrides[engine] = override{want: answer}
		}
	}

	return overrides
}

// docCountOverrides renders the DocCount overrides as a query override.
func (c lifecycleCase) docCountOverrides() map[string]override {
	var overrides map[string]override
	for engine, o := range c.engineOverrides {
		if o.wantDocCount != nil {
			if overrides == nil {
				overrides = map[string]override{}
			}

			overrides[engine] = override{want: []string{fmt.Sprint(*o.wantDocCount)}}
		}
	}

	return overrides
}

type expectation struct {
	query string
	want  []string
}

func (c lifecycleCase) label(group string) string {
	return fmt.Sprintf("%s-%02d", strings.ToUpper(group), c.id)
}

func (c lifecycleCase) name(group string) string {
	return c.label(group) + " " + c.title
}

type lifecycleGroup struct {
	name     string
	fixtures []search.Resource
	cases    []lifecycleCase
}

func lifecycleGroups() []lifecycleGroup {
	return []lifecycleGroup{
		deleteLifecycle(),
		restoreLifecycle(),
		purgeLifecycle(),
		purgeSpaceLifecycle(),
		moveLifecycle(),
		rootScopeLifecycle(),
		casePathLifecycle(),
		hiddenLifecycle(),
		upsertLifecycle(),
		idempotencyLifecycle(),
		batchLifecycle(),
	}
}

// expectAnswer holds an engine to its override when it has one, to the
// expectation otherwise. An override that no longer holds fails on purpose:
// the engine answers as expected now, the override has to go.
func expectAnswer(engine string, answer []string, expected override, overrides map[string]override) {
	GinkgoHelper()

	if o, ok := overrides[engine]; ok {
		Expect(o.rendered()).NotTo(Equal(expected.rendered()), "an override that equals the expectation documents nothing, remove it")
		Expect(answer).To(o.matcher(), "the override for %s no longer holds, remove it", engine)

		return
	}

	Expect(answer).To(expected.matcher())
}

// Every case runs once per engine as its own spec, so one engine failing
// leaves the other's answer in the matrix. The groups are Ordered containers:
// their engines are built once in a BeforeAll and shared by the specs inside,
// and they carry on after a failure so every engine gets to answer.

var _ = Describe("Queries", func() {
	for groupAt, group := range queryGroups() {
		Describe(group.name, Ordered, ContinueOnFailure, func() {
			var engines []testEngine

			BeforeAll(func() {
				engines = newEngines("opencloud-test-engine-parity-"+group.name, group.fixtures)
			})

			for caseAt, c := range group.cases {
				row := matrixRow{
					Section: "Queries", Group: group.name, ID: c.label(group.name),
					Query: c.query, Scope: matrixScope(c.ref), Limit: c.limit,
					Want: c.want, WantCount: c.wantCount, WantBadRequest: c.wantBadRequest, Overrides: renderOverrides(c.engineOverrides),
					GroupAt: groupAt, CaseAt: caseAt,
				}
				planRow(row)

				Describe(c.name(group.name), func() {
					for _, name := range engineNames {
						It("on "+name, func() {
							e := engineNamed(engines, name)
							if e.unavailable != "" {
								recordSkip(row, name)
								Skip(e.unavailable)
							}

							request := &searchService.SearchIndexRequest{Query: c.query, PageSize: c.limit, Ref: c.ref}

							expected := override{want: c.want, wantCount: c.wantCount, wantBadRequest: c.wantBadRequest}
							_, overridden := c.engineOverrides[name]

							if c.wantBadRequest {
								_, err := e.backend.Search(context.Background(), request)
								answer := badRequestAnswer(err)
								recordAnswer(row, name, answer)
								if !overridden {
									Expect(err).To(BeAssignableToTypeOf(errtypes.BadRequest("")))
								}

								expectAnswer(name, answer, expected, c.engineOverrides)

								return
							}

							answer, err := ask(e.backend, request)
							recordAnswer(row, name, answer)
							if !overridden {
								Expect(err).NotTo(HaveOccurred(), "the query has to answer, an empty result is an answer")
							}

							expectAnswer(name, answer, expected, c.engineOverrides)
						})
					}
				})
			}
		})
	}
})

var _ = Describe("Operations", func() {
	offset := len(queryGroups())

	for groupAt, group := range lifecycleGroups() {
		Describe(group.name, func() {
			for caseAt, c := range group.cases {
				fixtures := c.fixtures
				if fixtures == nil {
					fixtures = group.fixtures
				}

				index := fmt.Sprintf("opencloud-test-engine-parity-%s-%d", group.name, c.id)
				rows := c.matrixRows(group.name, offset+groupAt, caseAt)
				planRow(rows...)

				// an operation changes its index, so every spec builds its own engine
				Describe(c.name(group.name), func() {
					for _, name := range engineNames {
						It("on "+name, func() {
							e := newEngine(name, index, fixtures)
							if e.unavailable != "" {
								for _, row := range rows {
									recordSkip(row, name)
								}

								Skip(e.unavailable)
							}

							// every row gets its answer before anything is asserted, a
							// failed assertion must not leave the later rows unanswered
							err := c.do(e.backend)
							if failed := err != nil; failed != c.wantErr {
								for _, row := range rows {
									recordAnswer(row, name, badRequestAnswer(err))
								}
							}

							if c.wantErr {
								Expect(err).To(HaveOccurred(), "the operation had to report that it did not find the resource")
							} else {
								Expect(err).NotTo(HaveOccurred(), "the operation under test failed")
							}

							e.settle()

							answers := make([][]string, len(c.expect))
							errs := make([]error, len(c.expect))
							for i, expect := range c.expect {
								answers[i], errs[i] = ask(e.backend, &searchService.SearchIndexRequest{Query: expect.query})
								recordAnswer(rows[i], name, answers[i])
							}

							var count uint64
							if c.wantDocCount != nil {
								count, err = e.backend.DocCount()
								Expect(err).NotTo(HaveOccurred())
								recordAnswer(rows[len(rows)-1], name, []string{fmt.Sprint(count)})
							}

							for i, expect := range c.expect {
								overrides := c.overridesFor(expect.query)
								if _, overridden := overrides[name]; !overridden {
									Expect(errs[i]).NotTo(HaveOccurred(), "the query has to answer, an empty result is an answer")
								}

								expectAnswer(name, answers[i], override{want: expect.want}, overrides)
							}

							if c.wantDocCount != nil {
								expected := override{want: []string{fmt.Sprint(*c.wantDocCount)}}
								expectAnswer(name, []string{fmt.Sprint(count)}, expected, c.docCountOverrides())
							}
						})
					}
				})
			}
		})
	}
})

var _ = Describe("Response", func() {
	offset := len(queryGroups()) + len(lifecycleGroups())

	for groupAt, group := range responseGroups() {
		Describe(group.name, Ordered, ContinueOnFailure, func() {
			var shared []testEngine

			BeforeAll(func() {
				shared = newEngines("opencloud-test-engine-parity-"+group.name, group.fixtures)
			})

			for caseAt, c := range group.cases {
				row := matrixRow{
					Section: "Response", Group: group.name, ID: c.label(group.name),
					Query: c.query, Reads: c.reads, Context: c.after, Want: c.want, Overrides: renderOverrides(c.engineOverrides),
					GroupAt: offset + groupAt, CaseAt: caseAt,
				}
				planRow(row)

				index := fmt.Sprintf("opencloud-test-engine-parity-%s-%d", group.name, c.id)

				Describe(c.name(group.name), func() {
					for _, name := range engineNames {
						It("on "+name, func() {
							// a case with an operation changes its index, so it gets its own
							e := engineNamed(shared, name)
							if c.do != nil {
								e = newEngine(name, index, group.fixtures)
							}

							if e.unavailable != "" {
								recordSkip(row, name)
								Skip(e.unavailable)
							}

							if c.do != nil {
								err := c.do(e.backend)
								if err != nil {
									recordAnswer(row, name, []string{"error"})
								}

								Expect(err).NotTo(HaveOccurred(), "the operation under test failed")
								e.settle()
							}

							resp, err := e.backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: c.query})
							if err != nil {
								recordAnswer(row, name, []string{"error"})
							}
							Expect(err).NotTo(HaveOccurred(), "the query has to answer, an empty result is an answer")

							answer := c.read(resp)
							recordAnswer(row, name, answer)
							if o, overridden := c.engineOverrides[name]; overridden {
								Expect(o.want).NotTo(Equal(c.want), "an override that equals the expectation documents nothing, remove it")
								Expect(answer).To(Equal(o.want), "the override for %s no longer holds, remove it", name)

								return
							}

							Expect(answer).To(Equal(c.want), c.reads)
						})
					}
				})
			}
		})
	}
})
