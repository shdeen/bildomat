// Package config embeds the descriptor-class provider configurations: the
// providers served entirely by the shared request code in internal/provider,
// each declared by one JSON document named after its ID and nothing else.
package config

import _ "embed"

// The descriptor-class provider IDs. Each names its configuration file, <ID>.json.
const (
	IDOpenAI     = "openai"
	IDXAI        = "xai"
	IDOpenRouter = "openrouter"
	IDRecraft    = "recraft"
)

// The embedded configuration documents, one per descriptor-class provider.
var (
	//go:embed openai.json
	JSONOpenAI []byte

	//go:embed xai.json
	JSONXAI []byte

	//go:embed openrouter.json
	JSONOpenRouter []byte

	//go:embed recraft.json
	JSONRecraft []byte
)
