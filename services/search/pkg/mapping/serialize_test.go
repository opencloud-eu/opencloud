package mapping

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PrepareForIndex serialization", func() {
	It("errors for a non-marshallable value", func() {
		// a func field can't be json-marshalled -> conversions.To errors
		type bad struct {
			F func() `json:"f"`
		}
		_, err := PrepareForIndex(bad{}, nil)
		Expect(err).To(HaveOccurred())
	})

	It("returns nil map for a typed nil pointer", func() {
		// a typed nil pointer marshals to null -> nil map, no error, no panic
		out, err := PrepareForIndex((*struct{})(nil), nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(BeNil())
	})

	It("flattens embedded structs", func() {
		type inner struct {
			Name string `json:"Name"`
			Size uint64 `json:"Size"`
		}
		type outer struct {
			inner
			ID string `json:"ID"`
		}
		m, err := PrepareForIndex(outer{inner: inner{Name: "a", Size: 7}, ID: "x"}, nil)
		Expect(err).ToNot(HaveOccurred())
		want := map[string]any{"Name": "a", "Size": float64(7), "ID": "x"}
		Expect(m).To(Equal(want))
	})

	It("omits nil fields tagged omitempty", func() {
		type facet struct {
			Artist string `json:"artist"`
		}
		type doc struct {
			Name  string `json:"Name"`
			Audio *facet `json:"audio,omitempty"`
		}
		m, err := PrepareForIndex(doc{Name: "n"}, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(m).ToNot(HaveKey("audio"))
		Expect(m["Name"]).To(Equal("n"))
	})

	It("includes nested facets when set", func() {
		type facet struct {
			Artist string `json:"artist"`
		}
		type doc struct {
			Audio *facet `json:"audio,omitempty"`
		}
		m, err := PrepareForIndex(doc{Audio: &facet{Artist: "A"}}, nil)
		Expect(err).ToNot(HaveOccurred())
		nested, ok := m["audio"].(map[string]any)
		Expect(ok).To(BeTrue(), "audio should be a nested map: %#v", m["audio"])
		Expect(nested["artist"]).To(Equal("A"))
	})
})
