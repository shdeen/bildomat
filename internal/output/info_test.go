package output

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/term"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// Invariants tested:
//  1. Unconstrained model explanation: Given models with unrestricted and restricted aspect ratios,
//     PrintProviderInfo must show the unrestricted model's group heading followed by nonempty
//     explanatory text before the restricted group's heading.
//  2. Provider page media filters: With video selected, PrintProviderInfo must omit the fixture's
//     image models and quality option, show the video model before its duration option, and retain
//     the full catalog counts. With images selected, it must omit the video model and duration
//     option and show an image model before the quality option.
//  3. Provider information page: Given the Alpha Labs fixture, PrintProviderInfo must include the
//     provider name, ID, credential variable, model IDs, and aliases. It must place img-one before
//     vid-one.
//  4. Model information page: Given the image and video fixtures, PrintModelInfo must show their
//     IDs and media kinds, the image model's key and alias, and declared constraint values under
//     the corresponding flag sections. The video duration section must include its rule
//     description, and the image page must omit the tested provider request keys.
//  5. Model bounds and aliases: Given a model with custom size bounds and a lone numeric minimum,
//     PrintModelInfo must include the specified size values and 0.5 minimum in their parameter
//     sections. PrintListing must place both model aliases beside the model ID on one line.
//  6. Model comments: Given an input-media parameter with ModelInfoComment, PrintModelInfo must
//     include that comment after the flag's ordinary description in the input-media option.
//  7. Provider option inventory: Given the Gamma Systems image-only fixture, PrintProviderInfo must
//     include its display name and credential variable, show the aspect-ratio, quality, size, and
//     num-images options, and omit duration.
//  8. Provider option grouping: Given two models with identical quality values, different size
//     values, and only one with num-images, PrintProviderInfo must show quality constraints without
//     model names, show each size value with its model in catalog order, and name only the
//     supporting model under num-images.
//  9. Print provider info layout: Given the provider-page fixture, PrintProviderInfo must write
//     every line within the page width and leave no trailing spaces or tabs.
//  10. Model card: Given the full image-model fixture, PrintModelInfo must show model and provider
//      names and IDs, the model key, credential variable, aliases, and medium. It must order option
//      headings by flag records, align continuation lines, and place each option's descriptions,
//      comments, examples, constraints, and required mark in the specified order. The page must fit
//      the width without trailing whitespace and omit the tested request keys and model
//      description.
//  11. Model card identity variants: Given a video model with allowed durations and a rule
//      description, PrintModelInfo must show its name, ID, fully qualified key, credential
//      variable, and video medium. Its duration option must show the description, allowed values,
//      and rule in that order, and the page must fit the width without trailing whitespace.
//  12. Provider page: Given three image models and one video model, PrintProviderInfo must show
//      provider identity and full catalog counts, then bare model IDs and aliases in catalog order
//      with images before video. It must order each medium's options by flag records, repeat
//      aspect-ratio once per medium, and omit fully qualified example keys. Every line must fit the
//      page width without trailing whitespace.
//  13. Provider option groups: Given shared and differing parameter declarations, PrintProviderInfo
//      must show shared constraints without model names, identify models missing an option, and
//      group differing constraints under the applicable model names. It must preserve the specified
//      guidance and rule order, separate group introductions from details, and omit the
//      model-specific expanded comment.
//  14. Aggregator summary: Given a marked aggregator, PrintProviderInfo must show its identity and
//      full catalog counts, vendors and their counts in the specified count-then-name order, and
//      flag names in record order for each medium. Its footer must name the leading vendors and one
//      valid model key per medium, and it must omit full option headings. The page must fit the
//      width without trailing whitespace.
//  15. Aggregator marked only: Given an unmarked provider with 21 models, PrintProviderInfo must
//      list every bare model ID. Marking the same provider as an aggregator must produce one valid
//      image-model example key and omit the full aspect-ratio option heading.
//  16. Aggregator media filters: With only images selected for the aggregator fixture,
//      PrintProviderInfo must omit video vendors and duration, retain the leading image vendor's
//      count, and show one valid image-model example key. It must retain the full catalog counts
//      and fit the page width without trailing whitespace.
//  17. Option scope notes: For the five-model fixture, PrintProviderInfo must omit a scope note for
//      an option every model declares identically. Other uniform options must name the smaller
//      group: models missing the option or models supporting it. The option text and scope notes
//      must appear in the specified order, with aligned continuations and no page-width or
//      trailing-whitespace violations.
//  18. Option group introductions: For differing parameter declarations, PrintProviderInfo must put
//      the largest group first under the remainder heading and name each smaller group; tied groups
//      must each name their models. Scope notes must follow group details and be absent when every
//      model supports the flag. The tested introductions must follow blank lines and sit between
//      the heading and details columns, with aligned detail continuations.
//  19. Model roster alias continuation: Given a model with eight long aliases, PrintProviderInfo
//      must place the first alias beside the model ID and include every alias in order across
//      continuation lines at the alias column. Every page line must fit the width without trailing
//      whitespace.
//  20. Prompt-ignored card line: PrintModelInfo must include CardPromptIgnored when the model's
//      PromptIgnored field is true and omit it otherwise. The card for the model that ignores
//      prompts must fit the page width without trailing whitespace.
//  21. Documentation address: Given a provider documentation URL, PrintProviderInfo must include it
//      for both standard providers and aggregators. PrintModelInfo must use a model's own URL when
//      present, otherwise inherit the provider's, and show no HTTPS address when neither declares
//      one. The card with a model URL must fit the page width without trailing whitespace.

// TestProviderUnconstrainedGroup verifies invariant #1: Unconstrained model explanation.
//
// What is being tested:
// Given models with unrestricted and restricted aspect ratios, PrintProviderInfo must show the
// unrestricted model's group heading followed by nonempty explanatory text before the restricted
// group's heading.
//
// Test class: Expanded.
// Test layer: Coverage.
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
		_ = printProviderInfoSelection(t, os.Stdout, &providerConfig, paramFlags, true, false, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
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

// TestProviderPageMediaFilters verifies invariant #2: Provider page media filters.
//
// What is being tested:
// With video selected, PrintProviderInfo must omit the fixture's image models and quality option,
// show the video model before its duration option, and retain the full catalog counts. With images
// selected, it must omit the video model and duration option and show an image model before the
// quality option.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestProviderPageMediaFilters(t *testing.T) {
	prov, models := providerFixture(t)
	flags := cardFlags(t)

	videoOnly := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, prov, models), flags, false, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})

	// The image roster and options go.
	if lineHolding(t, pageLines(t, videoOnly), "gam-one", "gone") >= 0 {
		t.Errorf("✗ the video filter kept gam-one:\n%s", videoOnly)
	}

	checkAbsent(t, "the video-filtered page", videoOnly, "gam-one", "gam-two", flagHeading(t, flags, params.FlagTypeQuality))
	checkInOrder(t, "the video-filtered page", videoOnly, "gam-vid", flagHeading(t, flags, params.FlagTypeDuration))
	checkCounts(t, "the video-filtered page", videoOnly, 4, 3, 1)

	imageOnly := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, prov, models), flags, true, false, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})

	checkAbsent(t, "the image-filtered page", imageOnly, "gam-vid", flagHeading(t, flags, params.FlagTypeDuration))
	checkInOrder(t, "the image-filtered page", imageOnly, "gam-one", flagHeading(t, flags, params.FlagTypeQuality))

	if !t.Failed() {
		t.Log("✓ the media filters narrow the model roster and options and leave the catalog counts whole")
	}
}

// TestPrintProviderInfo verifies invariant #3: Provider information page.
//
// What is being tested:
// Given the Alpha Labs fixture, PrintProviderInfo must include the provider name, ID, credential
// variable, model IDs, and aliases. It must place img-one before vid-one.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintProviderInfo(t *testing.T) {
	pairs := infoFixture(t)
	models := []catalog.Model{pairs[0].Model, pairs[1].Model, pairs[2].Model}

	view := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, pairs[0].Provider, models), infoFlags(t), true, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})
	for _, want := range []string{"Alpha Labs", "alpha", "ALPHA_API_KEY", "img-one", "img-three", "vid-one", "ione", "vone"} {
		if !strings.Contains(view, want) {
			t.Errorf("✗ the provider view lacks %q:\n%s", want, view)
		}
	}

	if imgAt, vidAt := strings.Index(view, "img-one"), strings.Index(view, "vid-one"); imgAt < 0 || vidAt < 0 || imgAt > vidAt {
		t.Errorf("✗ media order: img-one at %d, vid-one at %d, want image before video", imgAt, vidAt)
	}

	if !t.Failed() {
		t.Log("✓ the provider view carries the identity and the models separated by media with aliases")
	}
}

