# Parameter adjustment

Bildomat checks supplied values against the selected model before submitting. An unsupported option is omitted. An unsupported value can be replaced with a nearby allowed value or omitted. Notices describe these decisions on standard error; JSON records them in `adjustments`. A file extension changed after generation is a notice of another kind, which JSON records in `notices`; see [output files](output-files.md#format-and-extension).

Successful adjustment does not guarantee that the provider will accept the request. Account restrictions and combinations that the catalog does not enforce can still produce a provider error. Bildomat does not check for options that the catalog marks required. A request that lacks one is submitted, and the provider can reject it. The exception is the input video of the direct BFL video tools, FLUX Video Edit and FLUX Video Upscale: a run on either one without a video fails before submission.

## Order and precedence

1. A supported image extension in `--output-path` requests `--output-format` when the model accepts that option; it supersedes a conflicting format flag.
2. Unsupported flags are omitted, with notices in alphabetical flag order. `--include-thoughts=false` counts as off.
3. Input media beyond the model’s repeat maximum are discarded from the end, and a model without input-media support discards them all. Discarded sources are never opened, read, or checked. When the list is capped, the notice gives the number supplied and the number kept; on a model without input-media support, it says that the inputs were ignored.
4. Sizing options are resolved.
5. Other accepted options are checked against allowed values and numeric bounds.
6. Provider rules adjust duration, frame selection, image conformance, and provider-specific audio values.

Changes that affect the request produce adjustment notices, except for the unused sizing values that the sections below name. Matching a listed value after trimming surrounding whitespace or changing letter case can use the catalog spelling without a notice. Reading invalid input files or resolving conflicting frames can fail the run.

## Explicit size

A supported `--size` accepts positive integer dimensions such as `1024x768`; an uppercase `X` and surrounding dimension whitespace are accepted. Invalid dimensions are omitted, allowing the remaining sizing options to apply.

For a model with custom bounds, Bildomat fits the dimensions to its edge, pixel-count, ratio, and increment constraints. It caps an excessive ratio, scales dimensions to fit edge and pixel limits, and rounds edges to the declared increment. The result satisfies every declared constraint at once, minimums included. When rounding would break one, Bildomat chooses the nearest dimensions that satisfy them all; among equally near choices it prefers the more balanced shape, then the one with the larger longer edge. If no dimensions satisfy the constraints, the run fails with exit code 1. The resulting dimensions appear in the adjustment.

For a model with fixed sizes, an exact match stays unchanged; otherwise Bildomat chooses the listed size whose width-to-height ratio is nearest. A model with no size bounds passes valid dimensions through.

Once a size is selected, supplied `--aspect-ratio` and `--resolution` are superseded and reported as ignored.

## Deriving a custom size

Without an explicit size, on models with custom bounds:

- A resolution expressed as `WxH` supplies dimensions, adjusted to the bounds. A separately supplied aspect ratio is then not used, and no notice reports it.
- A resolution expressed as a ratio supplies the aspect ratio. It replaces a separately supplied ratio without a notice.
- Other resolution text, including height labels such as `2K`, is omitted.
- An aspect ratio derives dimensions using the catalog’s preferred long edge, then applies its bounds. If no preferred long edge is declared, it cannot derive dimensions this way. If the derived dimensions cannot satisfy the bounds, the run fails with exit code 1.

Ratios must have positive finite components. An uninterpretable ratio is omitted. Options that the model does not declare have already been omitted before these rules apply.

## Choosing a fixed size

Without an explicit size, resolution written as dimensions or a ratio supplies a ratio. A resolution class supplies a representative height. A separate aspect ratio applies unless resolution already supplied one; in that case the separate ratio is not used, and no notice reports it. When resolution supplies a height, an aspect ratio that cannot be interpreted is also dropped without a notice.

With a height, Bildomat chooses the fixed size whose shorter edge is nearest, preferring the requested orientation. A ratio of at least 1 means landscape, as does an absent ratio. A square listed size counts as landscape. If no size matches the orientation, all listed sizes are considered. When several sizes are equally near and a ratio was supplied, the size with the nearest ratio wins. With a ratio alone, Bildomat chooses the closest numerical ratio. Remaining ties use the first listed value.

## Independent ratio and resolution

On models without a size option, ratio and resolution remain separate parameters. Exact allowed values match without regard to case and surrounding whitespace. An unlisted ratio becomes the nearest numerical allowed ratio; an unlisted resolution becomes the allowed value with the nearest representative height. A value that cannot be interpreted is omitted. An unlisted resolution is also omitted when none of the model's allowed values has a representative height, as with allowed values `hd` and `fhd`. If the model declares no allowed values, its text is passed through.

| Resolution text | Representative height |
| --- | --- |
| `4K` | 2160 |
| `8K` | 4320 |
| `720p`, or another number followed by `p` | That number |
| `1K`, `2K`, or another number followed by `k` | Number × 540 |
| `WxH` | The shorter edge |
| A whole number | That number |
| A value ending in `_<class>` | The final class, interpreted by these rules |

These heights compare resolution classes; they do not promise exact generated dimensions. Ties use the earlier listed value. A supported named value can match exactly even when it has no numerical interpretation.

## Other values

Text options with allowed values use the matching catalog spelling. Unlisted values are omitted. Text without an allowed-value list passes through.

Duration with a fixed list becomes the nearest allowed number; ties select the first listed value. Other numeric options with a fixed list must equal a member or are omitted. Numeric ranges clamp values at the minimum or maximum. `--num-images` is at least 1 when supported. Nonfinite numbers are omitted. Booleans retain the explicitly supplied value unless a provider requires a different representation.

Defaults belong to the provider when Bildomat leaves an option unset. Do not interpret an omitted option as an explicit zero or false.

## Provider rules

Direct Google Veo forces duration to 8 seconds whenever input media is supplied or a supported resolution is `1080p` or `4k`. This also supplies a duration when none was requested. Veo Lite has no `4k` option.

Models with frame support resolve prefixes according to [input media](input-media.md). Models without frame support remove prefixes with a notice while retaining supported media. On the direct BFL video tools, a prefix fails the run instead. BFL FLUX.3 can retain intermediate timestamps; endpoint models map timestamps to the opening or closing frame.

Direct OpenAI Sora fits an input image to the requested dimensions. An image of another size is center-cropped, resized, and encoded as PNG. An image that already has those dimensions is sent unchanged, in its own format. Without a selected size, it chooses the listed size matching the image’s orientation with the smallest shorter edge, and among those the size nearest the image’s ratio. Local-image conformance is reported before submission. URL images are downloaded and conformed while the request is prepared. Changes made at that point are reported after generation in text output and are included in JSON `adjustments`.

Direct Kling converts the audio Boolean to the provider’s audio choice. Kling 2.6’s provider requires `1080p` for generated audio or frame input; Bildomat does not automatically raise the resolution for that combination.

Revised 2026-10-06
