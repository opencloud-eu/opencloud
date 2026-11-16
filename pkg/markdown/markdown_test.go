package markdown

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	SmallMarkdown = `# Title

some abstract description

## SubTitle 1

subtitle one description

## SubTitle 2
subtitle two description
### Subpoint to SubTitle 2

description to subpoint

more text
`
	SmallMD = MD{
		Headings: []Heading{
			{Level: 1, Header: "Title", Content: "\nsome abstract description\n\n"},
			{Level: 2, Header: "SubTitle 1", Content: "\nsubtitle one description\n\n"},
			{Level: 2, Header: "SubTitle 2", Content: "subtitle two description\n"},
			{Level: 3, Header: "Subpoint to SubTitle 2", Content: "\ndescription to subpoint\n\nmore text\n"},
		},
	}
)

var _ = Describe("TestMarkdown", func() {
	DescribeTable("Conversion works both ways",
		func(mdfile string, expectedMD MD) {
			md := NewMD([]byte(mdfile))

			Expect(len(md.Headings)).To(Equal(len(expectedMD.Headings)))
			for i, h := range md.Headings {
				Expect(h).To(Equal(expectedMD.Headings[i]))
			}
			Expect(md.String()).To(Equal(mdfile))
		},
		Entry("converts a small markdown", SmallMarkdown, SmallMD),
	)
})

var _ = Describe("Table of contents", func() {
	// expected anchors are the output of github-slugger, which GitHub and Docusaurus use
	DescribeTable("creates the same anchors as github-slugger",
		func(header string, expected string) {
			Expect(toAnchor(header)).To(Equal(expected))
		},
		Entry("keeps hyphens", "Define Read-Only Attributes", "define-read-only-attributes"),
		Entry("keeps hyphens inside words", "Manage Trash-Bin Items", "manage-trash-bin-items"),
		Entry("removes slashes but keeps both spaces", "MIME Type Configuration / Creation Allow List", "mime-type-configuration--creation-allow-list"),
		Entry("removes backticks", "The `auth` Service Family", "the-auth-service-family"),
		Entry("removes parentheses", "Via CLI (developer only)", "via-cli-developer-only"),
		Entry("removes a closing parenthesis after a number", "1) Single Process Mode", "1-single-process-mode"),
		Entry("keeps underscores", "OC_ADD_RUN_SERVICES Usage", "oc_add_run_services-usage"),
		Entry("keeps non-ASCII letters", "Größe & Übersicht", "größe--übersicht"),
		Entry("removes other punctuation", "Version 2.0: What's New?", "version-20-whats-new"),
	)

	It("adds a suffix to duplicate anchors", func() {
		md := MD{
			Headings: []Heading{
				{Level: 1, Header: "Title"},
				{Level: 2, Header: "Command Examples"},
				{Level: 2, Header: "Other"},
				{Level: 3, Header: "Command Examples"},
				{Level: 3, Header: "Command Examples"},
				{Level: 2, Header: "Command Examples 1"},
			},
		}

		Expect(md.TocString()).To(Equal(`* [Command Examples](#command-examples)
* [Other](#other)
  * [Command Examples](#command-examples-1)
  * [Command Examples](#command-examples-2)
* [Command Examples 1](#command-examples-1-1)
`))
	})
})