// TestPrintModelInfo verifies invariant #4: Model information page.
//
// What is being tested:
// Given the image and video fixtures, PrintModelInfo must show their IDs and media kinds, the image
// model's key and alias, and declared constraint values under the corresponding flag sections. The
// video duration section must include its rule description, and the image page must omit the tested
// provider request keys.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintModelInfo(t *testing.T) {
	pairs := infoFixture(t)

	view := captureStdout(t, func() {
		_ = PrintModelInfo(os.Stdout, &pairs[0], infoFlags(t), term.IsTerminal(int(os.Stdout.Fd())))
	})
	for _, want := range []string{"alpha/img-one", "ione"} {
		if !strings.Contains(view, want) {
			t.Errorf("✗ the model view lacks %q:\n%s", want, view)
		}
	}

	if !strings.Contains(strings.ToLower(view), string(media.Image)) {
		t.Errorf("✗ the image model view does not name its medium:\n%s", view)
	}

	sections := []struct {
		flagName, nextFlagName string
		data                   []string
	}{
		// The enumeration's record order fixes the section sequence.
		{"size", "aspect-ratio", []string{"64", "4194304"}},
		{"quality", "number-fixture", []string{"low", "high"}},
		{"number-fixture", "num-images", []string{"0", "2"}},
		{"num-images", "input-media", []string{"10"}},
		{"input-media", "", []string{"4"}},
	}
	for _, s := range sections {
		section := paramSection(t, view, s.flagName, s.nextFlagName)
		for _, want := range s.data {
			if !strings.Contains(section, want) {
				t.Errorf("✗ the %s section lacks %q: %q", s.flagName, want, section)
			}
		}
	}

	for _, wire := range []string{"quality_wire", "temp_wire", "count_wire", "aspect_wire"} {
		if strings.Contains(view, wire) {
			t.Errorf("✗ the wire spelling %q renders in the model view", wire)
		}
	}

	vidView := captureStdout(t, func() {
		_ = PrintModelInfo(os.Stdout, &pairs[2], infoFlags(t), term.IsTerminal(int(os.Stdout.Fd())))
	})
	if !strings.Contains(vidView, "vid-one") {
		t.Errorf("✗ the video model view lacks its model id:\n%s", vidView)
	}

	if !strings.Contains(strings.ToLower(vidView), string(media.Video)) {
		t.Errorf("✗ the video model view does not name its medium:\n%s", vidView)
	}

	durSection := paramSection(t, vidView, "duration", "")
	for _, want := range []string{"4", "6", "8", "forced to 8 on the fixture's test condition"} {
		if !strings.Contains(flattened(t, durSection), want) {
			t.Errorf("✗ the duration section lacks %q: %q", want, durSection)
		}
	}

	if !t.Failed() {
		t.Log("✓ the model view renders the declared constraint data and rule descriptions under flag names, no wire spellings")
	}
}

// TestPrintModelInfoBoundsAndAliases verifies invariant #5: Model bounds and aliases.
//
// What is being tested:
// Given a model with custom size bounds and a lone numeric minimum, PrintModelInfo must include the
// specified size values and 0.5 minimum in their parameter sections. PrintListing must place both
// model aliases beside the model ID on one line.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintModelInfoBoundsAndAliases(t *testing.T) {
	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	prov := catalog.Provider{ID: "gamma", DisplayName: "Gamma", APIKeyEnvVar: "GAMMA_API_KEY"}
	model := catalog.Model{ID: "img-full", Media: media.Image, Aliases: []string{"gfull", "gf"}, Params: []params.Definition{
		{FlagID: params.FlagTypeSize, CustomSize: &params.SizeBounds{
			MaxRatio: 3, MinEdge: 16, MaxEdge: 3840, MinPx: 655360, MaxPx: 8294400, EdgeIncrem: 16, LongEdge: 1536,
		}},
		{FlagID: flagTypeNumberFixture, MinValue: params.GetSetIf(true, 0.5)},
	}}

	view := captureStdout(t, func() {
		_ = PrintModelInfo(os.Stdout, &catalog.ProvModelPair{Provider: prov, Model: model}, infoFlags(t), term.IsTerminal(int(os.Stdout.Fd())))
	})

	sizeSection := paramSection(t, view, "size", "number-fixture")
	for _, want := range []string{"3:1", "16", "3840", "655360", "8294400", "1536"} {
		if !strings.Contains(sizeSection, want) {
			t.Errorf("✗ the size-bounds line lacks %q: %q", want, sizeSection)
		}
	}

	numberSection := paramSection(t, view, "number-fixture", "")
	if !strings.Contains(numberSection, "0.5") {
		t.Errorf("✗ the lone minimum bound is missing: %q", numberSection)
	}

	listing := captureStdout(t, func() {
		_ = PrintListing(os.Stdout, ListingPage(catalog.SelectMedia([]catalog.ProvModelPair{{Provider: prov, Model: model}}, true, true), true, true), true, true, false)
	})

	if lineHolding(t, pageLines(t, listing), "img-full", "gfull", "gf") < 0 {
		t.Errorf("✗ the listing does not carry both alias tokens beside the model:\n%s", listing)
	}

	if !t.Failed() {
		t.Log("✓ every declared size bound, a lone minimum, and plural aliases render")
	}
}

// TestModelComments verifies invariant #6: Model comments.
//
// What is being tested:
// Given an input-media parameter with ModelInfoComment, PrintModelInfo must include that comment
// after the flag's ordinary description in the input-media option.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestModelComments(t *testing.T) {
	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	prov := catalog.Provider{ID: "delta", DisplayName: "Delta", APIKeyEnvVar: "DELTA_API_KEY"}
	model := catalog.Model{ID: "vid-frames", Name: "Vid Frames", Media: media.Video, Params: []params.Definition{
		{
			FlagID:           params.FlagTypeInputMedia,
			ModelInfoComment: "Fixture keyframe guidance in full.",
		},
		{FlagID: params.FlagTypeDuration, MaxValue: params.GetSetIf(true, 10.0)},
	}}
	pairs := []catalog.ProvModelPair{{Provider: prov, Model: model}}
	flags := infoFlags(t)

	view := captureStdout(t, func() {
		_ = PrintModelInfo(os.Stdout, &pairs[0], flags, term.IsTerminal(int(os.Stdout.Fd())))
	})

	checkOption(t, pageLines(t, view), flagHeading(t, flags, params.FlagTypeInputMedia), "File path or web URL to reference images.", "Fixture keyframe guidance in full.")

	if !t.Failed() {
		t.Log("✓ the modelInfoComment renders on the card after its option's description")
	}
}

// TestPrintProviderInfoOptions verifies invariant #7: Provider option inventory.
//
// What is being tested:
// Given the Gamma Systems image-only fixture, PrintProviderInfo must include its display name and
// credential variable, show the aspect-ratio, quality, size, and num-images options, and omit
// duration.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintProviderInfoOptions(t *testing.T) {
	prov, models := providerPageFixture(t)

	page := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, prov, models), infoFlags(t), true, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})
	for _, want := range []string{"Gamma Systems", "GAMMA_API_KEY"} {
		if !strings.Contains(page, want) {
			t.Errorf("✗ the page lacks %q:\n%s", want, page)
		}
	}

	// Every flag any model of the medium declares appears; no other does.
	for _, longForm := range []string{"aspect-ratio", "quality", "size", "num-images"} {
		if !strings.Contains(page, "--"+longForm) {
			t.Errorf("✗ the options section lacks --%s", longForm)
		}
	}

	// The provider declares no video model: the video-only flag renders nowhere.
	if strings.Contains(page, "--duration") {
		t.Errorf("✗ the page carries a flag no model of the provider declares")
	}

	if !t.Failed() {
		t.Log("✓ the provider page carries every declared flag of its medium and no other")
	}
}

// TestPrintProviderInfoOptionSplit verifies invariant #8: Provider option grouping.
//
// What is being tested:
// Given two models with identical quality values, different size values, and only one with
// num-images, PrintProviderInfo must show quality constraints without model names, show each size
// value with its model in catalog order, and name only the supporting model under num-images.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintProviderInfoOptionSplit(t *testing.T) {
	prov, models := providerPageFixture(t)

	page := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, prov, models), infoFlags(t), true, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})

	// Both models declare quality with the same allowed values: unattributed.
	qualityBlock := optionBlock(t, page, "quality")
	if !strings.Contains(qualityBlock, fmt.Sprintf(AllowedValues, "low, high")) {
		t.Errorf("✗ the quality block lacks its constraints:\n%s", qualityBlock)
	}

	for _, modelName := range []string{"gam-one", "gam-two"} {
		if strings.Contains(qualityBlock, modelName) {
			t.Errorf("✗ a uniformly declared option names %s:\n%s", modelName, qualityBlock)
		}
	}

	// Both declare size, with differing allowed values: one group each.
	sizeBlock := optionBlock(t, page, "size")
	for _, want := range []string{"gam-one", "1024x1024", "gam-two", "512x512"} {
		if !strings.Contains(sizeBlock, want) {
			t.Errorf("✗ the split size block lacks %q:\n%s", want, sizeBlock)
		}
	}

	if strings.Index(sizeBlock, "gam-one") > strings.Index(sizeBlock, "gam-two") {
		t.Errorf("✗ the size groups are not in the models' first-occurrence order:\n%s", sizeBlock)
	}

	// Only the first model declares num-images: the group names it.
	countBlock := optionBlock(t, page, "num-images")
	if !strings.Contains(countBlock, "gam-one") {
		t.Errorf("✗ a partially declared option does not name its model:\n%s", countBlock)
	}

	if strings.Contains(countBlock, "gam-two") {
		t.Errorf("✗ a partially declared option names a model that does not declare it:\n%s", countBlock)
	}

	if !t.Failed() {
		t.Log("✓ uniform options render unattributed and differing ones name the models each covers")
	}
}

