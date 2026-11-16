package engine

import (
	"context"
	"encoding/json"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	v0 "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/policies/v0"
	"google.golang.org/protobuf/encoding/protojson"
)

// Engine defines the granted handlers.
type Engine interface {
	Evaluate(ctx context.Context, query string, env Environment) (bool, error)
}

type (
	// Stage defines the used auth stage
	Stage string
)

var (
	// StagePP defines the post-processing stage
	StagePP Stage = "pp"

	// StageHTTP defines the http stage
	StageHTTP Stage = "http"
)

// Resource contains resource information and is used as part of the evaluated environment.
type Resource struct {
	ID   provider.ResourceId `json:"resource_id"`
	Name string              `json:"name"`
	URL  string              `json:"url"`
	Size uint64              `json:"size"`
}

// Request contains request information and is used as part of the evaluated environment.
type Request struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	// ContentLength is the length of the request body, 0 if unknown.
	ContentLength uint64 `json:"content_length"`
	// UploadLength is the total size of an upload as declared by the TUS Upload-Length header, 0 if absent.
	UploadLength uint64 `json:"upload_length"`
}

// Environment contains every data that is needed to decide if the request should pass or not
type Environment struct {
	Stage    Stage     `json:"stage"`
	User     user.User `json:"user"`
	Request  Request   `json:"request"`
	Resource Resource  `json:"resource"`
}

// NewEnvironmentFromPB converts a PBEnvironment to Environment.
func NewEnvironmentFromPB(pEnv *v0.Environment) (Environment, error) {
	env := Environment{}

	rData, err := protojson.Marshal(pEnv)
	if err != nil {
		return env, err
	}

	if err := json.Unmarshal(rData, &env); err != nil {
		return env, err
	}

	// protojson encodes uint64 as strings with camelCase keys, so copy the lengths explicitly
	env.Request.ContentLength = pEnv.GetRequest().GetContentLength()
	env.Request.UploadLength = pEnv.GetRequest().GetUploadLength()

	switch pEnv.Stage {
	case v0.Stage_STAGE_HTTP:
		env.Stage = StageHTTP
	case v0.Stage_STAGE_PP:
		env.Stage = StagePP
	}

	return env, nil
}
