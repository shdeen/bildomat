# Choose image dimensions and format

Inspect the model before choosing dimensions:

```sh
bild info openai/gpt-image-2
```

A model can accept exact dimensions, a fixed set of sizes, or aspect ratio and resolution options. Its page states which forms are available.

## Choose the shape

```sh
bild --model google/gemini-3.1-flash-image --aspect-ratio 16:9 --resolution 2K --output-path ./banner.png "A coastal village with open sky above the rooftops"
```

The model uses its accepted shape and resolution values. If a supplied ratio or resolution is not offered, Bildomat may choose the nearest supported value and report it.

## Request pixel dimensions

```sh
bild --model openai/gpt-image-2 --size 1536x1024 --output-path ./landscape.png "A coastal village with open sky above the rooftops"
```

On a model with custom dimensions, Bildomat adjusts dimensions to its edge, pixel-count, ratio, and increment constraints. On a model with fixed sizes, Bildomat chooses from that list. A valid supported `--size` supersedes both `--aspect-ratio` and `--resolution`. If the size is malformed and omitted, the other supported sizing options can still apply.

A model without `--size` ignores the option with a notice. Use its accepted ratio and resolution flags instead. A resolution label such as `2K` is not a universal way to request dimensions: custom-size models accept dimensions or a ratio through `--resolution`. See [the conversion rules](../reference/parameter-adjustment.md#explicit-size).

## Choose the output format

```sh
bild --model openai/gpt-image-2 --output-path ./banner.webp "A coastal village with open sky above the rooftops"
```

On a model that accepts `--output-format`, a recognized filename extension requests that format and overrides a conflicting format flag. Alternatively, use `--output-format webp` with a filename stem.

If the returned format differs, Bildomat changes the extension and reports the saved path. It does not convert an unsupported provider output into the requested format.

For transparency, select a supporting model and a format with an alpha channel:

```sh
bild --model openai/gpt-image-2 --background transparent --output-path ./compass.png "A brass compass isolated on a transparent background"
```

## Expand an existing image

To outpaint an image into a larger canvas, supply one reference and a size:

```sh
bild --model bfl/flux-tools/outpainting-v1 --input-media portrait.png --size 1536x1024 --output-path ./expanded.png
```

This model requires the input image and a size but not a prompt, so the command carries none. Check its [catalog constraints](../reference/catalog/bfl.md). An aspect ratio alone does not replace the required canvas size.

To add a set number of pixels to particular sides instead, use the expand model with one or more margin options:

```sh
bild --model bfl/flux-pro-1.0-expand --input-media portrait.png --expand-left 512 --expand-right 512 --output-path ./expanded-wide.png "Extend the surrounding landscape naturally"
```

This model requires the input image and at least one of `--expand-top`, `--expand-bottom`, `--expand-left`, and `--expand-right`. Each margin accepts 0 to 2048 pixels.

Use [output-file rules](../reference/output-files.md) when the exact destination name matters to another command.

Revised 2026-10-06
