package opa_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/config"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/engine"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/engine/opa"
)

func benchEngine(tb testing.TB) engine.Engine {
	tb.Helper()

	e, err := opa.NewOPA(10*time.Second, log.NopLogger(), config.Engine{Policies: []string{
		filepath.Join("testdata", "bench", "proxy.rego"),
		filepath.Join("testdata", "bench", "utils.rego"),
	}})
	if err != nil {
		tb.Fatal(err)
	}

	return e
}

func benchEnvironment(name string) engine.Environment {
	return engine.Environment{
		Stage:    engine.StageHTTP,
		Request:  engine.Request{Method: "PUT", Path: "/remote.php/dav/files/alice/" + name},
		Resource: engine.Resource{Name: name},
	}
}

func BenchmarkEvaluate(b *testing.B) {
	e := benchEngine(b)
	env := benchEnvironment("report.txt")
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		granted, err := e.Evaluate(ctx, "data.proxy.granted", env)
		if err != nil {
			b.Fatal(err)
		}
		if !granted {
			b.Fatal("expected .txt to be granted")
		}
	}
}

func BenchmarkEvaluateParallel(b *testing.B) {
	e := benchEngine(b)
	env := benchEnvironment("report.txt")
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := e.Evaluate(ctx, "data.proxy.granted", env); err != nil {
				b.Fatal(err)
			}
		}
	})
}
