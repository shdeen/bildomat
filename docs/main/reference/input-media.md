# Input media and frame rules

`--input-media`, `-i`, accepts local file paths and HTTP(S) URLs. Repeat the flag to preserve a sequence of sources. A value that begins with a URL, after any frame prefix, is kept whole, commas included. Every other value is split at each comma, even when quoted or given with its own flag. A local file whose name contains a comma therefore cannot be supplied, and a URL that follows a local path in the same value is split at its commas too. Give each URL its own flag. An empty entry, as in `-i a.png,`, is read as a missing file and fails the run.

```sh
bild --model gemini -i product.png -i background.png "Place the product in the supplied setting"
bild --model gemini -i product.png,background.png "Place the product in the supplied setting"
```

Local sources must be readable files no larger than 64 MiB. Content detection accepts PNG, JPEG, WebP, and MP4, regardless of filename extension. An MP4 file is recognized only when its header lists an `mp4` brand, such as `mp41` or `mp42`; a file that lists only other brands, such as `isom` or `avc1`, fails. The same rule applies to M4V and QuickTime MOV files: they pass only when their header lists an `mp4` brand. Directories and other formats, including SVG input, fail with exit code 1. Quote paths containing spaces. Bildomat does not expand a quoted `~` in input paths; use a full path or let the shell expand it.

URLs must have an HTTP or HTTPS scheme and a host. When the selected provider's API key is available, Bildomat contacts each URL that the model will use before adjusting options. It asks for the headers first and, when those do not identify the media, requests the beginning of the content. The reported content type, or the content itself, decides whether the source is an image or a video; the path's extension plays no part. Any image or video type that the server reports passes this check, including types such as GIF or SVG that are refused as local files. When the server reports no image or video type, Bildomat examines the content instead, and SVG content is not recognized that way. When Bildomat downloads a URL itself, as noted for some providers below, the download is limited to 64 MiB, and a larger file fails the run with exit code 1. An unreachable URL, an error status, or content that is neither an image nor a video fails the run with exit code 1 before submission. Bildomat sends no credential with these requests. An identified URL does not prove that the provider will accept its content.

Bildomat decides which sources the model keeps before it reads any of them. A model with a repeat maximum keeps the first sources up to that limit and reports the omission. A model without input support keeps none and reports that the inputs were ignored. Sources that are not kept are never opened, read, or checked, so an unreadable extra file does not fail the run. No declared maximum means Bildomat imposes no catalog count cap; the provider can still reject the request.

## Frame prefixes

| Prefix | Meaning on a supporting model |
| --- | --- |
| `first:` | Opening image. |
| `last:` | Closing image. |
| `SECONDS:` | Nonnegative finite time, for example `2.5:`. |

Keywords ignore case. A numeric time can use any decimal or exponent form, such as `2.5`, `+2`, or `1e1`. A negative or nonfinite numeric time fails. Text before the first colon that is neither a recognized keyword nor a parseable number remains part of the source, so HTTP(S) URLs remain intact. Such text placed in front of a URL, as in `start:https://example.com/a.png`, fails the run. For a local filename that begins with a keyword or a number followed by a colon, write `./first:study.png` or `./3:study.png` so that the start is not read as a prefix. A value with nothing after its first colon, such as `first:`, is read whole as a source.

A prefix on a model without frame support is removed with a notice; supported media remains as an ordinary reference. Frame resolution follows input capping and duration adjustment.

## Opening and closing frames

Supporting Google Veo, OpenRouter video, and direct Kling video models use endpoint positions. Consult the model catalog: input support alone does not guarantee both endpoints.

`first:` and `last:` claim their positions. Two images claiming the same position fail. Numeric times fill positions not already claimed:

- Two numeric times occupy opening and closing positions in ascending time order. Equal times fail.
- One numeric time takes the only free position, if just one remains.
- With both positions free and a known duration, a time at or before half the duration takes the opening position; a later time takes the closing position.
- With both free and no duration, zero takes the opening position and a positive time takes the closing position.

More prefixed images than free positions fails. A prefix on a video source fails for endpoint models. Numeric times are mapped to endpoints, even if a time exceeds the duration; they are not requests for exact intermediate frames. A mapping produces a notice unless the time already equals zero for the opening frame or the known duration for the closing frame.

Direct Veo requires an opening image whenever a closing image is supplied. Unprefixed standard/fast Veo images are references, not implicit opening frames. Lite treats its single unprefixed image as an opening frame. Veo’s duration adjustment makes it 8 seconds when inputs are present.

OpenRouter sends unprefixed media as ordinary references. Some models expose only a single input or require input media. `openrouter/black-forest-labs/flux-3-video` does not declare input-media. For models described as opening-frame-only, use `first:`; a generic endpoint mapping does not guarantee the provider accepts a closing frame.

Direct Kling standard video models place unprefixed images in remaining endpoint positions in order, within their model’s input cap. Omni models keep unprefixed images as references and can combine references with explicitly selected endpoints. They still allow only one explicitly chosen opening and one closing position.

