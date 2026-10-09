# Generation Flags

These options apply to generation. Every value defaults to **unset**: Bildomat sends no value unless you supply one or an adjustment derives one. `--include-thoughts` is off unless enabled. An omitted Boolean does not explicitly disable a provider default.

Supported flags and values depend on the model. Unsupported supplied options are omitted with a notice; `--include-thoughts=false` does not trigger an unsupported-option notice. Read [parameter adjustment](parameter-adjustment.md) for replacement and omission rules, and [commands](commands.md) for parsing and run-level flags.

## Model Parameters

| Flag and shorthand | Type | Meaning |
| --- | --- | --- |
| `--size`, `-s` | string | Positive width and height, `WxH`. A valid supported size supersedes ratio and resolution. |
| `--aspect-ratio`, `-a` | string | Width-to-height ratio such as `16:9`, or an allowed named value. |
| `--resolution`, `-r` | string | Resolution class, pixel height, or dimensions where supported. Derivation rules depend on the model. |
| `--duration`, `-d` | integer | Video length in seconds, from the values or range the model declares. |
| `--quality`, `-q` | string | Image quality level, from the values the model declares. A model declaring none passes the level to the provider as given. |
| `--output-format`, `-f` | string | Requested image format from the model's allowed values. A model declaring none passes the format to the provider as given. A supported image extension in output-path takes precedence. |
| `--num-images`, `-N` | integer | Number of images to generate, within the range the model declares. |
| `--thinking-level`, `-l` | string | Thinking level to apply to the model's generation effort, from the levels the model declares. |
| `--include-thoughts`, `-n` | boolean | Write model thought traces to a markdown file beside the artifact. |
| `--input-media`, `-i` | string; repeatable | File path or HTTP(S) URL of an image or video to edit or use as a reference. |
| `--strength` | number | How far the result may depart from the input image, from 0 (nearly identical) to 1 (minimal similarity). |
| `--seed`, `-e` | integer | Sampling seed, an integer, used by supporting providers. |
| `--background` | string | Background treatment of the generated image, from the treatments the model declares. |
| `--output-compression` | integer | Compression setting for JPEG or WebP. Bildomat sends the value for any output format, including PNG; the provider decides how to apply it. |
| `--safety-tolerance` | integer | Moderation strictness within the range the model declares, where 0 is strictest and higher numbers are more permissive. |
| `--generate-audio` | boolean | Generate a soundtrack alongside the video. |
| `--prompt-upsampling` | boolean | Let the provider expand the prompt before generating. |
| `--disable-prompt-upsampling` | boolean | Turn off the provider's automatic prompt expansion. |
| `--moderation-level` | string | Content moderation strictness applied by the provider, from the levels the model declares. |
| `--person-generation` | string | Whether the model may generate people, and of which ages, as a policy word the model's page explains. |
| `--negative-prompt` | string | Text describing what to exclude from an image or video. |
| `--guidance-scale` | number | Strength of prompt guidance, within the range the model declares. |
| `--steps` | integer | Number of generation steps, within the range the model declares. |
| `--expand-top` | integer | Pixels to add above the input image. |
| `--expand-bottom` | integer | Pixels to add below the input image. |
| `--expand-left` | integer | Pixels to add to the left of the input image. |
| `--expand-right` | integer | Pixels to add to the right of the input image. |
| `--finetune-id` | string | ID of a model customized through provider training. It must already be owned by or shared with your account. |
| `--finetune-strength` | number | How strongly to apply the selected fine-tune. |
| `--mask` | string | Encoded mask data or a mask URL, as supported by the model. |
| `--mask-dilation` | integer | Pixels by which to expand the masked area. |
| `--garment-url` | string | Public HTTP(S) URL of the garment image for virtual try-on. |
| `--style-id` | string | Identifier of an existing provider style compatible with the selected model. |
| `--style-match` | string | How closely the generated image follows the selected style, for example `precise` or `flexible`. |
| `--watermark` | boolean | Include the provider watermark in the generated video. |
| `--upscale-factor` | number | Multiplier applied to the input video dimensions. |
| `--creativity` | integer | Amount of detail the provider may invent while upscaling. |
| `--voice-id` | string | Provider voice identifier used to speak the prompt, for example `16a09e4706f74997ba4ed05ea11470f6`. |
| `--motion-prompt` | string | Text describing the avatar's gestures and movement. |
| `--expressiveness` | string | Avatar expression and movement intensity, for example `low`, `medium`, or `high`. |
| `--image-fit` | string | How the avatar image fits the output canvas, for example `contain` or `cover`. |
| `--remove-background` | boolean | Remove the original background from the avatar image. |
| `--voice-speed` | number | Playback speed multiplier for generated speech. |
| `--voice-pitch` | number | Pitch adjustment in semitones. |
| `--voice-volume` | number | Volume of generated speech, from silent to full volume. |
| `--voice-locale` | string | Locale or accent hint for a multilingual voice, for example `en-US` or `en-GB`. |