// TestPrintProviderInfoLayout verifies invariant #9: Print provider info layout.
//
// What is being tested:
// Given the provider-page fixture, PrintProviderInfo must write every line within the page width
// and leave no trailing spaces or tabs.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintProviderInfoLayout(t *testing.T) {
	prov, models := providerPageFixture(t)

	page := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, prov, models), infoFlags(t), true, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})
	checkPageLayout(t, "the provider page", page)

	if !t.Failed() {
		t.Log("✓ the provider page stays within the page width with no trailing whitespace")
	}
}

// TestModelCard verifies invariant #10: Model card.
//
// What is being tested:
// Given the full image-model fixture, PrintModelInfo must show model and provider names and IDs,
// the model key, credential variable, aliases, and medium. It must order option headings by flag
// records, align continuation lines, and place each option's descriptions, comments, examples,
// constraints, and required mark in the specified order. The page must fit the width without
// trailing whitespace and omit the tested request keys and model description.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestModelCard(t *testing.T) {
	pair := cardFixture(t)
	flags := cardFlags(t)

	page := captureStdout(t, func() {
		_ = PrintModelInfo(os.Stdout, &pair, flags, term.IsTerminal(int(os.Stdout.Fd())))
	})
	checkPageLayout(t, "the model card", page)

	lines := pageLines(t, page)
	for _, identity := range []string{"Image One", "img-one", "alpha/img-one", "ALPHA_API_KEY"} {
		if lineHolding(t, lines, identity) < 0 {
			t.Errorf("✗ the card's identity block lacks %q:\n%s", identity, page)
		}
	}

	if lineHolding(t, lines, "Alpha Labs", "alpha") < 0 {
		t.Errorf("✗ the card does not name the provider with its ID on one line:\n%s", page)
	}

	if at := lineHolding(t, lines, "img-one"); at < 0 || strings.Contains(lines[at], "alpha/") {
		t.Errorf("✗ the card does not carry the bare model ID by itself:\n%s", page)
	}

	if lineHolding(t, lines, "ione", "one") < 0 {
		t.Errorf("✗ the card carries no line with both alias tokens:\n%s", page)
	}

	if !strings.Contains(strings.ToLower(page), string(media.Image)) {
		t.Errorf("✗ the card does not name its medium:\n%s", page)
	}

	expectedOptions := []struct {
		flag   params.FlagType
		tokens []string
	}{
		{params.FlagTypeSize, []string{"Dimensions in width and height.", "Supersedes --aspect-ratio and --resolution.", "1024x768", "64", "4194304", "1024"}},
		{params.FlagTypeAspect, []string{"A value in <x:y> format.", "Superseded by an explicit --size value when supported.", "16:9", "2:3"}},
		{params.FlagTypeQuality, []string{"Image quality level.", "Fixture quality note.", "low", "high"}},
		{flagTypeNumberFixture, []string{"A number fixture.", "0", "2"}},
		{params.FlagTypeImageN, []string{"Number of images to generate.", "10"}},
		{params.FlagTypeInputMedia, []string{"File path or URL to an image or video.", "Repeat the flag for URLs.", "4", "Fixture keyframe guidance in full."}},
		{params.FlagTypeThoughts, []string{"Write model thought traces to a markdown file beside the artifact."}},
		{"seed", []string{RequiredLabel, "Sampling seed.", "Exact repetition is not guaranteed."}},
		{"disable-prompt-upsampling", []string{"Turn off the provider's automatic prompt expansion."}},
	}

	headings := make([]string, 0, len(expectedOptions))
	for _, expectedOption := range expectedOptions {
		headings = append(headings, flagHeading(t, flags, expectedOption.flag))
	}

	checkInOrder(t, "the card's option sequence", page, headings...)
	checkOptionsAligned(t, "the model card", lines, headings...)

	for i, expectedOption := range expectedOptions {
		checkOption(t, lines, headings[i], expectedOption.tokens...)
	}

	checkAbsent(t, "the card", page, "aspect_wire", "quality_wire", "temp_wire", "count_wire", "seed_wire", "disable_pup", "A fixture image model.")

	if !t.Failed() {
		t.Log("✓ the model card renders the identity block and every option's details in order at one column")
	}
}

// TestModelCardIdentityVariants verifies invariant #11: Model card identity variants.
//
// What is being tested:
// Given a video model with allowed durations and a rule description, PrintModelInfo must show its
// name, ID, fully qualified key, credential variable, and video medium. Its duration option must
// show the description, allowed values, and rule in that order, and the page must fit the width
// without trailing whitespace.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestModelCardIdentityVariants(t *testing.T) {
	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	pair := catalog.ProvModelPair{
		Provider: catalog.Provider{ID: "alpha", DisplayName: "Alpha Labs", APIKeyEnvVar: "ALPHA_API_KEY"},
		Model: catalog.Model{ID: "vid-one", Name: "Video One", Media: media.Video, Params: []params.Definition{
			{FlagID: params.FlagTypeDuration, AllowedValues: []string{"4", "6", "8"}, RuleDescription: "forced to 8 on the fixture's test condition"},
		}},
	}
	flags := cardFlags(t)

	page := captureStdout(t, func() {
		_ = PrintModelInfo(os.Stdout, &pair, flags, term.IsTerminal(int(os.Stdout.Fd())))
	})
	checkPageLayout(t, "the video card", page)

	lines := pageLines(t, page)
	for _, identity := range []string{"Video One", "vid-one", "alpha/vid-one", "ALPHA_API_KEY"} {
		if lineHolding(t, lines, identity) < 0 {
			t.Errorf("✗ the video card's identity block lacks %q:\n%s", identity, page)
		}
	}

	if !strings.Contains(strings.ToLower(page), string(media.Video)) {
		t.Errorf("✗ the video card does not name its medium:\n%s", page)
	}

	checkOption(t, lines, flagHeading(t, flags, params.FlagTypeDuration), "Video length in seconds.", "4", "6", "8", "forced to 8 on the fixture's test condition")

	if !t.Failed() {
		t.Log("✓ a video card names its medium and key and renders the rule description after the values")
	}
}

// TestProviderPage verifies invariant #12: Provider page.
//
// What is being tested:
// Given three image models and one video model, PrintProviderInfo must show provider identity and
// full catalog counts, then bare model IDs and aliases in catalog order with images before video.
// It must order each medium's options by flag records, repeat aspect-ratio once per medium, and
// omit fully qualified example keys. Every line must fit the page width without trailing
// whitespace.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestProviderPage(t *testing.T) {
	prov, models := providerFixture(t)
	flags := cardFlags(t)

	page := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, prov, models), flags, true, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})
	checkPageLayout(t, "the provider page", page)

	lines := pageLines(t, page)
	for _, identity := range []string{"Gamma Systems", "gamma", "GAMMA_API_KEY"} {
		if lineHolding(t, lines, identity) < 0 {
			t.Errorf("✗ the identity block lacks %q:\n%s", identity, page)
		}
	}

	checkCounts(t, "the provider page", page, 4, 3, 1)

	checkInOrder(t, "the model roster", page, "gam-one", "gam-two", "gam-long-identifier-exceeding-thirty-two", "glong", "gam-vid")

	if lineHolding(t, lines, "gam-one", "gone") < 0 || !slices.Contains(lines, "gam-two") {
		t.Errorf("✗ the model roster are not the bare IDs with the aliases beside them:\n%s", page)
	}

	firstHeading := slices.IndexFunc(lines, func(pageLine string) bool { return strings.HasPrefix(pageLine, "-") })
	if firstHeading < 0 || strings.Contains(strings.Join(lines[:firstHeading], "\n"), "gamma/") {
		t.Errorf("✗ the model roster carry the provider prefix:\n%s", page)
	}

	sizeHeading := flagHeading(t, flags, params.FlagTypeSize)
	aspectHeading := flagHeading(t, flags, params.FlagTypeAspect)
	checkInOrder(t, "the option sections", page,
		"gam-vid",
		sizeHeading, aspectHeading, flagHeading(t, flags, params.FlagTypeQuality), flagHeading(t, flags, params.FlagTypeImageN),
		aspectHeading, flagHeading(t, flags, params.FlagTypeDuration))

	if got := strings.Count(page, aspectHeading); got != 2 {
		t.Errorf("✗ the aspect-ratio option renders %d times, want once per medium section", got)
	}

	// Standard provider pages omit usage examples.
	if keys := pageKeys(t, page, prov.ID); len(keys) != 0 {
		t.Errorf("✗ the page names %v as example keys, want none", keys)
	}

	if !t.Failed() {
		t.Log("✓ the provider page renders the identity, the bare model roster, and the option sections per medium")
	}
}

