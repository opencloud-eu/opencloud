package version_test

import (
	"fmt"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/version"
)

func TestChannel(t *testing.T) {
	tests := map[string]struct {
		got   string
		valid bool
	}{
		"no channel, defaults to dev": {
			got:   "",
			valid: false,
		},
		"dev channel": {
			got:   version.EditionDev,
			valid: true,
		},
		"rolling channel": {
			got:   version.EditionRolling,
			valid: true,
		},
		"stable channel": {
			got:   version.EditionStable,
			valid: true,
		},
		"lts channel without version": {
			got:   version.EditionLTS,
			valid: false,
		},
		"lts-1.0.0 channel": {
			got:   fmt.Sprintf("%s-1", version.EditionLTS),
			valid: true,
		},
		"lts-one invalid version": {
			got:   fmt.Sprintf("%s-one", version.EditionLTS),
			valid: false,
		},
		"known channel with version": {
			got:   fmt.Sprintf("%s-1", version.EditionStable),
			valid: false,
		},
		"unknown channel": {
			got:   "foo",
			valid: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			version.Edition = test.got

			switch err := version.InitEdition(); {
			case err != nil && !test.valid && version.Edition != version.Dev: // if a given edition is unknown, the value is always dev
				fallthrough
			case test.valid != (err == nil):
				t.Fatalf("invalid edition: %s", version.Edition)
			}
		})
	}
}
