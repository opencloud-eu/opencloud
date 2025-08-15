package comdef

type (
	// MarshalFunc define
	MarshalFunc func(v any) ([]byte, error)

	// UnmarshalFunc define
	UnmarshalFunc func(bts []byte, ptr any) error
)

// Serializer interface definition
type Serializer interface {
	Serialize(v any) ([]byte, error)
	Deserialize(data []byte, v any) error
}

// GoSerializer interface definition
type GoSerializer interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(v []byte, ptr any) error
}

// Codec interface definition
type Codec interface {
	Decode(blob []byte, v any) (err error)
	Encode(v any) (out []byte, err error)
}