// TestProviderOptionGroups verifies invariant #13: Provider option groups.
//
// What is being tested:
// Given shared and differing parameter declarations, PrintProviderInfo must show shared constraints
// without model names, identify models missing an option, and group differing constraints under the
// applicable model names. It must preserve the specified guidance and rule order, separate group
// introductions from details, and omit the model-specific expanded comment.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestProviderOptionGroups(t *testing.T) {
	prov, models := providerFixture(t)
	flags := cardFlags(t)

	page := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, prov, models), flags, true, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})

	sizeHeading := flagHeading(t, flags, params.FlagTypeSize)
	aspectHeading := flagHeading(t, flags, params.FlagTypeAspect)

	// The image section opens with its first option; the video section opens with the second
	// rendering of the aspect-ratio option.
	imageStart := strings.Index(page, sizeHeading)
	videoStart := strings.LastIndex(page, aspectHeading)

	if imageStart < 0 || videoStart <= imageStart {
		t.Fatalf("💣 the page lacks the image options before the video options:\n%s", page)
	}

	imageSection := pageLines(t, page[imageStart:videoStart])
	videoSection := pageLines(t, page[videoStart:])

	longID := "gam-long-identifier-exceeding-thirty-two"

	checkOption(t, imageSection, aspectHeading, "A value in <x:y> format.", "Superseded by an explicit --size value when supported.", "16:9", "2:3")
	checkAbsent(t, "the uniform aspect-ratio option", optionDetails(t, imageSection, aspectHeading), "gam-one", "gam-two", "gam-long", GroupIntroRemainder)

	qualityHeading := flagHeading(t, flags, params.FlagTypeQuality)
	checkOption(t, imageSection, qualityHeading, "Image quality level.", "Fixture quality note.", "low", "high", fmt.Sprintf(NotUsedBy, longID))
	checkAbsent(t, "the quality option", optionDetails(t, imageSection, qualityHeading), "gam-one", "gam-two", GroupIntroRemainder)

	checkOption(t, imageSection, sizeHeading, "Dimensions in width and height.", "Supersedes --aspect-ratio and --resolution.", "1024x768",
		fmt.Sprintf(GroupIntro, "gam-one"), "1024x1024", fmt.Sprintf(GroupIntro, "gam-two"), "512x512", "768x768", fmt.Sprintf(NotUsedBy, longID))
	checkAbsent(t, "the size option", optionDetails(t, imageSection, sizeHeading), GroupIntroRemainder)
	checkGroupIntro(t, optionText(t, imageSection, sizeHeading), fmt.Sprintf(GroupIntro, "gam-two")+":")

	checkOption(t, imageSection, flagHeading(t, flags, params.FlagTypeImageN), "Number of images to generate.",
		fmt.Sprintf(GroupIntro, "gam-one"), "4", fmt.Sprintf(GroupIntro, longID), "9", fmt.Sprintf(NotUsedBy, "gam-two"))

	checkOption(t, videoSection, flagHeading(t, flags, params.FlagTypeDuration), "Video length in seconds.", "4", "6", "8", "forced to 8 on the fixture's test condition")

	checkAbsent(t, "the provider page", page, "Gamma One aspect guidance")

	if !t.Failed() {
		t.Log("✓ uniform options render as one option with a scope note and differing ones as introduced groups")
	}
}

// TestAggregatorSummary verifies invariant #14: Aggregator summary.
//
// What is being tested:
// Given a marked aggregator, PrintProviderInfo must show its identity and full catalog counts,
// vendors and their counts in the specified count-then-name order, and flag names in record order
// for each medium. Its footer must name the leading vendors and one valid model key per medium, and
// it must omit full option headings. The page must fit the width without trailing whitespace.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAggregatorSummary(t *testing.T) {
	prov, models := aggregatorFixture(t)
	flags := cardFlags(t)

	page := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, prov, models), flags, true, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})
	checkPageLayout(t, "the aggregator summary", page)

	lines := pageLines(t, page)
	for _, identity := range []string{"Gateway", "gate", "GATE_API_KEY"} {
		if lineHolding(t, lines, identity) < 0 {
			t.Errorf("✗ the summary's identity block lacks %q:\n%s", identity, page)
		}
	}

	catalogCountsText, _, ok := strings.Cut(page, "vendor-one")
	if !ok {
		t.Fatalf("💣 the summary names no vendor:\n%s", page)
	}

	checkInOrder(t, "the catalog counts", catalogCountsText, "8", "5", "3")

	checkInOrder(t, "the vendor counts", page, "vendor-one", "solo", "vendor-three", "vendor-two", "bytedance", "alibaba")

	for vendor, count := range map[string]string{"vendor-one": "2", "vendor-two": "1", "vendor-three": "1", "solo": "1", "bytedance": "2", "alibaba": "1"} {
		if lineHolding(t, lines, vendor, count) < 0 {
			t.Errorf("✗ the vendor %s renders without its count %s:\n%s", vendor, count, page)
		}
	}

	flagsStart := strings.Index(page, "alibaba")
	checkInOrder(t, "the flag names per medium", page[flagsStart:], "--size", "--aspect-ratio", "--quality", "--seed", "--aspect-ratio", "--duration", "--seed")

	footerStart := strings.LastIndex(page, "--seed")
	checkInOrder(t, "the footer", page[footerStart:], "vendor-one", "bytedance")

	if keys := checkKeysOfProvider(t, "the summary", page, prov, models); len(keys) != 2 {
		t.Errorf("✗ the summary names %v, want one example key per medium and no roster", keys)
	}

	for i := range flags {
		checkAbsent(t, "the summary", page, optionHeading(&flags[i]))
	}

	if !t.Failed() {
		t.Log("✓ the aggregator summary renders the counts, vendor counts, flag names, and footer without a roster or options")
	}
}

// TestAggregatorMarkedOnly verifies invariant #15: Aggregator marked only.
//
// What is being tested:
// Given an unmarked provider with 21 models, PrintProviderInfo must list every bare model ID.
// Marking the same provider as an aggregator must produce one valid image-model example key and
// omit the full aspect-ratio option heading.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAggregatorMarkedOnly(t *testing.T) {
	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	prov := catalog.Provider{ID: "wide", DisplayName: "Wide", APIKeyEnvVar: "WIDE_API_KEY"}
	flags := cardFlags(t)

	models := make([]catalog.Model, 0, 21)
	for i := range 21 {
		models = append(models, catalog.Model{ID: fmt.Sprintf("vendor-%d/model-%d", i%3, i), Name: "Model", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeAspect}}})
	}

	standard := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, prov, models), flags, true, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})

	standardLines := pageLines(t, standard)
	for _, model := range models {
		if !slices.Contains(standardLines, model.ID) {
			t.Errorf("✗ the standard page of the unmarked provider lacks %s", model.ID)
		}
	}

	marked := prov
	marked.Aggregator = true

	summary := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, marked, models), flags, true, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})

	checkAbsent(t, "the marked provider's summary", summary, flagHeading(t, flags, params.FlagTypeAspect))

	if keys := checkKeysOfProvider(t, "the marked provider's summary", summary, marked, models); len(keys) != 1 {
		t.Errorf("✗ the summary names %v, want one example key and no roster", keys)
	}

	if !t.Failed() {
		t.Log("✓ the aggregator mark alone selects the summary; an unmarked provider renders the standard page whatever its model count")
	}
}

// TestAggregatorMediaFilters verifies invariant #16: Aggregator media filters.
//
// What is being tested:
// With only images selected for the aggregator fixture, PrintProviderInfo must omit video vendors
// and duration, retain the leading image vendor's count, and show one valid image-model example
// key. It must retain the full catalog counts and fit the page width without trailing whitespace.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAggregatorMediaFilters(t *testing.T) {
	prov, models := aggregatorFixture(t)

	imageOnly := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, prov, models), cardFlags(t), true, false, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})
	checkPageLayout(t, "the filtered summary", imageOnly)

	checkAbsent(t, "the image-filtered summary", imageOnly, "bytedance", "alibaba", "--duration")

	lines := pageLines(t, imageOnly)
	if lineHolding(t, lines, "vendor-one", "2") < 0 {
		t.Errorf("✗ the image filter dropped the top image vendor's count:\n%s", imageOnly)
	}

	if keys := checkKeysOfProvider(t, "the image-filtered summary", imageOnly, prov, modelsOfMedia(models, media.Image)); len(keys) != 1 {
		t.Errorf("✗ the image filter's example keys %v are not one image model:\n%s", keys, imageOnly)
	}

	catalogCountsText, _, _ := strings.Cut(imageOnly, "vendor-one")
	checkInOrder(t, "the catalog counts under the image filter", catalogCountsText, "8", "5", "3")

	if !t.Failed() {
		t.Log("✓ the summary's sections follow the media filters while the catalog counts stay whole")
	}
}