## BFL timed keyframes

Direct `bfl/flux-3-video` accepts up to ten image inputs, or one continuation video. It rejects a mix of images and video and rejects multiple videos. A numeric time on a video fails. When `--duration` is set, a `first:` or `last:` prefix on the video also fails, because the prefix becomes a time. Use an untimed video for continuation.

For images with numeric times, supply `--duration`. Times must increase strictly in source order and lie between zero and the adjusted duration, inclusive. Duplicate or decreasing times fail.

With a duration, `first:` becomes zero and `last:` becomes the duration. Without a duration, those keywords instead move their images to the beginning and end of the sequence and are removed. Numeric times still require a duration. Without prefixes, images remain in the supplied order.

When at least one image has a time and other times are missing, Bildomat first assigns a missing first image to zero and a missing last image to the duration. If interior times remain unset, it distributes the whole sequence evenly: image index `i` of `n` uses `i × duration / (n - 1)`, starting at zero. Every supplied time must then agree with that spacing or the run fails. If assigning the endpoints leaves every time known, explicit intermediate times can have unequal spacing.

For example, a ten-second clip with three images at `0`, `3`, and `10` is valid when all those times are supplied. Four images with only the second timed at `3` and an untimed third require uniform spacing; `3` disagrees with `10/3`, so the run fails. Set every timestamp to avoid inferred spacing.

## Provider input behavior

| Provider and model group | Input behavior |
| --- | --- |
| OpenAI images | Up to 16 images. URLs are downloaded and sent as files. Video input fails. |
| OpenAI Sora | One image. URLs are downloaded. An image whose dimensions differ from the requested size is center-cropped, resized, and encoded as PNG; an image already at that size is sent unchanged. See [Sora sizing](parameter-adjustment.md#provider-rules). Video input fails. |
| xAI images / video | Up to three images / one image. Local images are sent inline; URLs are passed to xAI. Video input fails even for a video model. |
| OpenRouter | Local images or videos are sent inline; URLs are passed through. Actual media and frame support depends on the exact model. Count and required-input constraints are in the catalog. FLUX Video Edit and FLUX Video Upscale each require one video, and the provider accepts it as a public HTTPS URL, not as a local file. HeyGen Avatar IV requires one portrait image. |
| Google Gemini image | Images and videos, up to 14 inputs on Gemini 3 image models; Gemini 2.5 Flash Image has no declared count maximum. Prefixes become ordinary references. |
| Google Gemini Omni video | Images and videos, with no declared count maximum on either Omni model. One image is used for animation; multiple images supply references. Prefixes become ordinary references. |
| Google Veo standard / fast | Up to three inputs: images or one video, never both together. A video extends the supplied video. URLs are downloaded. Duration becomes 8 seconds. |
| Google Veo Lite | One input. URLs are downloaded. An unprefixed image becomes the opening frame. Bildomat can submit one video for extension, but the provider determines whether Lite accepts that operation. Duration becomes 8 seconds with input. |
| BFL images | Models declaring input-media accept images with the catalog’s cap. Prefixes are removed; video input fails. Fill, fine-tuned fill, outpainting, expand, deblur, erase, and both virtual try-on models require one image. Outpainting also requires size, erase requires `--mask`, and virtual try-on requires `--garment-url`. FLUX1.1 [pro] Ultra Finetuned takes one optional reference image. Models without input-media ignore the sources without reading them. |
| BFL FLUX.3 video | Up to ten images as keyframes, or one continuation video. See timing rules above. |
| BFL video tools | FLUX Video Edit and FLUX Video Upscale each take one video: a local MP4, sent inline, or an HTTP(S) URL, passed to BFL. A run without a video, with an image source, or with a frame prefix fails before submission. |
| Sourceful | Up to ten reference images, or four on Riverflow 2 Fast, sent inline for local files or by URL. Bildomat does not itself reject a video source before submission here; provider acceptance is not implied. |
| Recraft | One image on editing-capable models; include `--strength`. A local image is sent inline, and a URL is passed to Recraft unchanged. Bildomat does not itself reject a video source before submission here; provider acceptance is not implied. The raster and vector V2 models and the four V4 Styles models do not declare input-media. |
| Kling images | Standard models accept one reference; Omni models accept up to ten. Bildomat rejects a kept video source before submission: a local MP4, or a URL identified as video. Standard image models reject negative-prompt combined with input at the provider. |
| Kling video | Standard models accept one or two images according to the catalog; Omni models accept up to seven. Video sources fail as they do for Kling images. Endpoints and references follow the rules above. |

Where this table says that a model requires an input or an option, Bildomat does not check for it before submission. A run that lacks one is submitted, and the provider can reject it. The BFL video tools are the exception: Bildomat checks for their video.

Use [the catalog](providers-and-models.md) for exact per-model limits and [the keyframe guide](../how-to/use-keyframes.md) for complete examples.

Revised 2026-10-06
