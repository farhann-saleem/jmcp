package output

import "os"

const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiDim    = "\033[2m"
)

type Colorizer struct {
	enabled bool
}

func NewColorizer(noColor bool) *Colorizer {
	return &Colorizer{enabled: !noColor && isTerminalStdout()}
}

func DisabledColorizer() *Colorizer {
	return &Colorizer{enabled: false}
}

func (c *Colorizer) Red(s string) string {
	if !c.enabled {
		return s
	}
	return ansiRed + s + ansiReset
}

func (c *Colorizer) Green(s string) string {
	if !c.enabled {
		return s
	}
	return ansiGreen + s + ansiReset
}

func (c *Colorizer) Yellow(s string) string {
	if !c.enabled {
		return s
	}
	return ansiYellow + s + ansiReset
}

func (c *Colorizer) Cyan(s string) string {
	if !c.enabled {
		return s
	}
	return ansiCyan + s + ansiReset
}

func (c *Colorizer) Dim(s string) string {
	if !c.enabled {
		return s
	}
	return ansiDim + s + ansiReset
}

func (c *Colorizer) Status(ok bool, s string) string {
	if ok {
		return c.Green(s)
	}
	return c.Red(s)
}

func isTerminalStdout() bool {
	info, err := os.Stdout.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}
