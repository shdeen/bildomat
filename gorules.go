//go:build ruleguard

// Package gorules defines the repository's custom lint rule for literal prose
// passed to function calls. The gocritic ruleguard check loads this file through
// .golangci.yml when golangci-lint-v2 run ./... runs from the repository root.
// The ruleguard build tag excludes it from normal builds and tests; it is linter
// input and is not part of the bild executable.
package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// hardcodedCopy registers a diagnostic for double-quoted string arguments that
// contain at least two consecutive ASCII letters on each side of a space.
// The diagnostic directs the author to copy.toml or a named constant; this
// heuristic does not inspect raw strings or verify where other text is stored.
func hardcodedCopy(m dsl.Matcher) {
	m.Match(`$f($*_, $s, $*_)`, `$f($s)`).
		Where(m["s"].Type.Is(`string`) &&
			m["s"].Text.Matches(`^".*[a-zA-Z][a-zA-Z] [a-zA-Z][a-zA-Z].*"$`)).
		Report(`hardcoded copy $s: put it in copy.toml or a named constant`)
}
