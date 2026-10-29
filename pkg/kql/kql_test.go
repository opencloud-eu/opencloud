package kql_test

import (
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
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
			expectedError: kql.DanglingOperatorError{
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

func TestBuild_ValidationErrors(t *testing.T) {
	assert := tAssert.New(t)

	for _, q := range []string{
		"NOT",
		"cat NOT",
		"cat AND",
		"AND cat",
		"cat AND OR dog",
		"cat NOT AND dog",
		"(cat",
		"cat)",
		"()",
		"cat) OR (dog",
		`"cat`,
		`cat"`,
		`ca"t`,
		`name:ca"t`,
		`name=ca"t`,
		`"""`,
		`name:"cat""`,
		`content:"foo\""`,
	} {
		got, err := kql.Builder{}.Build(q)
		assert.Nil(got, q)
		assert.Error(err, q)
		assert.True(kql.IsValidationError(err), "%q: %v", q, err)
	}
}

func TestBuild_ValidQueries(t *testing.T) {
	assert := tAssert.New(t)

	for _, q := range []string{
		"cat",
		"cat dog",
		"NOT cat",
		"NOT NOT cat",
		"cat NOT dog",
		"cat AND NOT dog",
		"(cat OR dog)",
		`name:"cat dog"`,
		"cat AND (dog OR fish)",
	} {
		got, err := kql.Builder{}.Build(q)
		assert.NoError(err, q)
		assert.NotNil(got, q)
	}
}
