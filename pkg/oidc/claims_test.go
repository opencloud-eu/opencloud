package oidc_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/oidc"
)

type splitWithEscapingTest struct {
	// Name of the subtest.
	name string

	// string to split
	s string

	// seperator to use
	seperator string

	// escape character to use for escaping
	escape string

	expectedParts []string
}

func (swet splitWithEscapingTest) run(t *testing.T) {
	parts := oidc.SplitWithEscaping(swet.s, swet.seperator, swet.escape)
	if len(swet.expectedParts) != len(parts) {
		t.Errorf("mismatching length")
	}
	for i, v := range swet.expectedParts {
		if parts[i] != v {
			t.Errorf("expected part %d to be '%s', got '%s'", i, v, parts[i])
		}
	}
}

func TestSplitWithEscaping(t *testing.T) {
	tests := []splitWithEscapingTest{
		{
			name:          "plain claim name",
			s:             "roles",
			seperator:     ".",
			escape:        "\\",
			expectedParts: []string{"roles"},
		},
		{
			name:          "claim with .",
			s:             "my.roles",
			seperator:     ".",
			escape:        "\\",
			expectedParts: []string{"my", "roles"},
		},
		{
			name:          "claim with escaped .",
			s:             "my\\.roles",
			seperator:     ".",
			escape:        "\\",
			expectedParts: []string{"my.roles"},
		},
		{
			name:          "claim with escaped . left",
			s:             "my\\.other.roles",
			seperator:     ".",
			escape:        "\\",
			expectedParts: []string{"my.other", "roles"},
		},
		{
			name:          "claim with escaped . right",
			s:             "my.other\\.roles",
			seperator:     ".",
			escape:        "\\",
			expectedParts: []string{"my", "other.roles"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

type walkSegmentsTest struct {
	// Name of the subtest.
	name string

	// path segments to walk
	segments []string

	// seperator to use
	claims map[string]any

	expected any

	wantErr bool
}

func (wst walkSegmentsTest) run(t *testing.T) {
	v, err := oidc.WalkSegments(wst.segments, wst.claims)
	if err != nil && !wst.wantErr {
		t.Errorf("%v", err)
	}
	if err == nil && wst.wantErr {
		t.Errorf("expected error")
	}
	if !reflect.DeepEqual(v, wst.expected) {
		t.Errorf("expected %v got %v", wst.expected, v)
	}
}

func TestWalkSegments(t *testing.T) {
	byt := []byte(`{"first":{"second":{"third":["value1","value2"]},"foo":"bar"},"fizz":"buzz"}`)
	var dat map[string]any
	if err := json.Unmarshal(byt, &dat); err != nil {
		t.Errorf("%v", err)
	}

	tests := []walkSegmentsTest{
		{
			name:     "one segment, single value",
			segments: []string{"first"},
			claims: map[string]any{
				"first": "value",
			},
			expected: "value",
			wantErr:  false,
		},
		{
			name:     "one segment, array value",
			segments: []string{"first"},
			claims: map[string]any{
				"first": []string{"value1", "value2"},
			},
			expected: []string{"value1", "value2"},
			wantErr:  false,
		},
		{
			name:     "two segments, single value",
			segments: []string{"first", "second"},
			claims: map[string]any{
				"first": map[string]any{
					"second": "value",
				},
			},
			expected: "value",
			wantErr:  false,
		},
		{
			name:     "two segments, array value",
			segments: []string{"first", "second"},
			claims: map[string]any{
				"first": map[string]any{
					"second": []string{"value1", "value2"},
				},
			},
			expected: []string{"value1", "value2"},
			wantErr:  false,
		},
		{
			name:     "three segments, array value from json",
			segments: []string{"first", "second", "third"},
			claims:   dat,
			expected: []any{"value1", "value2"},
			wantErr:  false,
		},
		{
			name:     "three segments, array value with interface key",
			segments: []string{"first", "second", "third"},
			claims: map[string]any{
				"first": map[any]any{
					"second": map[any]any{
						"third": []string{"value1", "value2"},
					},
				},
			},
			expected: []string{"value1", "value2"},
			wantErr:  false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}
