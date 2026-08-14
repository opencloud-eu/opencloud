package opa

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/open-policy-agent/opa/loader"
	"github.com/open-policy-agent/opa/rego"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/config"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/engine"
)

// downloads go to the internal data gateway, its certificate is not verified.
const rHTTPInsecure = true

// OPA wraps open policy agent makes it possible to ask if an action is granted.
type OPA struct {
	timeout  time.Duration
	policies *loader.Result
	options  []func(r *rego.Rego)
}

// NewOPA returns a ready to use opa engine.
func NewOPA(timeout time.Duration, logger log.Logger, conf config.Engine) (*OPA, error) {
	var mtReader io.ReadCloser

	if conf.Mimes != "" {
		var err error
		mtReader, err = os.Open(conf.Mimes)
		if err != nil {
			return nil, err
		}

		defer func() {
			_ = mtReader.Close()
		}()
	}

	rfMimetypeExtensions, err := RFMimetypeExtensions(mtReader)
	if err != nil {
		return nil, err
	}

	policies, err := loader.NewFileLoader().WithProcessAnnotation(true).Filtered(conf.Policies, nil)
	if err != nil {
		return nil, err
	}

	options := []func(r *rego.Rego){
		rego.EnablePrintStatements(true),
		rego.PrintHook(logPrinter{logger: logger}),
		RFMimetypeDetect,
		RFResourceDownload(rhttp.GetHTTPClient(rhttp.Insecure(rHTTPInsecure))),
		rfMimetypeExtensions,
	}

	for _, module := range policies.ParsedModules() {
		options = append(options, rego.ParsedModule(module))
	}

	return &OPA{
		timeout:  timeout,
		policies: policies,
		options:  options,
	}, nil
}

// Evaluate evaluates the opa policies and returns the result.
func (o *OPA) Evaluate(ctx context.Context, qs string, env engine.Environment) (bool, error) {
	// note that we use the caller's context here because having a timeout is optional and up to the caller,
	// since this part only parses the rules, and the configured timeout
	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	store, err := o.policies.Store()
	if err != nil {
		return false, err
	}

	q, err := rego.New(
		append([]func(r *rego.Rego){
			rego.Query(qs),
			rego.Store(store),
		}, o.options...)...,
	).PrepareForEval(ctx)
	if err != nil {
		return false, err
	}

	result, err := q.Eval(ctx, rego.EvalInput(env))
	if err != nil {
		return false, err
	}

	return result.Allowed(), nil
}
