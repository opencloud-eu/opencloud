package parity

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// parityCase is one request, sent to every generator. A generator listed in
// generatorOverrides is held to that answer instead of want, which is how a
// known divergence stays visible without failing. One that is no longer true
// fails, so it cannot rot.
type parityCase struct {
	title              string
	path               string
	source             source
	partType           string
	want               string
	generatorOverrides map[string]override

	// probe adds to the answer what the header cannot say, like a crop position
	probe func(data []byte) string

	// for requests a multipart image post cannot express
	method         string
	rawBody        []byte
	rawContentType string
}

// override is what a generator answers today where it still differs from want,
// and why that is tolerated.
type override struct {
	answers string
	because string
}

func (c parityCase) wantFor(g testGenerator) (string, string) {
	for _, key := range g.overrideKeys {
		if o, ok := c.generatorOverrides[key]; ok {
			return o.answers, o.because
		}
	}

	return c.want, ""
}

func (c parityCase) ask(g testGenerator) string {
	GinkgoHelper()

	if c.method != "" || c.rawContentType != "" {
		method := c.method
		if method == "" {
			method = http.MethodPost
		}

		return c.describe(g.requestRaw(method, c.path, c.rawBody, c.rawContentType))
	}

	partType := c.partType
	if partType == "" {
		partType = c.source.partType
	}

	return c.describe(g.request(c.path, c.source.data, c.source.filename, partType))
}

func (c parityCase) describe(a answer) string {
	GinkgoHelper()

	out := a.String()
	if c.probe == nil || a.status != http.StatusOK {
		return out
	}

	return out + " " + c.probe(a.data)
}

type parityGroup struct {
	name  string
	cases []parityCase
}

func parityGroups() []parityGroup {
	return []parityGroup{
		resizeGroup(),
		formatGroup(),
		animationGroup(),
		errorsGroup(),
		limitsGroup(),
		iconGroup(),
		transparencyGroup(),
		grammarGroup(),
	}
}

// Every case runs once per generator as its own spec, so one generator failing
// still leaves the other's answer in the output. The groups are Ordered
// containers: their generators are built once in a BeforeAll, and they carry on
// after a failure so every generator gets to answer.
var _ = Describe("requests", func() {
	for _, group := range parityGroups() {
		Describe(group.name, Ordered, ContinueOnFailure, func() {
			var generators []testGenerator

			BeforeAll(func() { generators = newGenerators() })

			for _, c := range group.cases {
				Describe(c.title, func() {
					for _, name := range generatorNames {
						It("on "+name, func() {
							g := generatorNamed(generators, name)
							if g.unavailable != "" {
								Skip(g.unavailable)
							}

							want, because := c.wantFor(g)
							if because != "" {
								because = "\n" + because
							}

							got := c.ask(g)
							Expect(got).To(Equal(want),
								"%s answered %s for %s (%s)%s", name, got, c.path, c.source.name, because)
						})
					}
				})
			}
		})
	}
})

func generatorNamed(generators []testGenerator, name string) testGenerator {
	GinkgoHelper()

	for _, g := range generators {
		if g.name == name {
			return g
		}
	}

	Fail("no generator named " + name)

	return testGenerator{}
}
