package jwk

import (
	"sync"

	"github.com/lestrrat-go/jwx/v3/internal/json"
)

// AsymmetricKey describes a Key that represents a key in an asymmetric key pair,
// which in turn can be either a private or a public key. This interface
// allows those keys to be queried if they are one or the other.
type AsymmetricKey interface {
	IsPrivate() bool
}

// KeyUsageType is used to denote what this key should be used for
type KeyUsageType string

const (
	// ForSignature is the value used in the headers to indicate that
	// this key should be used for signatures
	ForSignature KeyUsageType = "sig"
	// ForEncryption is the value used in the headers to indicate that
	// this key should be used for encrypting
	ForEncryption KeyUsageType = "enc"
)

type KeyOperation string
type KeyOperationList []KeyOperation

const (
	KeyOpSign       KeyOperation = "sign"       // (compute digital signature or MAC)
	KeyOpVerify     KeyOperation = "verify"     // (verify digital signature or MAC)
	KeyOpEncrypt    KeyOperation = "encrypt"    // (encrypt content)
	KeyOpDecrypt    KeyOperation = "decrypt"    // (decrypt content and validate decryption, if applicable)
	KeyOpWrapKey    KeyOperation = "wrapKey"    // (encrypt key)
	KeyOpUnwrapKey  KeyOperation = "unwrapKey"  // (decrypt key and validate decryption, if applicable)
	KeyOpDeriveKey  KeyOperation = "deriveKey"  // (derive key)
	KeyOpDeriveBits KeyOperation = "deriveBits" // (derive bits not to be used as a key)
)

// Set represents JWKS object, a collection of jwk.Key objects.
//
// Sets can be marshaled and unmarshaled with the standard
// `"encoding/json".Marshal` and `"encoding/json".Unmarshal`. The
// unmarshal path requires JWKS shape (an object with a "keys" field).
// For input that may be either a single bare JWK or a JWKS, use
// [Parse], which dispatches between the two shapes and always returns
// a `jwk.Set`.
//
// JWKS-level extension members (any top-level field other than "keys")
// are preserved as set-level private parameters and are accessible via
// the `Field()` method. Per-key extension members live on the
// individual `jwk.Key` objects, accessible via that key's `Field()`.
//
//nolint:interfacebloat
type Set interface {
	// AddKey adds the specified key. If the key already exists in the set,
	// an error is returned.
	AddKey(Key) error

	// Clear resets the list of keys associated with this set, emptying the
	// internal list of `jwk.Key`s, as well as clearing any other non-key
	// fields
	Clear() error

	// Get returns the key at index `idx`. If the index is out of range,
	// then the second return value is false.
	Key(int) (Key, bool)

	// Get returns the value of a private field in the key set.
	//
	// For the purposes of a key set, any field other than the "keys" field is
	// considered to be a private field. In other words, you cannot use this
	// method to directly access the list of keys in the set
	Get(string, any) error

	// Set sets the value of a single field.
	//
	// This method, which takes an `any`, exists because
	// these objects can contain extra _arbitrary_ fields that users can
	// specify, and there is no way of knowing what type they could be.
	Set(string, any) error

	// Remove removes the specified non-key field from the set.
	// Keys may not be removed using this method. See RemoveKey for
	// removing keys.
	Remove(string) error

	// Index returns the index where the given key exists, -1 otherwise
	Index(Key) int

	// Len returns the number of keys in the set
	Len() int

	// LookupKeyID returns the first key matching the given key id.
	//
	// The second return value is false if there are no keys matching the key id.
	// The set *may* contain multiple keys with the same key id. If you
	// need all of them, Len() and Key(int)
	//
	// This method is meant to be used to lookup a key with a unique ID.
	// Bacauseof this, you cannot use this method to lookup keys with an empty key ID
	// (i.e. `kid` is not specified, or is an empty string).
	LookupKeyID(string) (Key, bool)

	// RemoveKey removes the key from the set.
	// RemoveKey returns an error when the specified key does not exist
	// in set.
	RemoveKey(Key) error

	// Keys returns the list of keys present in the Set, except for `keys`.
	// e.g. if you had `{"keys": ["a", "b"], "c": .., "d": ...}`, this method would
	// return `["c", "d"]`. Note that the order of the keys is not guaranteed.
	//
	// TODO: name is confusing between this and Key()
	Keys() []string

	// Clone create a new set with identical keys. Keys themselves are not cloned.
	Clone() (Set, error)
}

type set struct {
	keys                []Key
	mu                  sync.RWMutex
	dc                  DecodeCtx
	privateParams       map[string]any
	maxKeys             int   // scratch cap consumed by UnmarshalJSON; 0 means use global default
	rejectDuplicateKID  *bool // scratch override consumed by UnmarshalJSON; nil falls back to global
	strictKeySetParsing *bool // scratch override consumed by UnmarshalJSON; nil falls back to global
}

type PublicKeyer interface {
	// PublicKey creates the corresponding PublicKey type for this object.
	// All fields are copied onto the new public key, except for those that are not allowed.
	// Returned value must not be the receiver itself.
	PublicKey() (Key, error)
}

type DecodeCtx interface {
	json.DecodeCtx
	IgnoreParseError() bool
}
type KeyWithDecodeCtx interface {
	SetDecodeCtx(DecodeCtx)
	DecodeCtx() DecodeCtx
}

// Used internally: It's used to lock a key
type rlocker interface {
	rlock()
	runlock()
}
