// This file is available under the Apache 2.0 License
// This file was copied from: https://github.com/briandowns/spinner
//
// Please see the LICENSE file for the copy of the Apache 2.0 License.
//
// Modifications:
//
// - made validColors set map more idiomatic with an empty struct value
// - added a function for creating color functions from color list

package yacspin

import (
	"fmt"

	"github.com/fatih/color"
)

// ValidColors holds the list of the strings that are mapped to
// github.com/fatih/color color attributes. Any of these colors / attributes can
// be used with the *Spinner type, and it should be reflected in the output.
var ValidColors = map[string]struct{}{
	// default colors for backwards compatibility
	"black":   {},
	"red":     {},
	"green":   {},
	"yellow":  {},
	"blue":    {},
	"magenta": {},
	"cyan":    {},
	"white":   {},

	// attributes
	"reset":        {},
	"bold":         {},
	"faint":        {},
	"italic":       {},
	"underline":    {},
	"blinkslow":    {},
	"blinkrapid":   {},
	"reversevideo": {},
	"concealed":    {},
	"crossedout":   {},

	// foreground text
	"fgBlack":   {},
	"fgRed":     {},
	"fgGreen":   {},
	"fgYellow":  {},
	"fgBlue":    {},
	"fgMagenta": {},
	"fgCyan":    {},
	"fgWhite":   {},

	// foreground Hi-Intensity text
	"fgHiBlack":   {},
	"fgHiRed":     {},
	"fgHiGreen":   {},
	"fgHiYellow":  {},
	"fgHiBlue":    {},
	"fgHiMagenta": {},
	"fgHiCyan":    {},
	"fgHiWhite":   {},

	// background text
	"bgBlack":   {},
	"bgRed":     {},
	"bgGreen":   {},
	"bgYellow":  {},
	"bgBlue":    {},
	"bgMagenta": {},
	"bgCyan":    {},
	"bgWhite":   {},

	// background Hi-Intensity text
	"bgHiBlack":   {},
	"bgHiRed":     {},
	"bgHiGreen":   {},
	"bgHiYellow":  {},
	"bgHiBlue":    {},
	"bgHiMagenta": {},
	"bgHiCyan":    {},
	"bgHiWhite":   {},
}

// returns a valid color's foreground text color attribute
var colorAttributeMap = map[string]color.Attribute{
	// default colors for backwards compatibility
	"black":   color.FgBlack,
	"red":     color.FgRed,
	"green":   color.FgGreen,
	"yellow":  color.FgYellow,
	"blue":    color.FgBlue,
	"magenta": color.FgMagenta,
	"cyan":    color.FgCyan,
	"white":   color.FgWhite,

	// attributes
	"reset":        color.Reset,
	"bold":         color.Bold,
	"faint":        color.Faint,
	"italic":       color.Italic,
	"underline":    color.Underline,
	"blinkslow":    color.BlinkSlow,
	"blinkrapid":   color.BlinkRapid,
	"reversevideo": color.ReverseVideo,
	"concealed":    color.Concealed,
	"crossedout":   color.CrossedOut,

	// foreground text colors
	"fgBlack":   color.FgBlack,
	"fgRed":     color.FgRed,
	"fgGreen":   color.FgGreen,
	"fgYellow":  color.FgYellow,
	"fgBlue":    color.FgBlue,
	"fgMagenta": color.FgMagenta,
	"fgCyan":    color.FgCyan,
	"fgWhite":   color.FgWhite,

	// foreground Hi-Intensity text colors
	"fgHiBlack":   color.FgHiBlack,
	"fgHiRed":     color.FgHiRed,
	"fgHiGreen":   color.FgHiGreen,
	"fgHiYellow":  color.FgHiYellow,
	"fgHiBlue":    color.FgHiBlue,
	"fgHiMagenta": color.FgHiMagenta,
	"fgHiCyan":    color.FgHiCyan,
	"fgHiWhite":   color.FgHiWhite,

	// background text colors
	"bgBlack":   color.BgBlack,
	"bgRed":     color.BgRed,
	"bgGreen":   color.BgGreen,
	"bgYellow":  color.BgYellow,
	"bgBlue":    color.BgBlue,
	"bgMagenta": color.BgMagenta,
	"bgCyan":    color.BgCyan,
	"bgWhite":   color.BgWhite,

	// background Hi-Intensity text colors
	"bgHiBlack":   color.BgHiBlack,
	"bgHiRed":     color.BgHiRed,
	"bgHiGreen":   color.BgHiGreen,
	"bgHiYellow":  color.BgHiYellow,
	"bgHiBlue":    color.BgHiBlue,
	"bgHiMagenta": color.BgHiMagenta,
	"bgHiCyan":    color.BgHiCyan,
	"bgHiWhite":   color.BgHiWhite,
}

// validColor will make sure the given color is actually allowed
func validColor(c string) bool {
	_, ok := ValidColors[c]

	return ok
}

func colorFunc(colors ...string) (func(format string, a ...interface{}) string, error) {
	if len(colors) == 0 {
		return fmt.Sprintf, nil
	}

	attrib := make([]color.Attribute, len(colors))

	for i, color := range colors {
		if !validColor(color) {
			return nil, fmt.Errorf("%s is not a valid color", color)
		}

		attrib[i] = colorAttributeMap[color]
	}

	return color.New(attrib...).SprintfFunc(), nil
}