// TestScopeNotes verifies invariant #17: Option scope notes.
//
// What is being tested:
// For the five-model fixture, PrintProviderInfo must omit a scope note for an option every model
// declares identically. Other uniform options must name the smaller group: models missing the
// option or models supporting it. The option text and scope notes must appear in the specified
// order, with aligned continuations and no page-width or trailing-whitespace violations.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestScopeNotes(t *testing.T) {
	prov, models := scopeFixture(t)
	flags := cardFlags(t)

	page := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, prov, models), flags, true, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})
	checkPageLayout(t, "the scope page", page)

	lines := pageLines(t, page)

	allHeading := flagHeading(t, flags, "disable-prompt-upsampling")
	checkOption(t, lines, allHeading, "Turn off the provider's automatic prompt expansion.")
	checkAbsent(t, "the option every model declares", optionDetails(t, lines, allHeading), "m1", "m5", strings.TrimSuffix(NotUsedBy, " %s."), strings.TrimSuffix(OnlyForModels, " %s."), GroupIntroRemainder)

	mostHeading := flagHeading(t, flags, "seed")
	checkOption(t, lines, mostHeading, "Sampling seed.", "Exact repetition is not guaranteed.", fmt.Sprintf(NotUsedBy, "m5"))
	checkAbsent(t, "the option most models declare", optionDetails(t, lines, mostHeading), "m1", "m2", "m3", "m4", strings.TrimSuffix(OnlyForModels, " %s."))

	pairHeading := flagHeading(t, flags, params.FlagTypeInputMedia)
	checkOption(t, lines, pairHeading, "File path or URL to an image or video.", "2", fmt.Sprintf(NotUsedBy, fmt.Sprintf(ListPairOr, "m4", "m5")))

	fewHeading := flagHeading(t, flags, params.FlagTypeQuality)
	checkOption(t, lines, fewHeading, "Image quality level.", "low", "high", fmt.Sprintf(OnlyForModels, fmt.Sprintf(ListPairAnd, "m1", "m2")))
	checkAbsent(t, "the option few models declare", optionDetails(t, lines, fewHeading), "m3", "m4", "m5", strings.TrimSuffix(NotUsedBy, " %s."))

	checkOptionsAligned(t, "the scope page", lines, allHeading, mostHeading, pairHeading, fewHeading)

	if !t.Failed() {
		t.Log("✓ an option declared alike carries no note, the missing models, or the declaring models, whichever list is shorter")
	}
}

// TestGroupIntros verifies invariant #18: Option group introductions.
//
// What is being tested:
// For differing parameter declarations, PrintProviderInfo must put the largest group first under
// the remainder heading and name each smaller group; tied groups must each name their models. Scope
// notes must follow group details and be absent when every model supports the flag. The tested
// introductions must follow blank lines and sit between the heading and details columns, with
// aligned detail continuations.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestGroupIntros(t *testing.T) {
	prov, models := scopeFixture(t)
	flags := cardFlags(t)

	page := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, prov, models), flags, true, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})

	lines := pageLines(t, page)

	sizeHeading := flagHeading(t, flags, params.FlagTypeSize)
	checkOption(t, lines, sizeHeading, "Dimensions in width and height.", GroupIntroRemainder, "1024x1024", fmt.Sprintf(GroupIntro, "m4"), "512x512", fmt.Sprintf(NotUsedBy, "m5"))
	checkAbsent(t, "the size option", optionDetails(t, lines, sizeHeading), "m1", "m2", "m3")
	checkGroupIntro(t, optionText(t, lines, sizeHeading), GroupIntroRemainder+":")
	checkGroupIntro(t, optionText(t, lines, sizeHeading), fmt.Sprintf(GroupIntro, "m4")+":")

	countHeading := flagHeading(t, flags, params.FlagTypeImageN)
	checkOption(t, lines, countHeading, "Number of images to generate.", fmt.Sprintf(GroupIntro, "m1"), "4", fmt.Sprintf(GroupIntro, "m2"), "9", fmt.Sprintf(OnlyForModels, fmt.Sprintf(ListPairAnd, "m1", "m2")))
	checkAbsent(t, "the tied option", optionDetails(t, lines, countHeading), GroupIntroRemainder, "m3", "m4", "m5")

	numberHeading := flagHeading(t, flags, flagTypeNumberFixture)
	checkOption(t, lines, numberHeading, "A number fixture.", GroupIntroRemainder, "0", "1", fmt.Sprintf(GroupIntro, fmt.Sprintf(ListPairAnd, "m4", "m5")), "0", "2")
	checkAbsent(t, "the option every model declares in two ways", optionDetails(t, lines, numberHeading), "m1", "m2", "m3", strings.TrimSuffix(NotUsedBy, " %s."), strings.TrimSuffix(OnlyForModels, " %s."))

	aspectHeading := flagHeading(t, flags, params.FlagTypeAspect)
	checkOption(t, lines, aspectHeading, "A value in <x:y> format.", GroupIntroRemainder, "1:1", fmt.Sprintf(GroupIntro, "m1"))
	checkAbsent(t, "the option whose largest group comes second", optionDetails(t, lines, aspectHeading), "m2", "m3", "m4", "m5")

	checkOptionsAligned(t, "the group page", lines, sizeHeading, countHeading, numberHeading, aspectHeading)

	if !t.Failed() {
		t.Log("✓ the largest group leads in the remainder form, the others are named, ties are all named, and the scope note follows")
	}
}

// TestModelRosterAliasContinuation verifies invariant #19: Model roster alias continuation.
//
// What is being tested:
// Given a model with eight long aliases, PrintProviderInfo must place the first alias beside the
// model ID and include every alias in order across continuation lines at the alias column. Every
// page line must fit the width without trailing whitespace.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestModelRosterAliasContinuation(t *testing.T) {
	prov, models := providerFixture(t)
	flags := cardFlags(t)

	manyAliases := []string{
		"alias-number-one", "alias-number-two", "alias-number-three", "alias-number-four",
		"alias-number-five", "alias-number-six", "alias-number-seven", "alias-number-eight",
	}
	models[0].Aliases = manyAliases

	page := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, prov, models), flags, true, false, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})
	checkPageLayout(t, "the provider page with a wide alias list", page)

	lines := pageLines(t, page)

	rosterIndex := lineHolding(t, lines, models[0].ID, manyAliases[0])
	if rosterIndex < 0 {
		t.Fatalf("💣 nothing carries %s with its first alias:\n%s", models[0].ID, page)
	}

	continuationIndent := strings.Repeat(" ", strings.Index(lines[rosterIndex], manyAliases[0]))

	rosterText := []string{lines[rosterIndex]}
	for lineIndex := rosterIndex + 1; lineIndex < len(lines) && strings.HasPrefix(lines[lineIndex], continuationIndent); lineIndex++ {
		rosterText = append(rosterText, strings.TrimSpace(lines[lineIndex]))
	}

	if len(rosterText) == 1 {
		t.Errorf("✗ the wide alias list has no continuation line at the alias column:\n%s", page)
	}

	checkInOrder(t, "the flowed alias list", strings.Join(rosterText, " "), manyAliases...)

	if !t.Failed() {
		t.Log("✓ a wide alias list continues at the alias column with every alias in order")
	}
}

// TestModelCardPromptIgnored verifies invariant #20: Prompt-ignored card line.
//
// What is being tested:
// PrintModelInfo must include CardPromptIgnored when the model's PromptIgnored field is true and
// omit it otherwise. The card for the model that ignores prompts must fit the page width without
// trailing whitespace.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestModelCardPromptIgnored(t *testing.T) {
	flags := cardFlags(t)

	reading := cardFixture(t)
	readingCard := captureStdout(t, func() {
		_ = PrintModelInfo(os.Stdout, &reading, flags, term.IsTerminal(int(os.Stdout.Fd())))
	})

	if strings.Contains(readingCard, CardPromptIgnored) {
		t.Errorf("✗ a prompt-reading model's card states the prompt is ignored:\n%s", readingCard)
	}

	ignoring := cardFixture(t)
	ignoring.Model.PromptIgnored = true
	ignoringCard := captureStdout(t, func() {
		_ = PrintModelInfo(os.Stdout, &ignoring, flags, term.IsTerminal(int(os.Stdout.Fd())))
	})
	checkPageLayout(t, "the prompt-ignored card", ignoringCard)

	if lineHolding(t, pageLines(t, ignoringCard), CardPromptIgnored) < 0 {
		t.Errorf("✗ a prompt-ignored model's card lacks the prompt line:\n%s", ignoringCard)
	}

	if !t.Failed() {
		t.Log("✓ the card states that the prompt is ignored exactly for a model configured so")
	}
}

