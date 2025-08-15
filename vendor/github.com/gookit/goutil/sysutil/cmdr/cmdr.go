// Package cmdr Provide for quick build and run a cmd, batch run multi cmd tasks
package cmdr

import (
	"strings"

	"github.com/gookit/goutil/x/ccolor"
)

// PrintCmdline on before exec
func PrintCmdline(c *Cmd) {
	if c.DryRun {
		ccolor.Yellowln("DRY-RUN>", c.RawLine())
	} else {
		ccolor.Yellowln(">", c.RawLine())
	}
}

// PrintCmdline2 on before exec
func PrintCmdline2(c *Cmd) {
	if c.Dir != "" {
		ccolor.Magentaln("> Workdir:", c.Dir)
	}
	if c.DryRun {
		ccolor.Yellowln("DRY-RUN>", c.RawLine())
	} else {
		ccolor.Yellowln(">", c.RawLine())
	}
}

// OutputLines split output to lines
func OutputLines(output string) []string {
	output = strings.TrimSuffix(output, "\n")
	if output == "" {
		return nil
	}
	return strings.Split(output, "\n")
}

// FirstLine from command output
func FirstLine(output string) string {
	if i := strings.Index(output, "\n"); i >= 0 {
		return output[0:i]
	}
	return output
}