String flags take text; quote values containing spaces. Integer flags take whole numbers, number flags take numerical values, and Boolean flags use `--flag` or `--flag=false`. Invalid numerical syntax exits 2; nonfinite numbers that parse are omitted with a notice.

`--input-media` is the only repeatable model option. Repeat it for URLs; comma-separated local paths also work. See [input media](input-media.md) for formats, timing prefixes, validation, and count limits.

`--strength` ranges from 0 (little change) to 1 (minimal similarity) on direct Recraft editing models. Supply it when editing; it is optional for generation without an image. On `bfl/flux-pro-1.1-ultra-finetuned`, `--strength` accepts 0 to 1 and applies to the optional reference image. `--seed` does not guarantee an identical result on a later request.

`--mask` and `--garment-url` send their text to the provider unchanged. Bildomat does not read a local file named there, so supply a URL, or encoded data where the model accepts it. Only `--input-media` reads local files.

Transparent raster output requires a format with an alpha channel, such as PNG or WebP. Background values and compression combinations remain subject to the provider's checks. `--prompt-upsampling` and `--disable-prompt-upsampling` are separate options declared by different models; neither is a universal switch.

## Models Accepting Each Flag

Each provider link lists the exact model options, allowed values, ranges, required inputs, and conditional rules. "All" means every model of that provider; counts describe the catalog covered by these pages.

