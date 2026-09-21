package output

// Invariants tested:
// 1. Unconstrained model explanation: A provider's model group without declared parameter
//    limits includes explanatory text beneath its own heading.

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestProviderUnconstrainedGroup verifies invariant #1: Unconstrained model explanation.
//
// What makes it or breaks it:
// The unconstrained group's heading appears, followed by nonempty explanatory text that
// does not belong to the following constrained model group.
//
// Test class: Core.
func TestProviderUnconstrainedGroup(t *testing.T) {
	paramFlags := params.Flags()

	providerConfig := catalog.Provider{
		ID: "example", DisplayName: "Example",
		Models: []catalog.Model{
			{
				ID: "unrestricted", Name: "Unrestricted", Media: media.Image,
				Params: []params.Definition{{FlagID: params.FlagTypeAspect}},
			},
			{
				ID: "square", Name: "Square", Media: media.Image,
				Params: []params.Definition{{FlagID: params.FlagTypeAspect, AllowedValues: []string{"1:1"}}},
			},
		},
	}
	providerText := captureStdout(t, func() {
		_ = PrintProviderInfo(os.Stdout, &providerConfig, paramFlags, true, false, map[media.Kind]string{media.Image: "image", media.Video: "video"}, FileIsTTY(os.Stdout))
	})
	groupHeading := fmt.Sprintf(GroupIntro, "unrestricted") + ":"

	_, groupText, groupFound := strings.Cut(providerText, groupHeading)
	if !groupFound {
		t.Errorf("✗ the unconstrained model group is missing:\n%s", providerText)
	}

	groupText = strings.TrimPrefix(groupText, "\n")

	groupDetails, _, _ := strings.Cut(groupText, "\n\n")
	if strings.TrimSpace(groupDetails) == "" || strings.Contains(groupDetails, fmt.Sprintf(GroupIntro, "square")) {
		t.Errorf("✗ the unconstrained model group has no explanatory text:\n%s", providerText)
	}

	if !t.Failed() {
		t.Log("✓ the unconstrained model group includes explanatory text")
	}
}

// captureStdout runs fn with os.Stdout redirected and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("💣 os.Pipe: %v", err)
	}

	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("💣 close pipe: %v", err)
	}

	os.Stdout = old

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("💣 read captured stdout: %v", err)
	}

	return string(out)
}
