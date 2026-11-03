package middleware

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

var _ = DescribeTable("reading the profile picture claim", func(claim string, input map[string]any, want string) {
	m := accountResolver{logger: log.NopLogger(), pictureClaim: claim}
	Expect(m.readProfilePictureURL(input, "profile-picture-user")).To(Equal(want))
},
	Entry("disabled when picture claim is empty", "", map[string]any{"picture": "https://example.com/avatar.png"}, ""),
	Entry("returns URL when claim is present", "picture", map[string]any{"picture": "https://example.com/avatar.png"}, "https://example.com/avatar.png"),
	Entry("returns empty when claim is missing", "picture", map[string]any{"other": "value"}, ""),
	Entry("returns empty when claim value is empty", "picture", map[string]any{"picture": ""}, ""),
	Entry("supports custom claim name", "avatar_url", map[string]any{"avatar_url": "https://cdn.example.com/img.png"}, "https://cdn.example.com/img.png"),
)
