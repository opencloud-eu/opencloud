package jmap

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/opencloud/pkg/jscalendar"
)

func single[S any](s S) []S {
	return []S{s}
}

var UintPtrOne *uint = uintPtr(1)
var UintPtrZero *uint = uintPtr(0)

type eventListeners[T any] struct {
	listeners []T
	m         sync.Mutex // TODO get rid of the mutex to avoid lock contention, use a lock-free alternative, or get rid of it altogether
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
func mcid(accountId AccountId, tag string) string {
	// https://jmap.io/spec-core.html#the-invocation-data-type
	// May be any string of data:
	// An arbitrary string from the client to be echoed back with the responses emitted by that method
	// call (a method may return 1 or more responses, as it may make implicit calls to other methods;
	// all responses initiated by this method call get the same method call id in the response).
	return string(accountId) + ":" + tag
}

type Cmdr interface {
	ApiSupplier
	Hooks
}

type Operation string

func command[T any](client Cmdr, //NOSONAR
	operation Operation,
	ctx Context,
	request Request,
	mapper func(body *Response) (T, State, Error)) (Result[T], Error) {

	logger := ctx.Logger

	before := time.Now()
	responseBody, language, jmapErr := client.Api().Command(operation, request, ctx)
	duration := time.Since(before)
	if jmapErr != nil {
		return ZeroResult[T](single(duration)), jmapErr
	}
	if responseBody == nil {
		return ZeroResult[T](single(duration)), jmapError(fmt.Errorf("empty response body"), JmapErrorInvalidJmapResponsePayload)
	}

	var response Response
	if err := json.NewDecoder(responseBody).Decode(&response); err != nil {
		logger.Error().Err(err).Msgf("failed to deserialize body JSON payload into a %T", response)
		if err := responseBody.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close response body") //NOSONAR
		}
		return ZeroResult[T](single(duration)), jmapError(err, JmapErrorDecodingResponseBody)
	}

	if err := responseBody.Close(); err != nil {
		logger.Error().Err(err).Msg("failed to close response body") //NOSONAR
	}

	if response.SessionState != ctx.Session.State {
		client.OnSessionOutdated(ctx.Session, response.SessionState)
	}

	// search for an "error" response
	// https://jmap.io/spec-core.html#method-level-errors
	for _, mr := range response.MethodResponses {
		if mr.Command == ErrorCommand {
			if errorParameters, ok := mr.Parameters.(ErrorResponse); ok {
				// TODO deal with stateMismatch differently, as it's not an error per se, but rather "optimistic update"
				code := JmapErrorServerFail
				switch errorParameters.Type {
				case MethodLevelErrorServerUnavailable:
					code = JmapErrorServerUnavailable
				case MethodLevelErrorServerFail, MethodLevelErrorServerPartialFail:
					code = JmapErrorServerFail
				case MethodLevelErrorUnknownMethod:
					code = JmapErrorUnknownMethod
				case MethodLevelErrorInvalidArguments:
					code = JmapErrorInvalidArguments
					if strings.HasPrefix(errorParameters.Description, "invalid JMAP State") {
						code = JmapErrorInvalidObjectState
					}
				case MethodLevelErrorInvalidResultReference:
					code = JmapErrorInvalidResultReference
				case MethodLevelErrorForbidden:
					// there's a quirk here: when referencing an account that exists but that this
					// user has no access to, Stalwart returns the 'forbidden' error, but this might
					// leak the existence of an account to an attacker -- instead, we deem it safer to
					// return a "account does not exist" error instead
					if strings.HasPrefix(errorParameters.Description, "You do not have access to account") {
						code = JmapErrorAccountNotFound
					} else {
						code = JmapErrorForbidden
					}
				case MethodLevelErrorAccountNotFound:
					code = JmapErrorAccountNotFound
				case MethodLevelErrorAccountNotSupportedByMethod:
					code = JmapErrorAccountNotSupportedByMethod
				case MethodLevelErrorAccountReadOnly:
					code = JmapErrorAccountReadOnly
				}
				msg := fmt.Sprintf("found method level error in response '%v', type: '%v', description: '%v'", mr.Tag, errorParameters.Type, errorParameters.Description)
				err := errors.New(msg)
				logger.Warn().Int("code", code).Str("type", errorParameters.Type).Msg(msg)
				return newPartialResult[T](response.SessionState, language, single(duration)), jmapResponseError(code, err, errorParameters.Type, errorParameters.Description)
			} else {
				code := JmapErrorUnspecifiedType
				msg := fmt.Sprintf("found method level error in response '%v'", mr.Tag)
				err := errors.New(msg)
				logger.Warn().Int("code", code).Msg(msg)
				return newPartialResult[T](response.SessionState, language, single(duration)), jmapResponseError(code, err, errorParameters.Type, errorParameters.Description)
			}
		}
	}

	result, state, jerr := mapper(&response)
	sessionState := response.SessionState
	return NewResult(result, sessionState, state, language, single(duration)), jerr
}

