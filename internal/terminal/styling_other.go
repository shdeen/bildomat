//go:build !windows

package terminal

// EnableStyling reports whether the supplied stream is a terminal that renders styling escape
// sequences. On every platform other than Windows a terminal renders them as they are, so the
// answer is IsTerminal's.
func EnableStyling(stream any) bool {
	return IsTerminal(stream)
}
