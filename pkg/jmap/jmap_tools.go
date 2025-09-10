package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

type eventListeners[T any] struct {
	listeners []T
	m         sync.Mutex
}

func (e *eventListeners[T]) add(listener T) {
	e.m.Lock()
	defer e.m.Unlock()
	e.listeners = append(e.listeners, listener)
}

func (e *eventListeners[T]) signal(signal func(T)) {
	e.m.Lock()
	defer e.m.Unlock()
	for _, listener := range e.listeners {
		signal(listener)
	}
}

func newEventListeners[T any]() *eventListeners[T] {
	return &eventListeners[T]{
		listeners: []T{},
	}
}

// Create an identifier to use as a method call ID, from the specified accountId and additional
// tag, to make something unique within that API request.
func mcid(accountId string, tag string) string {
	// https://jmap.io/spec-core.html#the-invocation-data-type
	// May be any string of data:
	// An arbitrary string from the client to be echoed back with the responses emitted by that method
	// call (a method may return 1 or more responses, as it may make implicit calls to other methods;
	// all responses initiated by this method call get the same method call id in the response).
	return accountId + ":" + tag
}

func command[T any](api ApiClient,
	logger *log.Logger,
	ctx context.Context,
	session *Session,
	sessionOutdatedHandler func(session *Session, newState SessionState),
	request Request,
	mapper func(body *Response) (T, Error)) (T, SessionState, Error) {

	responseBody, jmapErr := api.Command(ctx, logger, session, request)
	if jmapErr != nil {
		var zero T
		return zero, "", jmapErr
	}

	var response Response
	err := json.Unmarshal(responseBody, &response)
	if err != nil {
		logger.Error().Err(err).Msg("failed to deserialize body JSON payload")
		var zero T
		return zero, "", SimpleError{code: JmapErrorDecodingResponseBody, err: err}
	}

	if response.SessionState != session.State {
		if sessionOutdatedHandler != nil {
			sessionOutdatedHandler(session, response.SessionState)
		}
	}

	// search for an "error" response
	// https://jmap.io/spec-core.html#method-level-errors
	for _, mr := range response.MethodResponses {
		if mr.Command == "error" {
			err := fmt.Errorf("found method level error in response '%v'", mr.Tag)
			if payload, ok := mr.Parameters.(map[string]any); ok {
				if errorType, ok := payload["type"]; ok {
					err = fmt.Errorf("found method level error in response '%v', type: '%v'", mr.Tag, errorType)
				}
			}
			var zero T
			return zero, response.SessionState, SimpleError{code: JmapErrorMethodLevel, err: err}
		}
	}

	result, jerr := mapper(&response)
	sessionState := response.SessionState
	return result, sessionState, jerr
}

func mapstructStringToTimeHook() mapstructure.DecodeHookFunc {
	// mapstruct isn't able to properly map RFC3339 date strings into Time
	// objects, which is why we require this custom hook,
	// see https://github.com/mitchellh/mapstructure/issues/41
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if to != reflect.TypeOf(time.Time{}) {
			return data, nil
		}
		switch from.Kind() {
		case reflect.String:
			return time.Parse(time.RFC3339, data.(string))
		case reflect.Float64:
			return time.Unix(0, int64(data.(float64))*int64(time.Millisecond)), nil
		case reflect.Int64:
			return time.Unix(0, data.(int64)*int64(time.Millisecond)), nil
		default:
			return data, nil
		}
	}
}

func decodeMap(input map[string]any, target any) error {
	// https://github.com/mitchellh/mapstructure/issues/41
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata:             nil,
		DecodeHook:           mapstructure.ComposeDecodeHookFunc(mapstructStringToTimeHook()),
		Result:               &target,
		ErrorUnused:          false,
		ErrorUnset:           false,
		IgnoreUntaggedFields: false,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

func decodeParameters(input any, target any) error {
	m, ok := input.(map[string]any)
	if !ok {
		return fmt.Errorf("decodeParameters: parameters is not a map but a %T", input)
	}
	return decodeMap(m, target)
}

func retrieveResponseMatch(data *Response, command Command, tag string) (Invocation, bool) {
	for _, inv := range data.MethodResponses {
		if command == inv.Command && tag == inv.Tag {
			return inv, true
		}
	}
	return Invocation{}, false
}

func retrieveResponseMatchParameters[T any](logger *log.Logger, data *Response, command Command, tag string, target *T) Error {
	match, ok := retrieveResponseMatch(data, command, tag)
	if !ok {
		err := fmt.Errorf("failed to find JMAP response invocation match for command '%v' and tag '%v'", command, tag)
		logger.Error().Msg(err.Error())
		return simpleError(err, JmapErrorInvalidJmapResponsePayload)
	}
	params := match.Parameters
	typedParams, ok := params.(T)
	if !ok {
		err := fmt.Errorf("JMAP response invocation matches command '%v' and tag '%v' but the type %T does not match the expected %T", command, tag, params, *target)
		logger.Error().Msg(err.Error())
		return simpleError(err, JmapErrorInvalidJmapResponsePayload)
	}
	*target = typedParams
	return nil
}

func (e EmailBodyStructure) MarshalJSON() ([]byte, error) {
	m := map[string]any{}
	maps.Copy(m, e.Other) // do this first to avoid overwriting type and partId
	m["type"] = e.Type
	m["partId"] = e.PartId
	return json.Marshal(m)
}

func (e *EmailBodyStructure) UnmarshalJSON(bs []byte) error {
	m := map[string]any{}
	err := json.Unmarshal(bs, &m)
	if err != nil {
		return err
	}
	return decodeMap(m, e)
}

func (i *Invocation) MarshalJSON() ([]byte, error) {
	// JMAP requests have a slightly unusual structure since they are not a JSON object
	// but, instead, a three-element array composed of
	// 0: the command (e.g. "Email/query")
	// 1: the actual payload of the request (structure depends on the command)
	// 2: a tag that can be used to identify the matching response payload
	// That implementation aspect thus requires us to use a custom marshalling hook.
	arr := []any{string(i.Command), i.Parameters, i.Tag}
	return json.Marshal(arr)
}

func (i *Invocation) UnmarshalJSON(bs []byte) error {
	// JMAP responses have a slightly unusual structure since they are not a JSON object
	// but, instead, a three-element array composed of
	// 0: the command (e.g. "Thread/get") this is a response to
	// 1: the actual payload of the response (structure depends on the command)
	// 2: the tag (same as in the request invocation)
	// That implementation aspect thus requires us to use a custom unmarshalling hook.
	arr := []any{}
	err := json.Unmarshal(bs, &arr)
	if err != nil {
		return err
	}
	if len(arr) != 3 {
		// JMAP response must really always be an array of three elements
		return fmt.Errorf("Invocation array length ought to be 3 but is %d", len(arr))
	}
	// The first element in the array is the command:
	i.Command = Command(arr[0].(string))
	// The third element in the array is the tag:
	i.Tag = arr[2].(string)

	// Due to the dynamic nature of request and response types in JMAP, we
	// switch to using mapstruct here to deserialize the payload in the "parameters"
	// element of JMAP invocation response arrays, as their expected struct type
	// is directly inferred from the command (e.g. "Mailbox/get")
	payload := arr[1]

	paramsFactory, ok := CommandResponseTypeMap[i.Command]
	if !ok {
		return fmt.Errorf("unsupported JMAP operation cannot be unmarshalled: %v", i.Command)
	}
	params := paramsFactory()
	err = decodeParameters(payload, &params)
	if err != nil {
		return err
	}
	i.Parameters = params
	return nil
}
