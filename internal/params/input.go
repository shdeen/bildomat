package params

// FlagInputs maps supplied parameter names to their parsed values.
type FlagInputs map[FlagType]any

// Supplied reports whether a parameter has a meaningful supplied value. Thought output requires
// true, and input media requires at least one source.
func (userInputs FlagInputs) Supplied(flag FlagType) bool {
	inputValue, ok := userInputs[flag]
	if !ok {
		return false
	}

	switch flag {
	case FlagTypeThoughts:
		// A stored value of another type is not a request for thoughts.
		thoughtsRequested, ok := inputValue.(bool)

		return ok && thoughtsRequested
	case FlagTypeInputMedia:
		// A stored value of another type carries no media paths.
		paths, ok := inputValue.([]string)

		return ok && len(paths) > 0
	default:
		return true
	}
}
