package search_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var _ = DescribeTable("PinnedRootID",
	func(query, want string) {
		Expect(search.PinnedRootID(query)).To(Equal(want))
	},
	Entry("bare driveId is completed to the root id", `driveId:"1$2"`, "1$2!2"),
	Entry("full root id passes through", `driveId:"1$2!4" name:x`, "1$2!4"),
	Entry("RootID works as well", `RootID:"1$2" AND name:x`, "1$2!2"),
	Entry("top-level AND conjuncts pin", `driveId:"1$2" AND name:x AND tag:y`, "1$2!2"),
	Entry("repeated identical AND restrictions pin", `driveId:"1$2" AND driveId:"1$2"`, "1$2!2"),
	// KQL groups adjacent same-key restrictions into an implicit OR group
	Entry("implicitly repeated restrictions do not pin", `driveId:"1$2" driveId:"1$2"`, ""),
	Entry("no restriction, no pin", `name:x`, ""),
	Entry("a top-level OR disables pruning", `name:x OR driveId:"1$2"`, ""),
	Entry("OR between other terms disables pruning", `driveId:"1$2" AND (a OR b)`, "1$2!2"),
	Entry("a negated restriction disables pruning", `NOT driveId:"1$2" AND name:x`, ""),
	Entry("restrictions inside groups do not pin", `(driveId:"1$2") AND name:x`, ""),
	Entry("two different roots disable pruning", `driveId:"1$2" AND driveId:"1$3"`, ""),
	Entry("invalid query, no pin", `((`, ""),
	Entry("a wildcard drive id does not pin", `driveId:"1$*" AND name:x`, ""),
	Entry("a wildcard space id does not pin", `driveId:"1$2*"`, ""),
)

var _ = DescribeTable("CompleteRootID",
	func(in, want string) {
		Expect(search.CompleteRootID(in)).To(Equal(want))
	},
	Entry("bare drive id gains the space id", "1$2", "1$2!2"),
	Entry("complete id passes through", "1$2!4", "1$2!4"),
	Entry("wildcards pass through", "1$*", "1$*"),
	Entry("question marks pass through", "1$2?", "1$2?"),
	Entry("no separator passes through", "abc", "abc"),
)
