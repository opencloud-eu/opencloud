package svc_test

import (
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/reva/v2/pkg/openextension"
)

// The codec lives in reva (WebDAV and graph share it); its behavior is pinned
// here because the graph endpoints and the search index build on it.
var _ = Describe("open extension codec", func() {
	const project = "com.example.project"
	key := func(name, property string) string {
		return "http://opencloud.eu/ns/extensions/" + name + "/" + property
	}

	parse := func(body string) openextension.Patch {
		GinkgoHelper()
		p, err := openextension.Parse([]byte(body))
		Expect(err).NotTo(HaveOccurred())
		return p
	}

	rejects := func(body, reason string) {
		GinkgoHelper()
		_, err := openextension.Parse([]byte(body))
		Expect(err).To(MatchError(openextension.ErrInvalid))
		Expect(err.Error()).To(ContainSubstring(reason))
	}

	Describe("Parse", func() {
		It("infers the kind from the JSON type", func() {
			p := parse(`{"extensionName":"com.example.project","status":"reviewed","priority":3,"done":false,"tags":["a","b"]}`)
			Expect(p.Set).To(HaveLen(4))
			Expect(p.Set["status"].Kind).To(Equal(openextension.KindString))
			Expect(p.Set["priority"].Kind).To(Equal(openextension.KindNumber))
			Expect(p.Set["priority"].IsInteger()).To(BeTrue())
			Expect(p.Set["done"].Kind).To(Equal(openextension.KindBool))
			Expect(p.Set["tags"].Kind).To(Equal(openextension.KindString))
			Expect(p.Set["tags"].Array).To(BeTrue())
			Expect(p.Set["tags"].Strings()).To(Equal([]string{"a", "b"}))
		})

		It("keeps the JSON as written", func() {
			p := parse(`{"n":"34","m":34,"f":1.50}`)
			Expect(string(p.Set["n"].Raw)).To(Equal(`"34"`))
			Expect(p.Set["n"].Kind).To(Equal(openextension.KindString))
			Expect(string(p.Set["m"].Raw)).To(Equal(`34`))
			Expect(string(p.Set["f"].Raw)).To(Equal(`1.50`))
			Expect(p.Set["f"].IsInteger()).To(BeFalse())
		})

		It("takes dates and geo points from their annotations", func() {
			p := parse(`{"due":"2026-10-01T00:00:00Z","due@odata.type":"#DateTimeOffset",` +
				`"site":{"latitude":52.5,"longitude":13.4},"site@odata.type":"#microsoft.graph.geoCoordinates",` +
				`"dates":["2026-10-01T00:00:00Z"],"dates@odata.type":"#Collection(DateTimeOffset)"}`)
			Expect(p.Set["due"].Kind).To(Equal(openextension.KindDate))
			Expect(p.Set["due"].Times()).To(HaveLen(1))
			Expect(p.Set["site"].Kind).To(Equal(openextension.KindGeo))
			geo, ok := p.Set["site"].Geo()
			Expect(ok).To(BeTrue())
			Expect(geo.Latitude).To(Equal(52.5))
			Expect(geo.Longitude).To(Equal(13.4))
			Expect(geo.Altitude).To(BeNil())
			Expect(p.Set["dates"].Kind).To(Equal(openextension.KindDate))
			Expect(p.Set["dates"].Array).To(BeTrue())
		})

		It("treats a date-time string without annotation as a string", func() {
			p := parse(`{"due":"2026-10-01T00:00:00Z"}`)
			Expect(p.Set["due"].Kind).To(Equal(openextension.KindString))
		})

		It("removes properties sent as null", func() {
			p := parse(`{"status":"approved","priority":null}`)
			Expect(p.Set).To(HaveKey("status"))
			Expect(p.Remove).To(Equal([]string{"priority"}))
		})

		It("accepts the optional inferable annotations", func() {
			p := parse(`{"a":"x","a@odata.type":"#String","b":3,"b@odata.type":"#Int64","c":2.5,"c@odata.type":"#Double","d":true,"d@odata.type":"#Boolean","e":[1,2],"e@odata.type":"#Collection(Int64)"}`)
			Expect(p.Set).To(HaveLen(5))
		})

		It("rejects a value that does not match its annotation", func() {
			rejects(`{"due":"next week","due@odata.type":"#DateTimeOffset"}`, "not an RFC 3339 date-time")
			rejects(`{"n":"34","n@odata.type":"#Int64"}`, "is a string but annotated as #Int64")
			rejects(`{"n":1.5,"n@odata.type":"#Int64"}`, "is not an integer")
			rejects(`{"n":1,"n@odata.type":"#Boolean"}`, "annotated as #Boolean")
			rejects(`{"n":1,"n@odata.type":"#Whatever"}`, "annotated as #Whatever")
		})

		It("rejects objects unless they are annotated geo coordinates", func() {
			rejects(`{"assignee":{"name":"alice"}}`, "objects are only allowed as geoCoordinates")
			rejects(`{"site":{"latitude":52.5},"site@odata.type":"#microsoft.graph.geoCoordinates"}`, "geoCoordinates need numeric latitude and longitude")
			rejects(`{"site":{"latitude":91,"longitude":0},"site@odata.type":"#microsoft.graph.geoCoordinates"}`, "geoCoordinates need numeric latitude and longitude")
			rejects(`{"site":{"latitude":1,"longitude":2,"name":"x"},"site@odata.type":"#microsoft.graph.geoCoordinates"}`, "geoCoordinates need numeric latitude and longitude")
		})

		It("rejects nested and mixed arrays", func() {
			rejects(`{"a":[[1]]}`, "arrays hold scalars only")
			rejects(`{"a":[1,"x"]}`, "array elements must all be number")
			rejects(`{"a":[{"latitude":1,"longitude":2}],"a@odata.type":"#Collection(microsoft.graph.geoCoordinates)"}`, "arrays hold scalars only")
		})

		It("rejects bad property names, dangling annotations and oversized values", func() {
			rejects(`{"a.b":1}`, "must be a simple identifier")
			rejects(`{"1a":1}`, "must be a simple identifier")
			rejects(`{"a@odata.type":"#String"}`, "annotates a property that is not in the body")
			rejects(`[1]`, "body must be a JSON object")
			rejects(`{"a":"`+strings.Repeat("a", openextension.MaxValueSize)+`"}`, "at most")
		})
	})

	Describe("stored form", func() {
		It("prefixes every value with the code of its kind", func() {
			p := parse(`{"s":"open","n":3,"f":1.50,"b":true,"t":"2026-10-01T00:00:00Z","t@odata.type":"#DateTimeOffset",` +
				`"g":{"latitude":52.5,"longitude":13.4,"altitude":34},"g@odata.type":"#microsoft.graph.geoCoordinates",` +
				`"l":["a","b"],"m":[1,2.5],"c":[true],"d":["2026-10-01T00:00:00Z"],"d@odata.type":"#Collection(DateTimeOffset)"}`)
			encoded := map[string]string{}
			for k, v := range p.Set {
				encoded[k] = openextension.EncodeValue(v)
			}
			Expect(encoded).To(Equal(map[string]string{
				"s": "s:open",
				"n": "n:3",
				"f": "n:1.50",
				"b": "b:true",
				"t": "d:2026-10-01T00:00:00Z",
				"g": "g:52.5,13.4,34",
				"l": `S:["a","b"]`,
				"m": `N:[1,2.5]`,
				"c": `B:[true]`,
				"d": `D:["2026-10-01T00:00:00Z"]`,
			}))
		})

		It("round-trips every kind", func() {
			p := parse(`{"s":"a:b, c","n":3,"f":1.50,"b":false,"t":"2026-10-01T00:00:00Z","t@odata.type":"#DateTimeOffset",` +
				`"g":{"latitude":52.5,"longitude":13.4},"g@odata.type":"#microsoft.graph.geoCoordinates",` +
				`"l":["a","b"],"m":[1,2.5],"d":["2026-10-01T00:00:00Z"],"d@odata.type":"#Collection(DateTimeOffset)"}`)
			for k, v := range p.Set {
				back, err := openextension.DecodeValue(openextension.EncodeValue(v))
				Expect(err).NotTo(HaveOccurred(), k)
				Expect(back).To(Equal(v), k)
			}
		})

		It("rejects a value without a known code or with a payload that does not fit it", func() {
			for _, raw := range []string{"open", "", "x:1", "34", "http://example.org", "s", "n:abc", "n:[1]", "b:maybe", "d:tomorrow", "g:1", "g:91,0", "S:1", "N:[1,\"x\"]", "D:[\"x\"]"} {
				_, err := openextension.DecodeValue(raw)
				Expect(err).To(MatchError(openextension.ErrInvalid), raw)
			}
		})

		It("turns a patch into metadata keys", func() {
			set, unset := parse(`{"status":"open","priority":3,"old":null,"due":"2026-10-01T00:00:00Z","due@odata.type":"#DateTimeOffset"}`).Metadata(project)
			Expect(set).To(Equal(map[string]string{
				key(project, "status"):   "s:open",
				key(project, "priority"): "n:3",
				key(project, "due"):      "d:2026-10-01T00:00:00Z",
			}))
			Expect(unset).To(Equal([]string{key(project, "old")}))
		})

		It("applies removals together with sets", func() {
			ext := openextension.OpenExtension{Name: "x.y"}
			ext.Apply(parse(`{"a":1,"b":2}`))
			ext.Apply(parse(`{"a":null,"c":3}`))
			Expect(ext.Keys()).To(Equal([]string{"b", "c"}))
		})
	})

	Describe("MarshalJSON", func() {
		It("annotates only what JSON cannot express", func() {
			ext := openextension.OpenExtension{Name: "com.example.project"}
			ext.Apply(parse(`{"status":"open","priority":3,"due":"2026-10-01T00:00:00Z","due@odata.type":"#DateTimeOffset","tags":["a"]}`))
			out, err := json.Marshal(ext)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal(`{"extensionName":"com.example.project","due":"2026-10-01T00:00:00Z","due@odata.type":"#DateTimeOffset","priority":3,"status":"open","tags":["a"]}`))
		})

		It("reports the OData type of every kind", func() {
			p := parse(`{"s":"x","i":3,"d":2.5,"b":true,"l":["a"],"n":[1],"t":"2026-10-01T00:00:00Z","t@odata.type":"#DateTimeOffset","g":{"latitude":1,"longitude":2},"g@odata.type":"#microsoft.graph.geoCoordinates"}`)
			Expect(p.Set["s"].ODataType()).To(Equal("#String"))
			Expect(p.Set["i"].ODataType()).To(Equal("#Int64"))
			Expect(p.Set["d"].ODataType()).To(Equal("#Double"))
			Expect(p.Set["b"].ODataType()).To(Equal("#Boolean"))
			Expect(p.Set["l"].ODataType()).To(Equal("#Collection(String)"))
			Expect(p.Set["n"].ODataType()).To(Equal("#Collection(Double)"))
			Expect(p.Set["t"].ODataType()).To(Equal("#DateTimeOffset"))
			Expect(p.Set["g"].ODataType()).To(Equal("#microsoft.graph.geoCoordinates"))
		})
	})

	Describe("names and keys", func() {
		It("requires reverse DNS extension names", func() {
			Expect(openextension.ValidateName("com.example.project")).To(Succeed())
			Expect(openextension.ValidateName("eu.opencloud.work-flow_2")).To(Succeed())
			Expect(openextension.ValidateName("project")).To(MatchError(openextension.ErrInvalid))
			Expect(openextension.ValidateName("com..example")).To(MatchError(openextension.ErrInvalid))
			Expect(openextension.ValidateName("com/example")).To(MatchError(openextension.ErrInvalid))
		})

		It("maps names and properties to namespaces and metadata keys", func() {
			Expect(openextension.Namespace(project)).To(Equal("http://opencloud.eu/ns/extensions/com.example.project"))
			Expect(openextension.Key(project, "status")).To(Equal(key(project, "status")))

			name, ok := openextension.NameFromNamespace("http://opencloud.eu/ns/extensions/com.example.project")
			Expect(ok).To(BeTrue())
			Expect(name).To(Equal(project))
			for _, ns := range []string{"http://opencloud.eu/ns/extensions/", "http://opencloud.eu/ns/extensions/a/b", "http://owncloud.org/ns", "DAV:"} {
				_, ok := openextension.NameFromNamespace(ns)
				Expect(ok).To(BeFalse(), ns)
			}

			name, property, ok := openextension.SplitKey(key(project, "status"))
			Expect(ok).To(BeTrue())
			Expect(name).To(Equal(project))
			Expect(property).To(Equal("status"))
			for _, k := range []string{"tags", "http://owncloud.org/ns/favorite", "http://opencloud.eu/ns/extensions/com.example.project", "http://opencloud.eu/ns/extensions/com.example.project/", "http://opencloud.eu/ns/extensions//status"} {
				_, _, ok := openextension.SplitKey(k)
				Expect(ok).To(BeFalse(), k)
			}
		})

		It("collects the extensions out of arbitrary metadata", func() {
			metadata := map[string]string{
				"tags":                        "a,b",
				key("com.example.b", "x"):     "n:1",
				key("com.example.a", "y"):     "s:z",
				key("com.example.a", "plain"): "written without a type code",
				key("com.example.a", "bad"):   "n:abc",
			}
			exts := openextension.FromMetadata(metadata)
			Expect(exts).To(HaveLen(2))
			Expect(exts[0].Name).To(Equal("com.example.a"))
			Expect(exts[0].Keys()).To(Equal([]string{"y"}), "unreadable values are skipped")
			Expect(exts[1].Name).To(Equal("com.example.b"))

			ext, found := openextension.Lookup(metadata, "com.example.b")
			Expect(found).To(BeTrue())
			Expect(ext.Values["x"].Numbers()).To(Equal([]float64{1}))
			_, found = openextension.Lookup(metadata, "com.example")
			Expect(found).To(BeFalse(), "a name that is a prefix of another does not match")

			Expect(openextension.MetadataKeys(metadata, "com.example.a")).To(Equal([]string{key("com.example.a", "bad"), key("com.example.a", "plain"), key("com.example.a", "y")}))
		})
	})

	Describe("WebDAV", func() {
		It("renders every kind with its xsi:type", func() {
			p := parse(`{"s":"a<b","i":3,"d":2.5,"b":true,"t":"2026-10-01T00:00:00Z","t@odata.type":"#DateTimeOffset",` +
				`"g":{"latitude":52.5,"longitude":13.4,"altitude":34},"g@odata.type":"#microsoft.graph.geoCoordinates","l":["x","y"],"n":[1,2]}`)
			render := func(key string) [2]string {
				d := openextension.ToDAV(p.Set[key])
				return [2]string{d.Type, string(d.InnerXML)}
			}
			Expect(render("s")).To(Equal([2]string{"", "a&lt;b"}))
			Expect(render("i")).To(Equal([2]string{"xs:integer", "3"}))
			Expect(render("d")).To(Equal([2]string{"xs:decimal", "2.5"}))
			Expect(render("b")).To(Equal([2]string{"xs:boolean", "true"}))
			Expect(render("t")).To(Equal([2]string{"xs:dateTime", "2026-10-01T00:00:00Z"}))
			Expect(render("g")).To(Equal([2]string{"oc:geoCoordinates", "<oc:latitude>52.5</oc:latitude><oc:longitude>13.4</oc:longitude><oc:altitude>34</oc:altitude>"}))
			Expect(render("l")).To(Equal([2]string{"oc:list", "<oc:item>x</oc:item><oc:item>y</oc:item>"}))
			Expect(render("n")).To(Equal([2]string{"oc:list", `<oc:item xsi:type="xs:integer">1</oc:item><oc:item xsi:type="xs:integer">2</oc:item>`}))
		})

		It("parses PROPPATCH values, a property without xsi:type is a string", func() {
			from := func(inner, typ string) openextension.Value {
				GinkgoHelper()
				v, err := openextension.FromDAV("p", []byte(inner), typ)
				Expect(err).NotTo(HaveOccurred())
				return v
			}
			Expect(from("34", "")).To(Equal(openextension.Value{Kind: openextension.KindString, Raw: json.RawMessage(`"34"`)}))
			Expect(from("a &amp; b", "")).To(Equal(openextension.Value{Kind: openextension.KindString, Raw: json.RawMessage(`"a & b"`)}))
			Expect(from("34", "xs:integer")).To(Equal(openextension.Value{Kind: openextension.KindNumber, Raw: json.RawMessage(`34`)}))
			Expect(from(" 2.5 ", "xsd:decimal")).To(Equal(openextension.Value{Kind: openextension.KindNumber, Raw: json.RawMessage(`2.5`)}))
			Expect(from("1", "xs:boolean")).To(Equal(openextension.Value{Kind: openextension.KindBool, Raw: json.RawMessage(`true`)}))
			Expect(from("2026-10-01T00:00:00Z", "xs:dateTime")).To(Equal(openextension.Value{Kind: openextension.KindDate, Raw: json.RawMessage(`"2026-10-01T00:00:00Z"`)}))
			Expect(from("<oc:latitude>52.5</oc:latitude><x:longitude xmlns:x=\"http://owncloud.org/ns\">13.4</x:longitude>", "oc:geoCoordinates")).
				To(Equal(openextension.Value{Kind: openextension.KindGeo, Raw: json.RawMessage(`{"latitude":52.5,"longitude":13.4}`)}))
			Expect(from(`<oc:item xsi:type="xs:integer">1</oc:item><oc:item xsi:type="xs:integer">2</oc:item>`, "oc:list")).
				To(Equal(openextension.Value{Kind: openextension.KindNumber, Array: true, Raw: json.RawMessage(`[1,2]`)}))
			Expect(from(`<oc:item>a</oc:item><oc:item>b</oc:item>`, "oc:list")).
				To(Equal(openextension.Value{Kind: openextension.KindString, Array: true, Raw: json.RawMessage(`["a","b"]`)}))
		})

		It("rejects PROPPATCH values that do not fit their type", func() {
			for _, c := range [][2]string{{"abc", "xs:integer"}, {"maybe", "xs:boolean"}, {"tomorrow", "xs:dateTime"}, {"<oc:latitude>x</oc:latitude>", "oc:geoCoordinates"}, {"<oc:item>1</oc:item><oc:item xsi:type=\"xs:integer\">2</oc:item>", "oc:list"}} {
				_, err := openextension.FromDAV("p", []byte(c[0]), c[1])
				Expect(err).To(MatchError(openextension.ErrInvalid), "%s as %s", c[0], c[1])
			}
		})

		It("round-trips a value through WebDAV and the stored form", func() {
			p := parse(`{"n":[1,2.5],"g":{"latitude":52.5,"longitude":13.4},"g@odata.type":"#microsoft.graph.geoCoordinates","t":"2026-10-01T00:00:00Z","t@odata.type":"#DateTimeOffset"}`)
			for key, v := range p.Set {
				d := openextension.ToDAV(v)
				back, err := openextension.FromDAV(key, d.InnerXML, d.Type)
				Expect(err).NotTo(HaveOccurred())
				Expect(back).To(Equal(v))
				stored, err := openextension.DecodeValue(openextension.EncodeValue(back))
				Expect(err).NotTo(HaveOccurred())
				Expect(stored).To(Equal(v))
			}
		})
	})
})
