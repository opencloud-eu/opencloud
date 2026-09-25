package jwk

import (
	"bytes"
	"fmt"
	"maps"
	"reflect"
	"sort"

	"github.com/lestrrat-go/blackmagic"
	"github.com/lestrrat-go/jwx/v3/internal/json"
	"github.com/lestrrat-go/jwx/v3/internal/pool"
	"github.com/lestrrat-go/jwx/v3/internal/tokens"
)

const keysKey = `keys` // appease linter

func newSet() *set {
	return &set{
		privateParams: make(map[string]any),
	}
}

// NewSet creates and empty `jwk.Set` object
func NewSet() Set {
	return newSet()
}

func (s *set) Set(n string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n == keysKey {
		vl, ok := v.([]Key)
		if !ok {
			return fmt.Errorf(`value for field "keys" must be []jwk.Key`)
		}
		s.keys = vl
		return nil
	}

	s.privateParams[n] = v
	return nil
}

func (s *set) Get(name string, dst any) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.privateParams[name]
	if !ok {
		return fmt.Errorf(`field %q not found`, name)
	}
	if err := blackmagic.AssignIfCompatible(dst, v); err != nil {
		return fmt.Errorf(`failed to assign value to dst: %w`, err)
	}
	return nil
}

func (s *set) Key(idx int) (Key, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if idx >= 0 && idx < len(s.keys) {
		return s.keys[idx], true
	}
	return nil, false
}

func (s *set) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.keys)
}

// indexNL is Index(), but without the locking
func (s *set) indexNL(key Key) int {
	for i, k := range s.keys {
		if k == key {
			return i
		}
	}
	return -1
}

func (s *set) Index(key Key) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.indexNL(key)
}

func (s *set) AddKey(key Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rv := reflect.ValueOf(key)
	if !rv.IsValid() {
		return fmt.Errorf(`(jwk.Set).AddKey: nil key`)
	}
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Chan, reflect.Func, reflect.Map, reflect.Slice:
		if rv.IsNil() {
			return fmt.Errorf(`(jwk.Set).AddKey: nil key`)
		}
	}

	if i := s.indexNL(key); i > -1 {
		return fmt.Errorf(`(jwk.Set).AddKey: key already exists`)
	}
	s.keys = append(s.keys, key)
	return nil
}

func (s *set) Remove(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.privateParams, name)
	return nil
}

func (s *set) RemoveKey(key Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, k := range s.keys {
		if k == key {
			switch i {
			case 0:
				s.keys = s.keys[1:]
			case len(s.keys) - 1:
				s.keys = s.keys[:i]
			default:
				s.keys = append(s.keys[:i], s.keys[i+1:]...)
			}
			return nil
		}
	}
	return fmt.Errorf(`(jwk.Set).RemoveKey: specified key does not exist in set`)
}

func (s *set) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.keys = nil
	s.privateParams = make(map[string]any)
	return nil
}

func (s *set) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ret := make([]string, len(s.privateParams))
	var i int
	for k := range s.privateParams {
		ret[i] = k
		i++
	}
	return ret
}

func (s *set) MarshalJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	buf := pool.BytesBuffer().Get()
	defer pool.BytesBuffer().Put(buf)
	enc := json.NewEncoder(buf)

	fields := make([]string, 0, 1+len(s.privateParams))
	fields = append(fields, keysKey)
	for k := range s.privateParams {
		fields = append(fields, k)
	}
	sort.Strings(fields)

	buf.WriteByte(tokens.OpenCurlyBracket)
	for i, field := range fields {
		if i > 0 {
			buf.WriteByte(tokens.Comma)
		}
		fmt.Fprintf(buf, `%q:`, field)
		if field != keysKey {
			if err := enc.Encode(s.privateParams[field]); err != nil {
				return nil, fmt.Errorf(`failed to marshal field %q: %w`, field, err)
			}
		} else {
			buf.WriteByte(tokens.OpenSquareBracket)
			for j, k := range s.keys {
				if j > 0 {
					buf.WriteByte(tokens.Comma)
				}
				if err := enc.Encode(k); err != nil {
					return nil, fmt.Errorf(`failed to marshal key #%d: %w`, i, err)
				}
			}
			buf.WriteByte(tokens.CloseSquareBracket)
		}
	}
	buf.WriteByte(tokens.CloseCurlyBracket)

	ret := make([]byte, buf.Len())
	copy(ret, buf.Bytes())
	return ret, nil
}

func (s *set) setMaxKeys(n int) {
	s.maxKeys = n
}

func (s *set) setRejectDuplicateKID(v *bool) {
	s.rejectDuplicateKID = v
}

func (s *set) setStrictKeySetParsing(v *bool) {
	s.strictKeySetParsing = v
}

