package middleware

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

var _ = Describe("reading the profile picture claim", func() {
	tests := []struct {
		name  string
		claim string
		input map[string]any
		want  string
	}{
		{
			name:  "disabled when picture claim is empty",
			claim: "",
			input: map[string]any{"picture": "https://example.com/avatar.png"},
			want:  "",
		},
		{
			name:  "returns URL when claim is present",
			claim: "picture",
			input: map[string]any{"picture": "https://example.com/avatar.png"},
			want:  "https://example.com/avatar.png",
		},
		{
			name:  "returns empty when claim is missing",
			claim: "picture",
			input: map[string]any{"other": "value"},
			want:  "",
		},
		{
			name:  "returns empty when claim value is empty",
			claim: "picture",
			input: map[string]any{"picture": ""},
			want:  "",
		},
		{
			name:  "supports custom claim name",
			claim: "avatar_url",
			input: map[string]any{"avatar_url": "https://cdn.example.com/img.png"},
			want:  "https://cdn.example.com/img.png",
		},
	}
	for _, tt := range tests {
		It(tt.name, func() {
			m := accountResolver{
				logger:       log.NopLogger(),
				pictureClaim: tt.claim,
			}
			got := m.readProfilePictureURL(tt.input)
			Expect(got).To(Equal(tt.want))
		})
	}
})
