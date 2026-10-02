# Edit an image or use references

Supply a source image to change its background, lighting, or appearance. A model may also accept several references to guide one result.

## Check accepted inputs

```sh
bild info google/gemini-3.1-flash-image
```

Look for `--input-media` and its maximum input count. On a model that does not accept input media, the sources are ignored with a notice and are never read.

## Edit a local image

```sh
bild --model google/gemini-3.1-flash-image --input-media product.jpg --output-path ./product-studio.png "Keep the product shape, markings, and color. Replace the setting with a white studio background and soft, even lighting"
```

Local sources must be PNG, JPEG, WebP, or MP4, at most 64 MiB each. The selected model must accept the media kind. Bildomat detects local formats from file contents, not filenames. It preserves the source file.

With direct Recraft editing, also specify how far the image may change:

```sh
bild --model recraft/recraftv4_1 --input-media product.jpg --strength 0.4 --output-path ./product-edited.png "A studio photograph on a pale gray background"
```

Recraft editing requires `--strength` from 0 to 1. Smaller values request greater similarity. Recraft V2 models and the V4 Styles models do not accept input media.

## Combine references

```sh
bild --model openai/gpt-image-2 --input-media front.png --input-media side.png --input-media palette.jpg --output-path ./product-view.png "Create one product photograph using the object views and the supplied color palette"
```

Repeated flags preserve source order. You can also use `--input-media front.png,side.png` for local files. A comma in a local filename is treated as a separator, even when quoted. Inputs beyond the model's maximum are dropped from the end, with a notice, and are never read.

## Use a URL

```sh
bild --model google/gemini-3.1-flash-image --input-media 'https://example.com/reference.png' --output-path ./restyled.png "Restyle the reference as a charcoal drawing"
```

Replace the example URL with an accessible image URL. Only HTTP and HTTPS are accepted. Give each URL its own flag: commas remain part of a URL. Before submitting, Bildomat requests the URL itself to learn whether it holds an image or a video; an unreachable URL, or one that holds neither, fails the run. Some providers fetch the URL themselves; others receive media downloaded by Bildomat. See [input-media reference](../reference/input-media.md#provider-input-behavior) for the distinctions.

For [animation](generate-video.md) or [video keyframes](use-keyframes.md), select a supporting video model. Image-editing models remove frame prefixes and retain ordinary references where they accept input media.

Revised 2026-10-01