// UnmarshalJSON streams a JWKS document. The "keys" array is read
// element-by-element with the configured cap enforced BEFORE the
// (cap+1)-th element is decoded — an attacker-controlled input length
// cannot force allocation past the cap. This entry point requires
// JWKS shape; bare JWK input is rejected here. Callers that don't
// know the shape ahead of time should use [Parse], which dispatches.
func (s *set) UnmarshalJSON(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.privateParams = make(map[string]any)
	s.keys = nil

	var options []ParseOption
	var ignoreParseError bool
	if dc := s.dc; dc != nil {
		if localReg := dc.Registry(); localReg != nil {
			options = append(options, withLocalRegistry(localReg))
		}
		ignoreParseError = dc.IgnoreParseError()
	}

	maxK := s.maxKeys
	if maxK <= 0 {
		maxK = int(maxKeys.Load())
	}
	rejectDupKid := rejectDuplicateKID.Load()
	if s.rejectDuplicateKID != nil {
		rejectDupKid = *s.rejectDuplicateKID
	}
	strict := strictKeySetParsing.Load()
	if s.strictKeySetParsing != nil {
		strict = *s.strictKeySetParsing
	}

	dec := json.NewDecoder(bytes.NewReader(data))
LOOP:
	for {
		tok, err := dec.Token()
		if err != nil {
			return fmt.Errorf(`error reading token: %w`, err)
		}

		switch tok := tok.(type) {
		case json.Delim:
			if tok == tokens.CloseCurlyBracket {
				break LOOP
			} else if tok != tokens.OpenCurlyBracket {
				return fmt.Errorf(`expected '%c' but got '%c'`, tokens.OpenCurlyBracket, tok)
			}
		case string:
			switch tok {
			case "keys":
				openTok, err := dec.Token()
				if err != nil {
					return fmt.Errorf(`failed to decode "keys": %w`, err)
				}
				openDelim, ok := openTok.(json.Delim)
				if !ok || openDelim != tokens.OpenSquareBracket {
					return fmt.Errorf(`failed to decode "keys": expected '%c' but got %v`, tokens.OpenSquareBracket, openTok)
				}

				var seenKIDs map[string]struct{}
				if rejectDupKid {
					seenKIDs = make(map[string]struct{})
				}
				var i int
				for dec.More() {
					if i >= maxK {
						return fmt.Errorf(`too many keys in "keys" array: max %d`, maxK)
					}
					var raw json.RawMessage
					if err := dec.Decode(&raw); err != nil {
						return fmt.Errorf(`failed to decode "keys": %w`, err)
					}
					key, err := ParseKey(raw, options...)
					if err != nil {
						// ignoreParseError is checked first so its
						// long-standing "drop the entry" behavior is
						// unchanged regardless of the strict flag. Then:
						// strict (v3 default) fails the whole set; otherwise
						// the entry is retained as an UnsupportedKey
						// placeholder (RFC 7517 §5, opt-in via
						// WithStrictKeySetParsing(false)).
						if ignoreParseError {
							i++
							continue
						}
						if strict {
							return fmt.Errorf(`failed to decode key #%d in "keys": %w`, i, err)
						}
						// dec.Decode may reuse its buffer, so
						// newUnsupportedKey clones the raw bytes.
						key = newUnsupportedKey(raw, err)
					}
					if seenKIDs != nil {
						if kid, ok := key.KeyID(); ok && kid != "" {
							if _, dup := seenKIDs[kid]; dup {
								return fmt.Errorf(`duplicate "kid" %q in "keys" array`, kid)
							}
							seenKIDs[kid] = struct{}{}
						}
					}
					s.keys = append(s.keys, key)
					i++
				}
				closeTok, err := dec.Token()
				if err != nil {
					return fmt.Errorf(`failed to decode "keys": %w`, err)
				}
				closeDelim, ok := closeTok.(json.Delim)
				if !ok || closeDelim != tokens.CloseSquareBracket {
					return fmt.Errorf(`failed to decode "keys": expected '%c' but got %v`, tokens.CloseSquareBracket, closeTok)
				}
			default:
				var v any
				if err := dec.Decode(&v); err != nil {
					return fmt.Errorf(`failed to decode value for key %q: %w`, tok, err)
				}
				s.privateParams[tok] = v
			}
		}
	}
	return nil
}

func (s *set) LookupKeyID(kid string) (Key, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, key := range s.keys {
		gotkid, ok := key.KeyID()
		if ok && gotkid == kid {
			return key, true
		}
	}
	return nil, false
}

func (s *set) DecodeCtx() DecodeCtx {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dc
}

func (s *set) SetDecodeCtx(dc DecodeCtx) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dc = dc
}

func (s *set) Clone() (Set, error) {
	s2 := newSet()

	s.mu.RLock()
	defer s.mu.RUnlock()

	s2.keys = make([]Key, len(s.keys))
	copy(s2.keys, s.keys)

	maps.Copy(s2.privateParams, s.privateParams)

	return s2, nil
}
