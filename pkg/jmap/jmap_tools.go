package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

func command[T any](api ApiClient,
	logger *log.Logger,
	ctx context.Context,
	session *Session,
	request Request,
	mapper func(body *Response) (T, error)) (T, error) {

	responseBody, err := api.Command(ctx, logger, session, request)
	if err != nil {
		var zero T
		return zero, err
	}

	var data Response
	err = json.Unmarshal(responseBody, &data)
	if err != nil {
		logger.Error().Err(err).Msg("failed to deserialize body JSON payload")
		var zero T
		return zero, err
	}

	return mapper(&data)
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

func retrieveResponseMatchParameters[T any](data *Response, command Command, tag string, target *T) error {
	match, ok := retrieveResponseMatch(data, command, tag)
	if !ok {
		return fmt.Errorf("failed to find JMAP response invocation match for command '%v' and tag '%v'", command, tag)
	}
	params := match.Parameters
	typedParams, ok := params.(T)
	if !ok {
		actualType := reflect.TypeOf(params)
		expectedType := reflect.TypeOf(*target)
		return fmt.Errorf("JMAP response invocation matches command '%v' and tag '%v' but the type %v does not match the expected %v", command, tag, actualType, expectedType)
	}
	*target = typedParams
	return nil
}

func (e *EmailBodyStructure) UnmarshalJSON(bs []byte) error {
	m := map[string]any{}
	err := json.Unmarshal(bs, &m)
	if err != nil {
		return err
	}
	return decodeMap(m, e)
}

func (e *EmailBodyStructure) MarshalJSON() ([]byte, error) {
	m := map[string]any{}
	m["type"] = e.Type
	m["partId"] = e.PartId
	for k, v := range e.Other {
		m[k] = v
	}
	return json.Marshal(m)
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
