# Generate and extend video

Select a video model and inspect its duration, resolution, and input options:

```sh
bild list --models --video
bild info google/veo-3.1-generate-preview
```

Set credentials for that provider. Bildomat waits for the provider job and saves the completed video; you do not need to poll a job URL yourself.

## Generate from a prompt

```sh
bild --model google/veo-3.1-generate-preview --duration 8 --aspect-ratio 16:9 --output-path ./harbor.mp4 "A slow camera move across a harbor at sunrise"
```

Durations outside a model’s supported set or range are adjusted with a notice. A requested resolution can also affect duration. Direct Veo uses eight seconds whenever input media is supplied or the selected resolution is `1080p` or `4k`.

## Animate an image

```sh
bild --model google/veo-3.1-generate-preview --input-media first:product.png --duration 8 --output-path ./product-motion.mp4 "The camera moves slowly around the product while the studio lighting remains steady"
```

`first:` makes the image the opening frame. On direct Veo standard and fast, an unprefixed image is a reference rather than an opening frame. On Veo Lite, one unprefixed image becomes the opening frame. Other models have different rules; use the [keyframe guide](use-keyframes.md) for endpoint and timed control.

## Continue a video

```sh
bild --model google/veo-3.1-generate-preview --input-media clip.mp4 --output-path ./extended.mp4 "Continue pulling back to reveal the coastline"
bild --model bfl/flux-3-video --duration 8 --input-media clip.mp4 --output-path ./continued.mp4 "Continue the camera movement into a sunlit courtyard"
```

Both models accept one continuation video and reject mixing a video with images. Use an untimed MP4. Direct Kling and xAI do not accept video input. An MP4 is not a universal input for every video model.

## Request audio or exclude content

```sh
bild --model bfl/flux-3-video --duration 8 --generate-audio --output-path ./harbor-sound.mp4 "Small waves break against the harbor wall as seabirds pass"
bild --model google/veo-3.1-generate-preview --negative-prompt "text, captions, watermarks" --output-path ./harbor-clean.mp4 "A slow camera move across a harbor at sunrise"
```

Each option applies only where the selected model declares it. On direct Kling 2.6, generated audio and frame input require `1080p`; Bildomat does not force that resolution for you.

## Wait or cancel

Asynchronous jobs are polled for up to 30 minutes, with a terminal progress display when ordinary output is shown in a terminal. Temporary network failures and busy or unavailable responses from the provider do not end the wait; Bildomat keeps polling within the time limit. Gemini Omni video models answer a single request instead of running a polled job. Each request, including the download of the finished video, must complete within eight minutes; see [network behavior](../reference/configuration.md#network-behavior). During generation, Ctrl-C cancels the request or the wait and exits with status 1. It does not retract a job from the provider or guarantee a refund. Once Bildomat has received the video, including any download, Ctrl-C no longer cancels the run: Bildomat saves the video and exits with status 0. See [cancellation and failures](../reference/exit-codes-and-errors.md#cancellation).

Revised 2026-10-01