// TestDocsURLLines verifies invariant #21: Documentation address.
//
// What is being tested:
// Given a provider documentation URL, PrintProviderInfo must include it for both standard providers
// and aggregators. PrintModelInfo must use a model's own URL when present, otherwise inherit the
// provider's, and show no HTTPS address when neither declares one. The card with a model URL must
// fit the page width without trailing whitespace.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestDocsURLLines(t *testing.T) {
	flags := cardFlags(t)
	providerAddress := "https://docs.example.test/alpha"
	modelAddress := "https://models.example.test/img-one"

	prov, models := providerFixture(t)
	prov.DocsURL = providerAddress

	providerPage := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, prov, models), flags, true, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})
	if lineHolding(t, pageLines(t, providerPage), providerAddress) < 0 {
		t.Errorf("✗ the provider page lacks the documentation address:\n%s", providerPage)
	}

	aggregator, aggregatorModels := aggregatorFixture(t)
	aggregator.DocsURL = providerAddress

	summary := captureStdout(t, func() {
		_ = printProviderInfoSelection(t, os.Stdout, providerWith(t, aggregator, aggregatorModels), flags, true, true, map[media.Kind]string{media.Image: "image", media.Video: "video"}, term.IsTerminal(int(os.Stdout.Fd())))
	})
	if lineHolding(t, pageLines(t, summary), providerAddress) < 0 {
		t.Errorf("✗ the aggregator summary lacks the documentation address:\n%s", summary)
	}

	pair := cardFixture(t)
	pair.Provider.DocsURL = providerAddress
	inheritedCard := captureStdout(t, func() {
		_ = PrintModelInfo(os.Stdout, &pair, flags, term.IsTerminal(int(os.Stdout.Fd())))
	})

	if lineHolding(t, pageLines(t, inheritedCard), providerAddress) < 0 {
		t.Errorf("✗ a card whose model declares no address lacks the provider's:\n%s", inheritedCard)
	}

	pair.Model.DocsURL = modelAddress
	ownCard := captureStdout(t, func() {
		_ = PrintModelInfo(os.Stdout, &pair, flags, term.IsTerminal(int(os.Stdout.Fd())))
	})
	checkPageLayout(t, "the card with a model address", ownCard)

	if lines := pageLines(t, ownCard); lineHolding(t, lines, modelAddress) < 0 || lineHolding(t, lines, providerAddress) >= 0 {
		t.Errorf("✗ a card whose model declares an address does not show that address alone:\n%s", ownCard)
	}

	bare := cardFixture(t)
	bareCard := captureStdout(t, func() {
		_ = PrintModelInfo(os.Stdout, &bare, flags, term.IsTerminal(int(os.Stdout.Fd())))
	})

	if strings.Contains(bareCard, "https://") {
		t.Errorf("✗ a card with no declared address shows one:\n%s", bareCard)
	}

	if !t.Failed() {
		t.Log("✓ each page shows the declared documentation address, the card preferring the model's own")
	}
}

// flagTypeNumberFixture is a synthetic number-typed parameter the fixture enumerations declare.
const flagTypeNumberFixture = "number-fixture"

// numericToken matches a token made only of digits and dots; such a token is searched as a whole
// word, so a bound of 4 never matches inside 4194304.
var numericToken = regexp.MustCompile(`^[0-9.]+$`)

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

// printProviderInfoSelection supplies the catalog-selected pairs to the renderer.
func printProviderInfoSelection(test testing.TB, destination io.Writer, provider *catalog.Provider, flags []params.Flag, imageSelected, videoSelected bool, mediaFlags map[media.Kind]string, styled bool) error {
	test.Helper()

	pairs := pairsOf(test, *provider, provider.Models)
	selected := catalog.SelectMedia(pairs, imageSelected, videoSelected)

	return PrintProviderInfo(destination, provider, selected, flags, mediaFlags, styled)
}

// pairsOf takes a provider and its models and returns their provider-model pairs, the form the
// search takes.
func pairsOf(test testing.TB, prov catalog.Provider, models []catalog.Model) []catalog.ProvModelPair {
	test.Helper()

	pairs := make([]catalog.ProvModelPair, 0, len(models))
	for i := range models {
		pairs = append(pairs, catalog.ProvModelPair{Provider: prov, Model: models[i]})
	}

	return pairs
}

// tokenIndex takes text, a token, and a start offset and returns the offset of the token's first
// occurrence at or after the start, or -1.
func tokenIndex(test testing.TB, text, token string, start int) int {
	test.Helper()

	rest := text[start:]

	if numericToken.MatchString(token) {
		loc := regexp.MustCompile(`\b` + regexp.QuoteMeta(token) + `\b`).FindStringIndex(rest)
		if loc == nil {
			return -1
		}

		return start + loc[0]
	}

	at := strings.Index(rest, token)
	if at < 0 {
		return -1
	}

	return start + at
}

// checkInOrder asserts that every token appears in the text, each after the one before it.
func checkInOrder(t *testing.T, label, text string, tokens ...string) {
	t.Helper()

	at := 0

	for _, token := range tokens {
		next := tokenIndex(t, text, token, at)
		if next < 0 {
			t.Errorf("✗ %s lacks %q after offset %d:\n%s", label, token, at, text)

			return
		}

		at = next + len(token)
	}
}

// checkAbsent asserts that none of the tokens appears in the text.
func checkAbsent(t *testing.T, label, text string, tokens ...string) {
	t.Helper()

	for _, token := range tokens {
		if tokenIndex(t, text, token, 0) >= 0 {
			t.Errorf("✗ %s carries %q, which must not render:\n%s", label, token, text)
		}
	}
}

// lineHolding returns the index of the first line containing every supplied token, or -1 if no line
// matches.
func lineHolding(test testing.TB, lines []string, tokens ...string) int {
	test.Helper()

	for i, pageLine := range lines {
		holdsAll := true

		for _, token := range tokens {
			if tokenIndex(test, pageLine, token, 0) < 0 {
				holdsAll = false

				break
			}
		}

		if holdsAll {
			return i
		}
	}

	return -1
}

// pageLines takes a rendered page and returns its lines without the final newline.
func pageLines(test testing.TB, page string) []string {
	test.Helper()

	return strings.Split(strings.TrimSuffix(page, "\n"), "\n")
}

// flagHeading takes an enumeration and a flag ID and returns that flag's option heading, the one
// the help page and the info pages share.
func flagHeading(t *testing.T, flags []params.Flag, id params.FlagType) string {
	t.Helper()

	for i := range flags {
		if flags[i].FlagID == id {
			return optionHeading(&flags[i])
		}
	}

	t.Fatalf("💣 the fixture enumeration lacks the flag %q", id)

	return ""
}

// optionText takes a page's lines and an option heading, and returns the lines of that option: from
// its heading line to the line before the next option or the next unindented line, blank lines
// within the option included, trailing blank lines dropped.
func optionText(t *testing.T, lines []string, heading string) []string {
	t.Helper()

	start := -1

	for i, pageLine := range lines {
		switch {
		case start < 0 && strings.HasPrefix(pageLine, heading) && !strings.HasPrefix(strings.TrimPrefix(pageLine, heading), "-"):
			start = i
		case start >= 0 && (strings.HasPrefix(pageLine, "-") || (pageLine != "" && !strings.HasPrefix(pageLine, " "))):
			return trimBlankTail(t, lines[start:i])
		}
	}

	if start < 0 {
		t.Fatalf("💣 the page carries no option for %s:\n%s", heading, strings.Join(lines, "\n"))
	}

	return trimBlankTail(t, lines[start:])
}

// trimBlankTail takes lines and returns them without their trailing blank lines.
func trimBlankTail(test testing.TB, lines []string) []string {
	test.Helper()

	end := len(lines)
	for end > 0 && lines[end-1] == "" {
		end--
	}

	return lines[:end]
}

// pageKeys takes a page and a provider ID and returns the distinct fully qualified keys of that
// provider the page names, in first-occurrence order.
func pageKeys(test testing.TB, page, providerID string) []string {
	test.Helper()

	var keys []string

	// A quoted pattern is a search example, not a key.
	for _, match := range regexp.MustCompile(`(^|[^'])(`+regexp.QuoteMeta(providerID)+`/[^\s,'"]+)`).FindAllStringSubmatch(page, -1) {
		key := strings.TrimSuffix(match[2], ".")
		if !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}

	return keys
}

// checkKeysOfProvider asserts that every fully qualified key a page names belongs to the provider's
// models and that at most one key names a model of each medium, then returns the keys.
func checkKeysOfProvider(t *testing.T, pageName, page string, prov catalog.Provider, models []catalog.Model) []string {
	t.Helper()

	if len(models) == 0 {
		t.Fatalf("💣 %s has no models to check the keys against", pageName)
	}

	keys := pageKeys(t, page, prov.ID)
	perMedia := map[media.Kind]int{}

	for _, key := range keys {
		modelIndex := slices.IndexFunc(models, func(model catalog.Model) bool { return prov.ID+"/"+model.ID == key })
		if modelIndex < 0 {
			t.Errorf("✗ %s names %s, which is no model of %s", pageName, key, prov.ID)

			continue
		}

		perMedia[models[modelIndex].Media]++
	}

	for media, count := range perMedia {
		if count > 1 {
			t.Errorf("✗ %s names %d %s keys, want at most one example per medium", pageName, count, media)
		}
	}

	return keys
}

// checkGroupIntro asserts that a group intro renders on its own line above the details column,
// after a blank line, with the group's first detail line beneath it at the details column.
func checkGroupIntro(t *testing.T, optionText []string, intro string) {
	t.Helper()

	at := slices.IndexFunc(optionText, func(optionLine string) bool { return strings.TrimSpace(optionLine) == intro })
	if at < 0 {
		t.Errorf("✗ the option carries no intro line %q:\n%s", intro, strings.Join(optionText, "\n"))

		return
	}

	if at == 0 || optionText[at-1] != "" {
		t.Errorf("✗ the intro %q does not follow a blank line", intro)
	}

	introIndent := len(optionText[at]) - len(strings.TrimLeft(optionText[at], " "))

	if at+1 >= len(optionText) {
		t.Errorf("✗ the intro %q has no detail line beneath it", intro)

		return
	}

	detailIndent := len(optionText[at+1]) - len(strings.TrimLeft(optionText[at+1], " "))
	if introIndent == 0 || introIndent >= detailIndent {
		t.Errorf("✗ the intro %q sits at column %d, want between the heading and the details column %d", intro, introIndent, detailIndent)
	}
}

