// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"testing"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name          string
		jwtSecret     string
		sessionSecret string
		wantErr       bool
	}{
		{name: "distinct secrets", jwtSecret: "reva-secret", sessionSecret: "session-secret"},
		{name: "missing jwt secret", sessionSecret: "session-secret", wantErr: true},
		{name: "missing session secret", jwtSecret: "reva-secret", wantErr: true},
		{name: "shared secret", jwtSecret: "same-secret", sessionSecret: "same-secret", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				TokenManager: &config.TokenManager{JWTSecret: tt.jwtSecret},
				JWT:          config.JWT{Secret: tt.sessionSecret},
			}

			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
