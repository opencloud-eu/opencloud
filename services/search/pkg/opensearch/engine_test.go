package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestEngine_Upsert(t *testing.T) {
	index := "test-engine-upsert"
	tc := ostest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, 0)

	engine, err := opensearch.NewEngine(index, tc.Client())
	assert.NoError(t, err)

	t.Run("Upsert with full document", func(t *testing.T) {
		document := ostest.Testdata.Resources.Full
		assert.NoError(t, engine.Upsert(document.ID, document))

		tc.Require.IndicesCount([]string{index}, 1)
		tc.Require.IndicesDelete([]string{index})
	})
}

func TestEngine_Move(t *testing.T) {}

func TestEngine_Delete(t *testing.T) {}

func TestEngine_Restore(t *testing.T) {}

func TestEngine_Purge(t *testing.T) {
	index := "test-engine-purge"
	tc := ostest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, 0)

	engine, err := opensearch.NewEngine(index, tc.Client())
	assert.NoError(t, err)

	t.Run("Purge with full document", func(t *testing.T) {
		document := ostest.Testdata.Resources.Full
		assert.NoError(t, engine.Upsert(document.ID, document))

		tc.Require.IndicesCount([]string{index}, 1)

		assert.NoError(t, engine.Purge(document.ID))

		tc.Require.IndicesCount([]string{index}, 0)

		tc.Require.IndicesDelete([]string{index})
	})
}

func TestEngine_DocCount(t *testing.T) {}
