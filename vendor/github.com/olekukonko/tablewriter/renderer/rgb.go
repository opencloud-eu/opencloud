package renderer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

// SGR parameters used to build 24-bit ("true color") sequences.
//
// fatih/color keeps the equivalent prefixes unexported, so tablewriter defines
// them here. This lets RGB colors be emitted through the already vendored
// fatih/color version, without requiring a newer release of that dependency.
const (
	fgTrueColor   color.Attribute = 38 // select foreground color
	bgTrueColor   color.Attribute = 48 // select background color
	trueColorMode color.Attribute = 2  // "2" selects the 24-bit RGB sub-mode
)

// clampRGBComponent constrains a single color channel to the valid 0-255 range
// so out-of-range values produce a usable color instead of a malformed escape
// sequence.
func clampRGBComponent(v int) color.Attribute {
	switch {
	case v < 0:
		return 0
	case v > 255:
		return 255
	default:
		return color.Attribute(v)
	}
}

// RGB returns foreground Colors for a 24-bit ("true color") value. Each of r, g
// and b is a channel in the range 0-255; values outside that range are clamped.
//
// The result is a plain Colors slice, so it can be used anywhere Colors are
// accepted and combined with regular attributes via append, for example:
//
//	Tint{FG: append(renderer.RGB(255, 128, 0), color.Bold)}
//
// Terminals that do not support 24-bit color may ignore or approximate the
// sequence.
func RGB(r, g, b int) Colors {
	return Colors{fgTrueColor, trueColorMode, clampRGBComponent(r), clampRGBComponent(g), clampRGBComponent(b)}
}

// BgRGB returns background Colors for a 24-bit ("true color") value. It behaves
// like RGB but sets the background rather than the foreground.
func BgRGB(r, g, b int) Colors {
	return Colors{bgTrueColor, trueColorMode, clampRGBComponent(r), clampRGBComponent(g), clampRGBComponent(b)}
}

// Hex parses a hexadecimal color string and returns foreground Colors for the
// equivalent 24-bit color. Both the shorthand "#RGB" and full "#RRGGBB" forms
// are accepted, with or without the leading '#'. An error is returned for
// strings that are not a valid hex color.
func Hex(s string) (Colors, error) {
	r, g, b, err := parseHexColor(s)
	if err != nil {
		return nil, err
	}
	return RGB(r, g, b), nil
}

// BgHex behaves like Hex but returns background Colors.
func BgHex(s string) (Colors, error) {
	r, g, b, err := parseHexColor(s)
	if err != nil {
		return nil, err
	}
	return BgRGB(r, g, b), nil
}

// parseHexColor decodes a "#RGB" or "#RRGGBB" color (the leading '#' is
// optional) into its red, green and blue components.
func parseHexColor(s string) (r, g, b int, err error) {
	h := strings.TrimPrefix(strings.TrimSpace(s), "#")

	switch len(h) {
	case 3:
		// Expand shorthand: "abc" -> "aabbcc".
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	case 6:
		// Already full length.
	default:
		return 0, 0, 0, fmt.Errorf("tablewriter: invalid hex color %q: want \"#RGB\" or \"#RRGGBB\"", s)
	}

	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("tablewriter: invalid hex color %q: %w", s, err)
	}

	return int(v >> 16 & 0xFF), int(v >> 8 & 0xFF), int(v & 0xFF), nil
}