// optionDetails takes a page's lines and an option heading and returns the option's details: its
// lines with the heading removed, every run of whitespace collapsed to one space so an assertion
// survives wrapping.
func optionDetails(t *testing.T, lines []string, heading string) string {
	t.Helper()

	detailText := slices.Clone(optionText(t, lines, heading))
	if len(detailText) == 0 {
		return ""
	}

	detailText[0] = strings.TrimPrefix(detailText[0], heading)

	return flattened(t, strings.Join(detailText, "\n"))
}

// checkOption asserts that an option renders under its heading and that its details carry the
// tokens in order.
func checkOption(t *testing.T, lines []string, heading string, tokens ...string) {
	t.Helper()

	checkInOrder(t, "the "+heading+" option", optionDetails(t, lines, heading), tokens...)
}

// checkOptionsAligned asserts that every continuation line of every option starts at one shared
// details column.
func checkOptionsAligned(t *testing.T, pageName string, lines []string, headings ...string) {
	t.Helper()

	column := -1

	for _, heading := range headings {
		for _, continuation := range optionText(t, lines, heading)[1:] {
			if continuation == "" || strings.TrimSpace(continuation) == GroupIntroRemainder+":" || strings.HasPrefix(strings.TrimSpace(continuation), strings.TrimSuffix(GroupIntro, "%s")) {
				continue
			}

			indent := len(continuation) - len(strings.TrimLeft(continuation, " "))

			switch {
			case column < 0:
				column = indent
			case indent != column:
				t.Errorf("✗ %s: a %s continuation line sits at column %d, others at %d: %q", pageName, heading, indent, column, continuation)
			}
		}
	}
}

// checkPageLayout checks that every line fits within pageWidth and has no trailing spaces or tabs.
func checkPageLayout(t *testing.T, pageName, page string) {
	t.Helper()

	for lineNo, pageLine := range pageLines(t, page) {
		if width := len([]rune(pageLine)); width > pageWidth {
			t.Errorf("✗ %s line %d runs to %d columns, past the page width: %q", pageName, lineNo+1, width, pageLine)
		}

		if pageLine != strings.TrimRight(pageLine, " \t") {
			t.Errorf("✗ %s line %d carries trailing whitespace: %q", pageName, lineNo+1, pageLine)
		}
	}
}

// checkCounts asserts that one line carries the model counts: the total and each medium's nonzero
// count, as whole words.
func checkCounts(t *testing.T, pageName, page string, total, image, video int) {
	t.Helper()

	counts := []string{strconv.Itoa(total)}
	if image > 0 {
		counts = append(counts, strconv.Itoa(image))
	}

	if video > 0 {
		counts = append(counts, strconv.Itoa(video))
	}

	if lineHolding(t, pageLines(t, page), counts...) < 0 {
		t.Errorf("✗ %s carries no line counting %d models (%d image, %d video):\n%s", pageName, total, image, video, page)
	}
}

// cardFlags returns ordered flag records with descriptions, value hints, comments, examples, and
// boolean options for the page-formatting tests.
func cardFlags(test testing.TB) []params.Flag {
	test.Helper()

	return []params.Flag{
		{FlagID: params.FlagTypeSize, DataType: params.DataString, FlagName: "Size", Aliases: []string{"s"}, TextHint: "WxH", ExampleValues: []string{"1024x768"}, Description: "Dimensions in width and height.", Comment: "Supersedes --aspect-ratio and --resolution."},
		{FlagID: params.FlagTypeAspect, DataType: params.DataString, FlagName: "Aspect ratio", Aliases: []string{"a"}, TextHint: "ratio", ExampleValues: []string{"16:9", "2:3"}, Description: "A value in <x:y> format.", Comment: "Superseded by an explicit --size value when supported."},
		{FlagID: params.FlagTypeQuality, DataType: params.DataString, FlagName: "Quality", Aliases: []string{"q"}, TextHint: "level", Description: "Image quality level.", Comment: "Fixture quality note."},
		{FlagID: flagTypeNumberFixture, DataType: params.DataNumber, FlagName: "Number fixture", TextHint: "number", Description: "A number fixture."},
		{FlagID: params.FlagTypeDuration, DataType: params.DataInteger, FlagName: "Duration", Aliases: []string{"d"}, TextHint: "seconds", Description: "Video length in seconds."},
		{FlagID: params.FlagTypeImageN, DataType: params.DataInteger, FlagName: "Images", Aliases: []string{"N"}, TextHint: "count", Description: "Number of images to generate."},
		{FlagID: params.FlagTypeInputMedia, DataType: params.DataString, FlagName: "Input media", Aliases: []string{"i"}, TextHint: "media-file", Description: "File path or URL to an image or video.", Comment: "Repeat the flag for URLs.", AllowMultiple: true},
		{FlagID: params.FlagTypeThoughts, DataType: params.DataBoolean, FlagName: "Include thoughts", Aliases: []string{"n"}, Description: "Write model thought traces to a markdown file beside the artifact."},
		{FlagID: "seed", DataType: params.DataInteger, FlagName: "Seed", Aliases: []string{"e"}, TextHint: "number", Description: "Sampling seed.", Comment: "Exact repetition is not guaranteed."},
		{FlagID: "disable-prompt-upsampling", DataType: params.DataBoolean, FlagName: "Disable prompt upsampling", Description: "Turn off the provider's automatic prompt expansion."},
	}
}

// cardFixture returns an image model with aliases, varied parameter constraints, and a required
// seed parameter.
func cardFixture(test testing.TB) catalog.ProvModelPair {
	test.Helper()

	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	return catalog.ProvModelPair{
		Provider: catalog.Provider{ID: "alpha", DisplayName: "Alpha Labs", APIKeyEnvVar: "ALPHA_API_KEY"},
		Model: catalog.Model{ID: "img-one", Name: "Image One", Description: "A fixture image model.", Media: media.Image, Aliases: []string{"ione", "one"}, Params: []params.Definition{
			{FlagID: params.FlagTypeSize, CustomSize: &params.SizeBounds{MinEdge: 64, MaxPx: 4194304, LongEdge: 1024}},
			{ParamID: "aspect_wire", FlagID: params.FlagTypeAspect},
			{ParamID: "quality_wire", FlagID: params.FlagTypeQuality, AllowedValues: []string{"low", "high"}},
			{ParamID: "number_wire", FlagID: flagTypeNumberFixture, MinValue: params.GetSetIf(true, 0.0), MaxValue: params.GetSetIf(true, 2.0)},
			{ParamID: "count_wire", FlagID: params.FlagTypeImageN, MaxValue: params.GetSetIf(true, 10.0)},
			{FlagID: params.FlagTypeInputMedia, MaxMultiple: 4, ModelInfoComment: "Fixture keyframe guidance in full."},
			{FlagID: params.FlagTypeThoughts},
			{ParamID: "seed_wire", FlagID: "seed", Required: true},
			{ParamID: "disable_pup", FlagID: "disable-prompt-upsampling"},
		}},
	}
}

// providerFixture supplies image and video models with shared and differing parameter declarations.
// Its long model identifier exercises wrapping.
func providerFixture(test testing.TB) (catalog.Provider, []catalog.Model) {
	test.Helper()

	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	prov := catalog.Provider{ID: "gamma", DisplayName: "Gamma Systems", APIKeyEnvVar: "GAMMA_API_KEY"}
	models := []catalog.Model{
		{ID: "gam-one", Name: "Gamma One", Media: media.Image, Aliases: []string{"gone"}, Params: []params.Definition{
			{FlagID: params.FlagTypeAspect, ModelInfoComment: "Gamma One aspect guidance, card only."},
			{FlagID: params.FlagTypeQuality, AllowedValues: []string{"low", "high"}},
			{FlagID: params.FlagTypeSize, AllowedValues: []string{"1024x1024"}},
			{FlagID: params.FlagTypeImageN, MaxValue: params.GetSetIf(true, 4.0)},
		}},
		{ID: "gam-two", Name: "Gamma Two", Media: media.Image, Params: []params.Definition{
			{FlagID: params.FlagTypeAspect},
			{FlagID: params.FlagTypeQuality, AllowedValues: []string{"low", "high"}},
			{FlagID: params.FlagTypeSize, AllowedValues: []string{"512x512", "768x768"}},
		}},
		{ID: "gam-long-identifier-exceeding-thirty-two", Name: "Gamma Long", Media: media.Image, Aliases: []string{"glong"}, Params: []params.Definition{
			{FlagID: params.FlagTypeAspect},
			{FlagID: params.FlagTypeImageN, MaxValue: params.GetSetIf(true, 9.0)},
		}},
		{ID: "gam-vid", Name: "Gamma Video", Media: media.Video, Params: []params.Definition{
			{FlagID: params.FlagTypeAspect},
			{FlagID: params.FlagTypeDuration, AllowedValues: []string{"4", "6", "8"}, RuleDescription: "forced to 8 on the fixture's test condition"},
		}},
	}

	return prov, models
}