func mapstructStringToTimeHook() mapstructure.DecodeHookFunc {
	// mapstruct isn't able to properly map RFC3339 date strings into Time
	// objects, which is why we require this custom hook,
	// see https://github.com/mitchellh/mapstructure/issues/41
	wanted := reflect.TypeFor[time.Time]()
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if to != wanted {
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
		Metadata: nil,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructStringToTimeHook(),
			jscalendar.MapstructTriggerHook(),
		),
		Result:               &target,
		ErrorUnused:          false,
		ErrorUnset:           false,
		IgnoreUntaggedFields: false,
		Squash:               true,
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

func retrieveResponseMatchParameters[T any](ctx Context, data *Response, command Command, tag string, target *T) Error {
	match, ok := retrieveResponseMatch(data, command, tag)
	if !ok {
		err := fmt.Errorf("failed to find JMAP response invocation match for command '%v' and tag '%v'", command, tag) // NOSONAR
		ctx.Logger.Error().Msg(err.Error())
		return jmapError(err, JmapErrorInvalidJmapResponsePayload)
	}
	params := match.Parameters
	typedParams, ok := params.(T)
	if !ok {
		err := fmt.Errorf("JMAP response invocation matches command '%v' and tag '%v' but the type %T does not match the expected %T", command, tag, params, *target) // NOSONAR
		ctx.Logger.Error().Msg(err.Error())
		return jmapError(err, JmapErrorInvalidJmapResponsePayload)
	}
	*target = typedParams
	return nil
}

func tryRetrieveResponseMatchParameters[T any](ctx Context, data *Response, command Command, tag string, target *T) (bool, Error) {
	match, ok := retrieveResponseMatch(data, command, tag)
	if !ok {
		return false, nil
	}
	params := match.Parameters
	typedParams, ok := params.(T)
	if !ok {
		err := fmt.Errorf("JMAP response invocation matches command '%v' and tag '%v' but the type %T does not match the expected %T", command, tag, params, *target)
		ctx.Logger.Error().Msg(err.Error())
		return true, jmapError(err, JmapErrorInvalidJmapResponsePayload)
	}
	*target = typedParams
	return true, nil
}

func retrieveGet[T Foo, C GetCommand[T], R GetResponse[T]](ctx Context, data *Response, command C, tag string, target *R) Error {
	return retrieveResponseMatchParameters(ctx, data, command.GetCommand(), tag, target)
}

func retrieveSet[T Foo, C SetCommand[T], R SetResponse[T]](ctx Context, data *Response, command C, tag string, target *R) Error {
	return retrieveResponseMatchParameters(ctx, data, command.GetCommand(), tag, target)
}

// TODO enable self-referencing generics parameter when upgrading to Go 1.26
func retrieveQuery[T Foo, C QueryCommand[T /*, C*/], R QueryResponse[T]](ctx Context, data *Response, command C, tag string, target *R) Error {
	return retrieveResponseMatchParameters(ctx, data, command.GetCommand(), tag, target)
}

func retrieveChanges[T Foo, C ChangesCommand[T], R ChangesResponse[T]](ctx Context, data *Response, command C, tag string, target *R) Error {
	return retrieveResponseMatchParameters(ctx, data, command.GetCommand(), tag, target)
}

func retrieveUpload[T Foo, C UploadCommand[T], R UploadResponse[T]](ctx Context, data *Response, command C, tag string, target *R) Error {
	return retrieveResponseMatchParameters(ctx, data, command.GetCommand(), tag, target)
}

func retrieveParse[T Foo, C ParseCommand[T], R ParseResponse[T]](ctx Context, data *Response, command C, tag string, target *R) Error {
	return retrieveResponseMatchParameters(ctx, data, command.GetCommand(), tag, target)
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

func squashState[K ~string](all map[K]State) State {
	return squashStateFunc(all, func(s State) State { return s })
}

/*
func squashStates(states ...State) State {
	return State(strings.Join(structs.Map(states, func(s State) string { return string(s) }), ","))
}
*/

func squashKeyedStates[K ~string](m map[K]State) State {
	return squashStateFunc(m, identity1)
}

func squashStateFunc[K ~string, V any](all map[K]V, mapper func(V) State) State {
	n := len(all)
	if n == 0 {
		return State("")
	}
	if n == 1 {
		for _, v := range all {
			return mapper(v)
		}
	}

	parts := make([]string, n)
	sortedKeys := make([]K, n)
	i := 0
	for k := range all {
		sortedKeys[i] = k
		i++
	}
	slices.Sort(sortedKeys)
	for i, k := range sortedKeys {
		if v, ok := all[k]; ok {
			parts[i] = string(k) + ":" + string(mapper(v))
		} else {
			parts[i] = string(k) + ":"
		}
	}
	return State(strings.Join(parts, ","))
}

func squashStateMaps(first map[AccountId]State, second map[AccountId]State) State {
	return squashStateFunc(mapPairs(first, second), func(p pair[State, State]) State {
		if p.left != nil {
			if p.right != nil {
				return *p.left + ";" + *p.right
			} else {
				return *p.left + ";"
			}
		} else if p.right != nil {
			return ";" + *p.right
		} else {
			return ";"
		}
	})
}

type pair[L any, R any] struct {
	left  *L
	right *R
}

func mapPairs[K comparable, L, R any](left map[K]L, right map[K]R) map[K]pair[L, R] {
	result := map[K]pair[L, R]{}
	for k, l := range left {
		if r, ok := right[k]; ok {
			result[k] = pair[L, R]{left: &l, right: &r}
		} else {
			result[k] = pair[L, R]{left: &l, right: nil}
		}
	}
	for k, r := range right {
		if _, ok := left[k]; !ok {
			result[k] = pair[L, R]{left: nil, right: &r}
		}
	}
	return result
}

var (
	truep = ptr(true)
)

func identity1[T any](t T) T {
	return t
}

func uintPtr[T int | uint](i T) *uint {
	return ptr(uint(i))
}

func valueIf[T any | uint | int | bool | any](value *T, condition bool) *T {
	if condition {
		return value
	} else {
		return nil
	}
}

func ptrIf[T any | uint | int | bool](value T, condition bool) *T {
	if condition {
		return &value
	} else {
		return nil
	}
}

func ns(namespaces ...JmapNamespace) []JmapNamespace {
	result := make([]JmapNamespace, len(namespaces)+1)
	result[0] = JmapCore
	for i, n := range namespaces {
		result[i+1] = n
	}
	return result
}

// TODO remove and replace with calls to new() when upgrading to Go 1.26
func ptr[T any | int | uint | bool | string](t T) *T {
	return &t
}

func dumpHttpRequest(req *http.Request, maxBodySize int64, closure func(method string, uri string, content string, truncated bool)) error {
	var logBuilder bytes.Buffer
	uri := req.URL.String()
	fmt.Fprintf(&logBuilder, "%s %s\n", req.Method, uri)
	for key, values := range req.Header {
		if key == "Authorization" {
			continue
		}
		for _, value := range values {
			fmt.Fprintf(&logBuilder, "%s: %s\n", key, value)
		}
	}
	truncated := false
	if maxBodySize >= 0 && req.Body != nil && req.Body != http.NoBody {
		peekBuffer := new(bytes.Buffer)
		limitedReader := io.LimitReader(req.Body, maxBodySize)
		tee := io.TeeReader(limitedReader, peekBuffer)
		if _, err := io.Copy(io.Discard, tee); err != nil {
			return fmt.Errorf("failed to peek at the request body for tracing: %v", err)
		} else {
			logBuilder.Write(peekBuffer.Bytes())
			if int64(peekBuffer.Len()) >= maxBodySize {
				truncated = true
			}
			fullBodyReader := io.MultiReader(peekBuffer, req.Body)
			req.Body = io.NopCloser(fullBodyReader)
		}
	}
	closure(req.Method, uri, logBuilder.String(), truncated)
	return nil
}

func dumpHttpResponse(req *http.Request, resp *http.Response, maxBodySize int64,
	closure func(method string, uri string, content string, truncated bool),
	closeErrorClosure func(err error)) (io.ReadCloser, error) {
	var logBuilder bytes.Buffer
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Fprintf(&logBuilder, "%s: %s\n", key, value)
		}
	}

	var peekBuffer *bytes.Buffer = nil
	truncated := false
	if maxBodySize > 0 {
		peekBuffer = new(bytes.Buffer)
		limitedReader := io.LimitReader(resp.Body, maxBodySize)
		tee := io.TeeReader(limitedReader, peekBuffer)
		if _, err := io.Copy(io.Discard, tee); err != nil {
			if resp.Body != nil {
				if err := resp.Body.Close(); err != nil {
					closeErrorClosure(err)
				}
			}
			return nil, err
		}

		if int64(peekBuffer.Len()) >= maxBodySize {
			truncated = true
		}

		logBuilder.Write(peekBuffer.Bytes())
	}

	if req != nil {
		closure(req.Method, req.URL.String(), logBuilder.String(), truncated)
	} else {
		closure("", "", logBuilder.String(), truncated)
	}

	if peekBuffer != nil {
		return httpResponseReadCloser{
			body:   resp.Body,
			Reader: io.MultiReader(peekBuffer, resp.Body),
		}, nil
	} else {
		return resp.Body, nil
	}
}
