package middleware

import (
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestReadProfilePictureURL(t *testing.T) {
	tests := []struct {
		name   string
		claims config.AutoProvisionClaims
		input  map[string]any
		want   string
	}{
		{
			name:   "disabled when picture claim is empty",
			claims: config.AutoProvisionClaims{Picture: ""},
			input:  map[string]any{"picture": "https://example.com/avatar.png"},
			want:   "",
		},
		{
			name:   "returns URL when claim is present",
			claims: config.AutoProvisionClaims{Picture: "picture"},
			input:  map[string]any{"picture": "https://example.com/avatar.png"},
			want:   "https://example.com/avatar.png",
		},
		{
			name:   "returns empty when claim is missing",
			claims: config.AutoProvisionClaims{Picture: "picture"},
			input:  map[string]any{"other": "value"},
			want:   "",
		},
		{
			name:   "returns empty when claim value is empty",
			claims: config.AutoProvisionClaims{Picture: "picture"},
			input:  map[string]any{"picture": ""},
			want:   "",
		},
		{
			name:   "supports custom claim name",
			claims: config.AutoProvisionClaims{Picture: "avatar_url"},
			input:  map[string]any{"avatar_url": "https://cdn.example.com/img.png"},
			want:   "https://cdn.example.com/img.png",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := accountResolver{
				logger:              log.NopLogger(),
				autoProvisionClaims: tt.claims,
			}
			got := m.readProfilePictureURL(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
