package output

// File: internal/output/style.go
// The terminal styling sentinels the generation display renders. Each ANSI
// escape sequence is named exactly once here.

const (
	// ansiDim renders the following text in the terminal's faint shade, which
	// stays legible on light and dark backgrounds.
	ansiDim = "\x1b[2m"

	// ansiClay renders the following text in the clay accent color: 256-color
	// palette entry 173, a terracotta legible on light and dark backgrounds.
	// The interactive prompts carry it on their values; the saved-file report
	// carries it on the path.
	ansiClay = "\x1b[38;5;173m"

	// ansiSteelBlue renders the following text in the muted steel-blue accent:
	// 256-color palette entry 67, the clay accent's complementary cool color.
	// The interactive prompts carry it on their sentences.
	ansiSteelBlue = "\x1b[38;5;67m"

	// ansiErrorAccent renders the error prefix in a muted red: 256-color
	// palette entry 167, softer than pure red and legible on light and dark
	// backgrounds.
	ansiErrorAccent = "\x1b[38;5;167m"

	// ansiReset restores the terminal's default rendition.
	ansiReset = "\x1b[0m"

	// ansiEraseStatus returns the cursor to the start of the status line and
	// erases the line's remainder.
	ansiEraseStatus = "\r\x1b[K"
)
