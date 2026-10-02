# Generate and Edit Your First Image

Create an image of a paper city, make a wide version, and edit its lighting. Each command saves a separate file so you can compare the results.

You need `bild` on your `PATH`, network access, and a Google API key. See [install and configure Bildomat](../get-started/install-and-configure.md) to install `bild` or create a key. Generation requests may incur provider charges.

## Set the Credential

If you have not configured a Google key in Bildomat's configuration file, export it in this shell:

```sh
export GOOGLE_API_KEY='YOUR_API_KEY'
bild --version
```

A key in `~/.bildomat/config.yml` takes precedence over the environment variable. See [configuration](../reference/configuration.md) if you need to change an existing key.

## Generate the First Image

```sh
bild --model google/gemini-3.1-flash-image --output-path ./paper-city.png "A paper city photographed in soft window light"
```

Bildomat reports the selected provider and model, waits for the result, and prints the saved path. Open that file. The image's appearance varies between runs.

A relative output path is relative to the current directory, even if you configured a default output directory. Bildomat never replaces an existing file: a repeated run can save `paper-city-02.png`. A different returned format can also change the extension. Always use the path Bildomat reports.

## Inspect the Model

```sh
bild info google/gemini-3.1-flash-image
```

Find `--aspect-ratio`, which lists the accepted shapes. Find `--input-media`, which accepts up to 14 sources on this model. Catalog inspection makes no generation request.

## Make a Wide Version

```sh
bild --model google/gemini-3.1-flash-image --aspect-ratio 16:9 --resolution 1K --output-path ./paper-city-wide.png "A paper city photographed in soft window light, wide composition"
```

Open the saved file and compare the composition with the first image. A supplied value outside a model's accepted values can be adjusted or omitted; Bildomat reports the change.

## Edit the Image

Use the actual saved path from the preceding step. The command below assumes it was `./paper-city-wide.png`:

```sh
bild --model google/gemini-3.1-flash-image --input-media ./paper-city-wide.png --aspect-ratio 16:9 --output-path ./paper-city-night.png "Keep the paper city and composition, but make it night with every window glowing"
```

Open the edited file. The source remains available beside it. You have generated an image, selected its shape, inspected a model, and used an image as the starting point for an edit.

Continue with [processing images in scripts](use-bild-in-scripts.md), [using an agent](use-bild-with-agents.md), or [animating an image](generate-video.md).

Revised 2026-10-02