| Flag | Provider coverage |
| --- | --- |
| `--size` | [OpenAI](catalog/openai.md): all; [Black Forest Labs](catalog/bfl.md): 10 of 25 models; [Recraft](catalog/recraft.md): 11 of 21 models |
| `--aspect-ratio` | [OpenAI](catalog/openai.md): all; [xAI](catalog/xai.md): all; [OpenRouter](catalog/openrouter.md): 83 of 87 models; [Google](catalog/google.md): all; [Black Forest Labs](catalog/bfl.md): 15 of 25 models; [Sourceful](catalog/sourceful.md): all; [Recraft](catalog/recraft.md): all; [Kling](catalog/kling.md): all |
| `--resolution` | [OpenAI](catalog/openai.md): all; [xAI](catalog/xai.md): all; [OpenRouter](catalog/openrouter.md): 73 of 87 models; [Google](catalog/google.md): 8 of 10 models; [Black Forest Labs](catalog/bfl.md): 11 of 25 models; [Sourceful](catalog/sourceful.md): all; [Recraft](catalog/recraft.md): 11 of 21 models; [Kling](catalog/kling.md): all |
| `--duration` | [OpenAI](catalog/openai.md): 2 of 9 models; [xAI](catalog/xai.md): 3 of 7 models; [OpenRouter](catalog/openrouter.md): 26 of 87 models; [Google](catalog/google.md): 3 of 10 models; [Black Forest Labs](catalog/bfl.md): 1 of 25 models; [Kling](catalog/kling.md): 6 of 10 models |
| `--quality` | [OpenAI](catalog/openai.md): 7 of 9 models; [xAI](catalog/xai.md): 1 of 7 models; [OpenRouter](catalog/openrouter.md): 44 of 87 models |
| `--output-format` | [OpenAI](catalog/openai.md): 7 of 9 models; [OpenRouter](catalog/openrouter.md): 46 of 87 models; [Black Forest Labs](catalog/bfl.md): 21 of 25 models; [Sourceful](catalog/sourceful.md): 2 of 4 models; [Recraft](catalog/recraft.md): 1 of 21 models |
| `--num-images` | [OpenAI](catalog/openai.md): 7 of 9 models; [xAI](catalog/xai.md): 4 of 7 models; [OpenRouter](catalog/openrouter.md): 54 of 87 models; [Recraft](catalog/recraft.md): all; [Kling](catalog/kling.md): 4 of 10 models |
| `--thinking-level` | [Google](catalog/google.md): 4 of 10 models; [Sourceful](catalog/sourceful.md): 2 of 4 models |
| `--include-thoughts` | [Google](catalog/google.md): 4 of 10 models |
| `--input-media` | [OpenAI](catalog/openai.md): all; [xAI](catalog/xai.md): all; [OpenRouter](catalog/openrouter.md): 84 of 87 models; [Google](catalog/google.md): all; [Black Forest Labs](catalog/bfl.md): 22 of 25 models; [Sourceful](catalog/sourceful.md): all; [Recraft](catalog/recraft.md): 14 of 21 models; [Kling](catalog/kling.md): all |
| `--strength` | [Black Forest Labs](catalog/bfl.md): 1 of 25 models; [Recraft](catalog/recraft.md): 14 of 21 models |
| `--seed` | [OpenRouter](catalog/openrouter.md): 30 of 87 models; [Google](catalog/google.md): 3 of 10 models; [Black Forest Labs](catalog/bfl.md): 20 of 25 models; [Recraft](catalog/recraft.md): all |
| `--background` | [OpenAI](catalog/openai.md): 7 of 9 models; [OpenRouter](catalog/openrouter.md): 44 of 87 models; [Sourceful](catalog/sourceful.md): 2 of 4 models |
| `--output-compression` | [OpenAI](catalog/openai.md): 7 of 9 models; [OpenRouter](catalog/openrouter.md): 44 of 87 models |
| `--safety-tolerance` | [OpenRouter](catalog/openrouter.md): 3 of 87 models; [Black Forest Labs](catalog/bfl.md): all |
| `--generate-audio` | [OpenRouter](catalog/openrouter.md): 18 of 87 models; [Black Forest Labs](catalog/bfl.md): 1 of 25 models; [Kling](catalog/kling.md): 3 of 10 models |
| `--prompt-upsampling` | [Black Forest Labs](catalog/bfl.md): 10 of 25 models; [Sourceful](catalog/sourceful.md): all |
| `--disable-prompt-upsampling` | [Black Forest Labs](catalog/bfl.md): 4 of 25 models |
| `--moderation-level` | [OpenAI](catalog/openai.md): 7 of 9 models |
| `--person-generation` | [Google](catalog/google.md): 3 of 10 models |
| `--negative-prompt` | [Google](catalog/google.md): 2 of 10 models; [Recraft](catalog/recraft.md): 4 of 21 models; [Kling](catalog/kling.md): 2 of 10 models |
| `--guidance-scale` | [Black Forest Labs](catalog/bfl.md): 5 of 25 models |
| `--steps` | [Black Forest Labs](catalog/bfl.md): 5 of 25 models |
| `--expand-top` | [Black Forest Labs](catalog/bfl.md): 1 of 25 models |
| `--expand-bottom` | [Black Forest Labs](catalog/bfl.md): 1 of 25 models |
| `--expand-left` | [Black Forest Labs](catalog/bfl.md): 1 of 25 models |
| `--expand-right` | [Black Forest Labs](catalog/bfl.md): 1 of 25 models |
| `--finetune-id` | [Black Forest Labs](catalog/bfl.md): 2 of 25 models |
| `--finetune-strength` | [Black Forest Labs](catalog/bfl.md): 2 of 25 models |
| `--mask` | [Black Forest Labs](catalog/bfl.md): 2 of 25 models |
| `--mask-dilation` | [Black Forest Labs](catalog/bfl.md): 1 of 25 models |
| `--garment-url` | [Black Forest Labs](catalog/bfl.md): 2 of 25 models |
| `--style-id` | [Recraft](catalog/recraft.md): 4 of 21 models |
| `--style-match` | [Recraft](catalog/recraft.md): 4 of 21 models |
| `--watermark` | [OpenRouter](catalog/openrouter.md): 1 of 87 models |
| `--upscale-factor` | [OpenRouter](catalog/openrouter.md): 1 of 87 models; [Black Forest Labs](catalog/bfl.md): 1 of 25 models |
| `--creativity` | [OpenRouter](catalog/openrouter.md): 1 of 87 models; [Black Forest Labs](catalog/bfl.md): 1 of 25 models |
| `--voice-id` | [OpenRouter](catalog/openrouter.md): 1 of 87 models |
| `--motion-prompt` | [OpenRouter](catalog/openrouter.md): 1 of 87 models |
| `--expressiveness` | [OpenRouter](catalog/openrouter.md): 1 of 87 models |
| `--image-fit` | [OpenRouter](catalog/openrouter.md): 1 of 87 models |
| `--remove-background` | [OpenRouter](catalog/openrouter.md): 1 of 87 models |
| `--voice-speed` | [OpenRouter](catalog/openrouter.md): 1 of 87 models |
| `--voice-pitch` | [OpenRouter](catalog/openrouter.md): 1 of 87 models |
| `--voice-volume` | [OpenRouter](catalog/openrouter.md): 1 of 87 models |
| `--voice-locale` | [OpenRouter](catalog/openrouter.md): 1 of 87 models |

For a machine-readable answer for a selected model, use `bild info PROVIDER/MODEL --json`. Every accepted flag appears in that model's `params`; general definitions appear in `flags`.

Revised 2026-10-07
