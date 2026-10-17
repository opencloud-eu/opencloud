package generator

import (
	"fmt"
	"mime"
	"strings"
)

// Operations are the imagor resize/crop operations the generator understands.
const (
	// OpFill center-crops to fill the box exactly (imagor default resize).
	OpFill = "fill"
	// OpFitIn fits within the box without cropping and never upscales (imagor
	// fit-in), preserving the source aspect ratio.
	OpFitIn = "fit-in"
	// OpStretch resizes to the exact box without preserving aspect ratio.
	OpStretch = "stretch"
)

// operationSegment maps an operation to its imagor URL path segment. The fill
// operation is the bare WxH resize (imagor's default), so it contributes no
// leading segment; stretch and fit-in use their imagor names. Note that real
// imagor has no /fill/ route, "fill" is only our internal name for the default
// center-crop resize.
func operationSegment(op string) string {
	switch op {
	case OpFitIn:
		return "fit-in"
	case OpStretch:
		return "stretch"
	default: // OpFill and anything unrecognized fall back to the default resize
		return ""
	}
}

// BuildURL constructs the thumbnail generator processing URL for a matched box
// and operation. The operation segment selects the resize/crop mode (which also
// governs upscaling); the image is re-encoded to the requested output format via
// an imagor filters segment. Like real imagor, the default (fill) resize upscales
// small sources to fill the box exactly; pass noUpscale=true to emit the
// no_upscale() filter so the generator caps the result at the source size.
func BuildURL(baseURL string, width, height int32, operation, format string, noUpscale bool) string {
	base := strings.TrimRight(baseURL, "/")

	box := fmt.Sprintf("%dx%d", width, height)

	filters := "format(" + format + ")"
	if noUpscale {
		filters = "no_upscale():format(" + format + ")"
	}
	filterSegment := "filters:" + filters

	segment := operationSegment(operation)
	if segment == "" {
		return fmt.Sprintf("%s/unsafe/%s/%s/", base, box, filterSegment)
	}
	return fmt.Sprintf("%s/unsafe/%s/%s/%s/", base, segment, box, filterSegment)
}

// DefaultFormats are the types the built-in generator writes, by the name its
// format filter uses for them. A generator behind the URL that writes more than
// that is told so through the configuration.
var DefaultFormats = []string{
	"image/png:png",
	"image/gif:gif",
	"image/jpeg:jpg",
}

// Formats says which format the generator is asked for to get a given type.
type Formats struct {
	byType map[string]string
}

// NewFormats parses "mime:format" entries on top of DefaultFormats.
func NewFormats(entries []string) Formats {
	return Formats{byType: ParsePairs(DefaultFormats, entries)}
}

// For is the name the generator's format filter uses for a type, "" for a type
// it cannot write. It guards the configured type mapping: a target nobody can
// write leaves the caller with its default.
func (f Formats) For(contentType string) string {
	m, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}

	return f.byType[m]
}

// ParsePairs reads "key:value" configuration entries on top of the defaults,
// lowercased and trimmed, so one configured entry does not take the answer away
// from all the others.
func ParsePairs(defaults, entries []string) map[string]string {
	out := map[string]string{}
	for _, entry := range append(append([]string{}, defaults...), entries...) {
		key, value, _ := strings.Cut(strings.ToLower(strings.TrimSpace(entry)), ":")
		if key = strings.TrimSpace(key); key != "" {
			out[key] = strings.TrimSpace(value)
		}
	}

	return out
}
