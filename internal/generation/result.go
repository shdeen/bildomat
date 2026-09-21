package generation

import "github.com/shdeen/bildomat/internal/artifact"

// Result contains the outputs from one provider request.
//   - Preparation: final submission facts, including completed changes on failure
//   - Artifacts: the generated media
//   - Thoughts: the provider's thought text
type Result struct {
	Preparation Preparation
	Artifacts   []artifact.Media
	Thoughts    []string
}
