package output

// File: internal/output/style.go ANSI escape sequences used by terminal output.

// Terminal styling sequences, omitted by callers for plain output.
//   - ansiDim: faint text
//   - ansiClay: the accent for parameter values and saved paths
//   - ansiSteelBlue: the accent for headings and prompts
//   - ansiErrorAccent: the error-prefix accent
//   - ansiReset: the default text rendition
const (
	ansiDim = "\x1b[2m"

	ansiClay = "\x1b[38;5;173m"

	ansiSteelBlue = "\x1b[38;5;67m"

	ansiErrorAccent = "\x1b[38;5;167m"

	ansiReset = "\x1b[0m"
)
