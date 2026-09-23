package main

// File: tools/flagsetup/sentinel.go The generator's sentinel errors. The generator imports nothing
// from this module, so it declares its own.

import (
	"errors"
	"fmt"
)

// Sentinels of the parameter flag document.
//   - errParamFlags is the category root for a broken parameter flag document. A document that
//     cannot be read, and generated source that cannot be formatted or written, wrap it beside the
//     failure's own cause.
//   - errParamFlagsDecode marks a document that failed to decode.
//   - errParamFlagsTrailing marks content after the document.
//   - errParamFlagsInvalid marks a decoded document with invalid content.
var (
	errParamFlags         = errors.New("param-flag")
	errParamFlagsDecode   = fmt.Errorf("%w: decode failed", errParamFlags)
	errParamFlagsTrailing = fmt.Errorf("%w: trailing content after the enumeration document", errParamFlagsDecode)
	errParamFlagsInvalid  = fmt.Errorf("%w: invalid content", errParamFlags)
)