// aggregatorFixture returns a marked aggregator provider whose models span several vendors per
// medium, one model without a vendor token, and flags that differ per model.
func aggregatorFixture(test testing.TB) (catalog.Provider, []catalog.Model) {
	test.Helper()

	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	prov := catalog.Provider{ID: "gate", DisplayName: "Gateway", APIKeyEnvVar: "GATE_API_KEY", Aggregator: true}
	models := []catalog.Model{
		{ID: "vendor-one/r1", Name: "R1", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeSize, AllowedValues: []string{"1024x1024"}}, {FlagID: params.FlagTypeAspect}, {FlagID: params.FlagTypeQuality, AllowedValues: []string{"low"}}}},
		{ID: "vendor-one/r2", Name: "R2", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeAspect}, {FlagID: "seed"}}},
		{ID: "vendor-two/g1", Name: "G1", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeAspect}}},
		{ID: "vendor-three/o1", Name: "O1", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeQuality, AllowedValues: []string{"low"}}}},
		{ID: "solo", Name: "Solo", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeAspect}}},
		{ID: "bytedance/v1", Name: "V1", Media: media.Video, Params: []params.Definition{{FlagID: params.FlagTypeAspect}, {FlagID: params.FlagTypeDuration, AllowedValues: []string{"5"}}}},
		{ID: "bytedance/v2", Name: "V2", Media: media.Video, Params: []params.Definition{{FlagID: params.FlagTypeDuration, AllowedValues: []string{"5"}}}},
		{ID: "alibaba/w1", Name: "W1", Media: media.Video, Params: []params.Definition{{FlagID: params.FlagTypeAspect}, {FlagID: "seed"}}},
	}

	return prov, models
}

// paramSection cuts the model view's text from one flag name to the next (or the end), so each
// parameter's constraint data is asserted inside its own section rather than against digits
// anywhere in the view.
func paramSection(t *testing.T, view, flagName, nextFlagName string) string {
	t.Helper()

	start := strings.Index(view, flagName)
	if start < 0 {
		t.Fatalf("💣 the view lacks the %q section:\n%s", flagName, view)
	}

	section := view[start:]
	if nextFlagName == "" {
		return section
	}

	stop := strings.Index(section, nextFlagName)
	if stop < 0 {
		t.Fatalf("💣 the view lacks the %q section after %q:\n%s", nextFlagName, flagName, view)
	}

	return section[:stop]
}

// providerPageFixture supplies models with identical, differing, and partially shared options.
func providerPageFixture(test testing.TB) (catalog.Provider, []catalog.Model) {
	test.Helper()

	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	prov := catalog.Provider{ID: "gamma", DisplayName: "Gamma Systems", APIKeyEnvVar: "GAMMA_API_KEY"}
	first := catalog.Model{ID: "gam-one", Name: "Gamma One", Media: media.Image, Aliases: []string{"gone"}, Params: []params.Definition{
		{FlagID: params.FlagTypeAspect},
		{FlagID: params.FlagTypeQuality, AllowedValues: []string{"low", "high"}},
		{FlagID: params.FlagTypeSize, AllowedValues: []string{"1024x1024"}},
		{FlagID: params.FlagTypeImageN, MaxValue: params.GetSetIf(true, 4.0)},
	}}
	second := catalog.Model{ID: "gam-two", Name: "Gamma Two", Media: media.Image, Params: []params.Definition{
		{FlagID: params.FlagTypeAspect},
		{FlagID: params.FlagTypeQuality, AllowedValues: []string{"low", "high"}},
		{FlagID: params.FlagTypeSize, AllowedValues: []string{"512x512", "768x768"}},
	}}

	return prov, []catalog.Model{first, second}
}

// optionBlock takes a rendered provider page and a flag's long form, and returns the page text from
// that option's heading up to the next option heading or section label.
func optionBlock(t *testing.T, page, longForm string) string {
	t.Helper()

	pageLines := strings.Split(page, "\n")
	blockStart := -1

	// An option starts with a dash at the margin and ends before the next option or nonempty
	// unindented line.
	for i, pageLine := range pageLines {
		switch {
		case blockStart < 0 && strings.HasPrefix(pageLine, "-") && strings.Contains(pageLine, "--"+longForm):
			blockStart = i
		case blockStart >= 0 && (strings.HasPrefix(pageLine, "-") || (pageLine != "" && !strings.HasPrefix(pageLine, " "))):
			return strings.Join(pageLines[blockStart:i], "\n")
		}
	}

	if blockStart < 0 {
		t.Fatalf("💣 the page carries no option block for --%s:\n%s", longForm, page)
	}

	return strings.Join(pageLines[blockStart:], "\n")
}

// flattened takes rendered page text and returns it with every run of whitespace collapsed to one
// space, so an assertion on a phrase survives the page's wrapping.
func flattened(test testing.TB, pageText string) string {
	test.Helper()

	return strings.Join(strings.Fields(pageText), " ")
}

// scopeFixture returns five image models with universal and partial flag support. Their
// declarations include uniform constraints, groups of equal size, and one largest group both before
// and after smaller groups in catalog order.
func scopeFixture(test testing.TB) (catalog.Provider, []catalog.Model) {
	test.Helper()

	//nolint:gosec // APIKeyEnvVar stores an environment-variable name, not a credential value.
	prov := catalog.Provider{ID: "delta", DisplayName: "Delta", APIKeyEnvVar: "DELTA_API_KEY"}
	sizeA := params.Definition{FlagID: params.FlagTypeSize, AllowedValues: []string{"1024x1024"}}
	sizeB := params.Definition{FlagID: params.FlagTypeSize, AllowedValues: []string{"512x512"}}
	lowNumber := params.Definition{FlagID: flagTypeNumberFixture, MinValue: params.GetSetIf(true, 0.0), MaxValue: params.GetSetIf(true, 1.0)}
	highNumber := params.Definition{FlagID: flagTypeNumberFixture, MinValue: params.GetSetIf(true, 0.0), MaxValue: params.GetSetIf(true, 2.0)}
	squareAspect := params.Definition{FlagID: params.FlagTypeAspect, AllowedValues: []string{"1:1"}}
	quality := params.Definition{FlagID: params.FlagTypeQuality, AllowedValues: []string{"low", "high"}}
	seed := params.Definition{FlagID: "seed"}
	upsampling := params.Definition{FlagID: "disable-prompt-upsampling"}
	mediaParam := params.Definition{FlagID: params.FlagTypeInputMedia, MaxMultiple: 2}
	models := []catalog.Model{
		{ID: "m1", Name: "M1", Media: media.Image, Params: []params.Definition{{FlagID: params.FlagTypeAspect}, quality, seed, sizeA, {FlagID: params.FlagTypeImageN, MaxValue: params.GetSetIf(true, 4.0)}, lowNumber, upsampling, mediaParam}},
		{ID: "m2", Name: "M2", Media: media.Image, Params: []params.Definition{squareAspect, quality, seed, sizeA, {FlagID: params.FlagTypeImageN, MaxValue: params.GetSetIf(true, 9.0)}, lowNumber, upsampling, mediaParam}},
		{ID: "m3", Name: "M3", Media: media.Image, Params: []params.Definition{squareAspect, seed, sizeA, lowNumber, upsampling, mediaParam}},
		{ID: "m4", Name: "M4", Media: media.Image, Params: []params.Definition{squareAspect, seed, sizeB, highNumber, upsampling}},
		{ID: "m5", Name: "M5", Media: media.Image, Params: []params.Definition{squareAspect, highNumber, upsampling}},
	}

	return prov, models
}

// providerWith returns a copy of the provider with its Models field set to the supplied models.
func providerWith(test testing.TB, prov catalog.Provider, models []catalog.Model) *catalog.Provider {
	test.Helper()

	prov.Models = models

	return &prov
}

// infoFlags returns the ordered flag records used by the information-page fixtures.
func infoFlags(test testing.TB) []params.Flag {
	test.Helper()

	return []params.Flag{
		{FlagID: params.FlagTypeSize, DataType: params.DataString, FlagName: "Size", Aliases: []string{"s"}, TextHint: "WxH", Description: "Dimensions in width and height."},
		{FlagID: params.FlagTypeAspect, DataType: params.DataString, FlagName: "Aspect ratio", Aliases: []string{"a"}, TextHint: "ratio", Description: "A value in <x:y> format."},
		{FlagID: params.FlagTypeQuality, DataType: params.DataString, FlagName: "Quality", Aliases: []string{"q"}, TextHint: "level", Description: "Image quality level."},
		{FlagID: flagTypeNumberFixture, DataType: params.DataNumber, FlagName: "Number fixture", TextHint: "number", Description: "A number fixture."},
		{FlagID: params.FlagTypeDuration, DataType: params.DataInteger, FlagName: "Duration", Aliases: []string{"d"}, TextHint: "seconds", Description: "Video length in seconds."},
		{FlagID: params.FlagTypeImageN, DataType: params.DataInteger, FlagName: "Images", Aliases: []string{"N"}, TextHint: "count", Description: "Number of images to generate."},
		{FlagID: params.FlagTypeInputMedia, DataType: params.DataString, FlagName: "Reference image paths", Aliases: []string{"i"}, TextHint: "path", Description: "File path or web URL to reference images.", AllowMultiple: true},
	}
}
