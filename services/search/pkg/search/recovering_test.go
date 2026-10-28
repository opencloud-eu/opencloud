package search_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search/mocks"
)

func TestRecoveringEngineSearchRecoversPanic(t *testing.T) {
	engine := mocks.NewEngine(t)
	engine.EXPECT().Search(mock.Anything, mock.Anything).Run(func(context.Context, *searchService.SearchIndexRequest) {
		panic("assignment to entry in nil map")
	})

	res, err := search.NewRecoveringEngine(engine, log.NopLogger()).Search(context.Background(), &searchService.SearchIndexRequest{})
	if res != nil {
		t.Errorf("expected a nil result, got %v", res)
	}
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "search engine panic") {
		t.Errorf("expected the error to name the engine panic, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "assignment to entry in nil map") {
		t.Errorf("expected the error to carry the panic text, got %q", err.Error())
	}
}

func TestRecoveringEngineSearchLogsPanic(t *testing.T) {
	previousLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	t.Cleanup(func() { zerolog.SetGlobalLevel(previousLevel) })

	var buf bytes.Buffer
	logger := log.Logger{Logger: zerolog.New(&buf)}

	engine := mocks.NewEngine(t)
	engine.EXPECT().Search(mock.Anything, mock.Anything).Run(func(context.Context, *searchService.SearchIndexRequest) {
		panic("assignment to entry in nil map")
	})

	if _, err := search.NewRecoveringEngine(engine, logger).Search(context.Background(), &searchService.SearchIndexRequest{}); err == nil {
		t.Fatal("expected an error, got nil")
	}

	out := buf.String()
	if !strings.Contains(out, `"op":"Search"`) {
		t.Errorf("expected the op field, got %q", out)
	}
	if !strings.Contains(out, `"stack":"`) || strings.Contains(out, `"stack":""`) {
		t.Errorf("expected a non-empty stack field, got %q", out)
	}
}

func TestRecoveringEngineUpsertRecoversPanic(t *testing.T) {
	engine := mocks.NewEngine(t)
	engine.EXPECT().Upsert(mock.Anything, mock.Anything).Run(func(string, search.Resource) {
		panic("assignment to entry in nil map")
	})

	err := search.NewRecoveringEngine(engine, log.NopLogger()).Upsert("id", search.Resource{})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "search engine panic") {
		t.Errorf("expected the error to name the engine panic, got %q", err.Error())
	}
}

func TestRecoveringEnginePassThrough(t *testing.T) {
	sentinel := errors.New("index closed")

	engine := mocks.NewEngine(t)
	engine.EXPECT().DocCount().Return(uint64(42), nil).Once()
	engine.EXPECT().DocCount().Return(uint64(0), sentinel).Once()

	recovering := search.NewRecoveringEngine(engine, log.NopLogger())

	count, err := recovering.DocCount()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 42 {
		t.Errorf("expected 42, got %d", count)
	}

	if _, err := recovering.DocCount(); !errors.Is(err, sentinel) {
		t.Errorf("expected %v, got %v", sentinel, err)
	}
}

func TestRecoveringBatchPushRecoversPanic(t *testing.T) {
	batch := mocks.NewBatchOperator(t)
	batch.EXPECT().Push().Run(func() {
		panic("assignment to entry in nil map")
	})

	engine := mocks.NewEngine(t)
	engine.EXPECT().NewBatch(mock.Anything).Return(batch, nil)

	got, err := search.NewRecoveringEngine(engine, log.NopLogger()).NewBatch(10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = got.Push()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "search engine panic") {
		t.Errorf("expected the error to name the engine panic, got %q", err.Error())
	}
}

func TestRecoveringBatchPushPassThrough(t *testing.T) {
	sentinel := errors.New("index closed")

	batch := mocks.NewBatchOperator(t)
	batch.EXPECT().Push().Return(sentinel)

	engine := mocks.NewEngine(t)
	engine.EXPECT().NewBatch(mock.Anything).Return(batch, nil)

	got, err := search.NewRecoveringEngine(engine, log.NopLogger()).NewBatch(10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := got.Push(); !errors.Is(err, sentinel) {
		t.Errorf("expected %v, got %v", sentinel, err)
	}
}

func TestRecoveringBatchUpsertPassThrough(t *testing.T) {
	batch := mocks.NewBatchOperator(t)
	batch.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)

	engine := mocks.NewEngine(t)
	engine.EXPECT().NewBatch(mock.Anything).Return(batch, nil)

	got, err := search.NewRecoveringEngine(engine, log.NopLogger()).NewBatch(10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := got.Upsert("id", search.Resource{}); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
