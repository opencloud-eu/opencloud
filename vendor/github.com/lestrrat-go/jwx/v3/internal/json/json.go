package json

import (
	"bytes"
	"fmt"

	"sync/atomic"

	"github.com/lestrrat-go/jwx/v3/internal/base64"
	"github.com/lestrrat-go/jwx/v3/internal/tokens"
)

var useNumber atomic.Uint32

func UseNumber() bool {
	return useNumber.Load() == 1
}

// Sets the global configuration for json decoding
func DecoderSettings(inUseNumber bool) {
	var val uint32
	if inUseNumber {
		val = 1
	}
	useNumber.Store(val)
}

// Unmarshal respects the values specified in DecoderSettings,
// and uses a Decoder that has certain features turned on/off
func Unmarshal(b []byte, v any) error {
	dec := NewDecoder(bytes.NewReader(b))
	return dec.Decode(v)
}

func AssignNextBytesToken(dst *[]byte, dec *Decoder) error {
	var val string
	if err := dec.Decode(&val); err != nil {
		return fmt.Errorf(`error reading next value: %w`, err)
	}

	buf, err := base64.DecodeString(val)
	if err != nil {
		return fmt.Errorf(`expected base64 encoded []byte (%T)`, val)
	}
	*dst = buf
	return nil
}

func shouldRejectNullStrings(dc DecodeCtx) bool {
	if dc != nil {
		if sdc, ok := dc.(StrictStringDecodeCtx); ok {
			return sdc.StrictStrings()
		}
	}
	return false
}

// ReadNextStringToken reads the next JSON token from the decoder and
// returns it as a string. By default, JSON null is silently accepted as "".
// When the given DecodeCtx implements StrictStringDecodeCtx and StrictStrings()
// returns true, null values are rejected.
func ReadNextStringToken(dec *Decoder, dc DecodeCtx) (string, error) {
	if shouldRejectNullStrings(dc) {
		var val any
		if err := dec.Decode(&val); err != nil {
			return "", fmt.Errorf(`error reading next value: %w`, err)
		}
		if val == nil {
			return "", fmt.Errorf(`error reading next value: expected string, got null`)
		}
		s, ok := val.(string)
		if !ok {
			return "", fmt.Errorf(`error reading next value: expected string, got %T`, val)
		}
		return s, nil
	}

	var val string
	if err := dec.Decode(&val); err != nil {
		return "", fmt.Errorf(`error reading next value: %w`, err)
	}
	return val, nil
}

func AssignNextStringToken(dst **string, dec *Decoder, dc DecodeCtx) error {
	val, err := ReadNextStringToken(dec, dc)
	if err != nil {
		return err
	}
	*dst = &val
	return nil
}

// FlattenAudience is a flag to specify if we should flatten the "aud"
// entry to a string when there's only one entry.
// In jwx < 1.1.8 we just dumped everything as an array of strings,
// but apparently AWS Cognito doesn't handle this well.
//
// So now we have the ability to dump "aud" as a string if there's
// only one entry, but we need to retain the old behavior so that
// we don't accidentally break somebody else's code. (e.g. messing
// up how signatures are calculated)
var FlattenAudience uint32

func MarshalAudience(aud []string, flatten bool) ([]byte, error) {
	var val any
	if len(aud) == 1 && flatten {
		val = aud[0]
	} else {
		val = aud
	}
	return Marshal(val)
}

func EncodeAudience(enc *Encoder, aud []string, flatten bool) error {
	var val any
	if len(aud) == 1 && flatten {
		val = aud[0]
	} else {
		val = aud
	}
	return enc.Encode(val)
}

// DecodeCtx is an interface for objects that needs that extra something
// when decoding JSON into an object.
type DecodeCtx interface {
	Registry() *Registry
}

// DecodeCtxContainer is used to differentiate objects that can carry extra
// decoding hints and those who can't.
type DecodeCtxContainer interface {
	DecodeCtx() DecodeCtx
	SetDecodeCtx(DecodeCtx)
}

// StrictStringDecodeCtx is an optional interface that DecodeCtx implementations
// can satisfy to control per-call null string rejection.
type StrictStringDecodeCtx interface {
	StrictStrings() bool
}

// stock decodeCtx. should cover 80% of the cases
type decodeCtx struct {
	registry      *Registry
	strictStrings bool
}

// NewDecodeCtx creates a new DecodeCtx with the given registry.
func NewDecodeCtx(r *Registry) DecodeCtx {
	return &decodeCtx{registry: r}
}

// NewDecodeCtxStrictStrings creates a new DecodeCtx with the given registry
// and strict string rejection flag.
func NewDecodeCtxStrictStrings(r *Registry, strict bool) DecodeCtx {
	return &decodeCtx{registry: r, strictStrings: strict}
}

func (dc *decodeCtx) Registry() *Registry {
	return dc.registry
}

func (dc *decodeCtx) StrictStrings() bool {
	return dc.strictStrings
}

// WriteQuotedKey writes key as a quoted JSON object member name followed by
// the separating colon and a space.
//
// Member names come from public methods such as Set and Builder.Claim, so
// they may contain any byte, including `"`. A name copied raw between the
// quotes could end its own member and start further ones, so the serialized
// object would no longer match the one the caller built
// (GHSA-4cf7-xm37-g63h). A name that needs no escaping is written directly,
// which keeps the common path free of allocations. Every other name goes
// through the JSON string encoder.
func WriteQuotedKey(buf *bytes.Buffer, key string) error {
	if tokens.IsJSONSafeASCII(key) {
		buf.WriteByte(tokens.DoubleQuote)
		buf.WriteString(key)
		buf.WriteString(`": `)
		return nil
	}

	encoded, err := Marshal(key)
	if err != nil {
		return fmt.Errorf(`failed to encode object member name: %w`, err)
	}
	buf.Write(encoded)
	buf.WriteByte(tokens.Colon)
	buf.WriteByte(' ')
	return nil
}
