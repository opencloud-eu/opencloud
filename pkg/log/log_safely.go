package log

import "github.com/rs/zerolog"

const (
	logMaxStrLength      = 512
	logMaxStrArrayLength = 16 // 8kb
)

// Safely caps a string to a given size to avoid log bombing.
// Use this function to wrap strings that are user input (HTTP headers, path parameters, URI parameters, HTTP body, ...).
func SafeString(text string) string {
	runes := []rune(text)

	if len(runes) <= logMaxStrLength {
		return text
	} else {
		return string(runes[0:logMaxStrLength-1]) + `\u2026` // hellip
	}
}

type SafeLogStringArrayMarshaller struct {
	array []string
}

func (m SafeLogStringArrayMarshaller) MarshalZerologArray(a *zerolog.Array) {
	for i, elem := range m.array {
		if i >= logMaxStrArrayLength {
			return
		}
		a.Str(SafeString(elem))
	}
}

var _ zerolog.LogArrayMarshaler = SafeLogStringArrayMarshaller{}

func SafeStringArray(array []string) SafeLogStringArrayMarshaller {
	return SafeLogStringArrayMarshaller{array: array}
}

func From(context zerolog.Context) *Logger {
	return &Logger{Logger: context.Logger()}
}
