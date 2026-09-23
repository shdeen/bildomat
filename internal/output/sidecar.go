package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// frontMatterFence is the line that opens and closes the front matter and separates one thought
// from the next.
const frontMatterFence = "---"

// RenderSidecar returns Markdown with the prompt, model ID, current UTC timestamp, and nonempty
// parameters in flag-record order. It records the supplied input-media sources under input-media
// and places thought chunks after the front matter, separated by fences.
func RenderSidecar(prompt, modelID string, paramFlags []params.Flag, adjusted params.Values, inputMedia []media.Input, thoughts []string) []byte {
	var builder strings.Builder
	builder.WriteString(frontMatterFence + "\n")
	fmt.Fprintf(&builder, FrontMatterPromptForm+"\n", prompt)
	fmt.Fprintf(&builder, FrontMatterModelForm+"\n", modelID)
	fmt.Fprintf(&builder, FrontMatterTimestampForm+"\n", documentTimestamp(time.Now()))

	for paramFlagIndex := range paramFlags {
		paramFlag := &paramFlags[paramFlagIndex]
		if paramFlag.FlagID == params.FlagTypeInputMedia {
			if len(inputMedia) > 0 {
				fmt.Fprintf(&builder, FrontMatterParamForm+"\n", paramFlag.FlagID, joinQuotedMediaSources(inputMedia))
			}

			continue
		}

		paramVal, ok := adjusted[paramFlag.FlagID]
		if !ok {
			continue
		}

		rendered := params.FormatValue(paramVal)
		if rendered == "" {
			continue
		}

		fmt.Fprintf(&builder, FrontMatterParamForm+"\n", paramFlag.FlagID, rendered)
	}

	builder.WriteString(frontMatterFence + "\n\n")

	for i, t := range thoughts {
		if i > 0 {
			builder.WriteString("\n" + frontMatterFence + "\n\n")
		}

		builder.WriteString(t)
		builder.WriteString("\n")
	}

	return []byte(builder.String())
}
