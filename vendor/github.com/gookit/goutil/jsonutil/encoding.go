package jsonutil

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// MustString encode data to json string, will panic on error
func MustString(v any) string {
	bs, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(bs)
}

// Encode data to json bytes. alias of json.Marshal
func Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

// EncodePretty encode data to pretty JSON bytes.
func EncodePretty(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "    ")
}

// EncodeString encode data to JSON string.
func EncodeString(v any) (string, error) {
	bs, err := json.MarshalIndent(v, "", "    ")
	return string(bs), err
}

// EncodeToWriter encode data to json and write to writer.
func EncodeToWriter(v any, w io.Writer) error {
	return json.NewEncoder(w).Encode(v)
}

// EncodeUnescapeHTML data to json bytes. will close escape HTML
func EncodeUnescapeHTML(v any) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode json bytes to data ptr. alias of json.Unmarshal
func Decode(bts []byte, ptr any) error {
	return json.Unmarshal(bts, ptr)
}

// DecodeString json string to data ptr.
func DecodeString(str string, ptr any) error {
	return json.Unmarshal([]byte(str), ptr)
}

// DecodeReader decode JSON from io reader.
func DecodeReader(r io.Reader, ptr any) error {
	return json.NewDecoder(r).Decode(ptr)
}

// DecodeFile decode JSON from file, bind data to ptr.
func DecodeFile(file string, ptr any) error {
	bs, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	return json.Unmarshal(bs, ptr)
}