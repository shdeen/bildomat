// Package config embeds the descriptor-class provider configurations: the providers served entirely
// by the shared request code in internal/provider, each declared by one JSON document named after
// its ID and nothing else.
package config

import _ "embed"

// Descriptor-class provider IDs name their embedded configuration files.
//   - IDOpenAI: OpenAI
//   - IDXAI: xAI
//   - IDOpenRouter: OpenRouter
//   - IDRecraft: Recraft
const (
	IDOpenAI     = "openai"
	IDXAI        = "xai"
	IDOpenRouter = "openrouter"
	IDRecraft    = "recraft"
)

// Embedded documents define descriptor-class providers.
//   - JSONOpenAI: OpenAI configuration
//   - JSONXAI: xAI configuration
//   - JSONOpenRouter: OpenRouter configuration
//   - JSONRecraft: Recraft configuration
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
