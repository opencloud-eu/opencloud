package kql_test

import (
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query"
	tAssert "github.com/stretchr/testify/assert"
)

func TestNewAST(t *testing.T) {
	tests := []struct {
		name          string
		givenQuery    string
		expectedError error
	}{
		{
			name:       "success",
			givenQuery: "foo:bar",
		},
		{
			name:       "error",
			givenQuery: kql.BoolAND,
			expectedError: query.StartsWithBinaryOperatorError{
				Node: &ast.OperatorNode{Value: kql.BoolAND},
			},
		},
	}

	assert := tAssert.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := kql.Builder{}.Build(tt.givenQuery)

			if tt.expectedError != nil {
				if tt.expectedError.Error() != "" {
					assert.Equal(err.Error(), tt.expectedError.Error())
				} else {
					assert.NotNil(err)
				}

				assert.Nil(got)

				return
			}

			assert.Nil(err)
			assert.NotNil(got)
		})
	}
}
