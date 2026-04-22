package locks

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLegacyLockParser(t *testing.T) {
	tests := []struct {
		name      string
		lock      string
		cleanLock string
	}{
		{
			name:      "JsonStringWithLKey",
			lock:      createJsonString(map[string]any{"L": "12345678", "F": 4, "E": 2, "C": "", "P": "3453345345346", "M": "12345678"}),
			cleanLock: "12345678",
		},
		{
			name:      "JsonStringWithSKey",
			lock:      createJsonString(map[string]any{"S": "12345678", "F": 4, "E": 2, "C": "", "P": "3453345345346", "M": "12345678"}),
			cleanLock: "12345678",
		},
		{
			name:      "PlainString",
			lock:      "12345678",
			cleanLock: "12345678",
		},
		{
			name:      "JsonStringUnknownFormat",
			lock:      createJsonString(map[string]any{"A": "12345678", "F": 4, "E": 2, "C": "", "P": "3453345345346", "X": "12345678"}),
			cleanLock: `{"A":"12345678","C":"","E":2,"F":4,"P":"3453345345346","X":"12345678"}`,
		},
		{
			name:      "InvalidJsonString",
			lock:      `"A":"12345678","C":"","E":2,"F":4,"P":"3453345345346","X":"12345678"}`,
			cleanLock: `"A":"12345678","C":"","E":2,"F":4,"P":"3453345345346","X":"12345678"}`,
		},
		{
			name:      "EmptyString",
			lock:      "",
			cleanLock: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lockParser := &LegacyLockParser{}
			lock := lockParser.ParseLock(test.lock)
			assert.Equal(t, test.cleanLock, lock)
		})
	}
}

func TestNoopLockParser(t *testing.T) {
	tests := []struct {
		name string
		lock string
	}{
		{
			name: "PlainString",
			lock: "123",
		},
		{
			name: "EmptyString",
			lock: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lockParser := &NoopLockParser{}
			lock := lockParser.ParseLock(test.lock)
			assert.Equal(t, test.lock, lock)
		})
	}
}

func createJsonString(input map[string]any) string {
	rawData, err := json.Marshal(&input)
	if err != nil {
		return ""
	}
	return string(rawData)
}
