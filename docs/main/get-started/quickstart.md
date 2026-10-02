# Quickstart

Install `bild`, give it a Google API key, and generate an image. You need network access and a [Google AI Studio key](https://aistudio.google.com/apikey). Generation requests may incur provider charges.

## Install `bild`

On macOS or Linux, run:

```sh
curl -fsSL https://raw.githubusercontent.com/shdeen/bildomat/main/scripts/install/install.sh | sh
```

On Windows, run in PowerShell:

```powershell
irm https://raw.githubusercontent.com/shdeen/bildomat/main/scripts/install/install.ps1 | iex
```

For binaries and other ways to install, see [install and configure Bildomat](install-and-configure.md).

## Add Your Key

On macOS or Linux:

```sh
export GOOGLE_API_KEY='YOUR_API_KEY'
```

In PowerShell:

```powershell
$env:GOOGLE_API_KEY = 'YOUR_API_KEY'
```

The key lasts for this terminal session. To keep it, put it in `~/.bildomat/config.yml` as shown in [install and configure Bildomat](install-and-configure.md#use-the-configuration-file).

## Generate an Image

```sh
bild --model gemini --output-path ./lighthouse.png "A lighthouse on a rocky coast at dawn, watercolor"
```

`gemini` is a short name for Google's `google/gemini-3.1-flash-image` model. Bildomat reports the provider and model, waits for the result, and prints the path of the saved file. Open it.

Quote a prompt that contains spaces. Options can come before or after the prompt. Bildomat never replaces an existing file, so running the command again saves `lighthouse-02.png`. If the model returns a different format, the extension changes to match. The path that Bildomat prints is always the file it saved.

## See What Else Is Available

```sh
bild list
bild info gemini
```

`bild list` shows every provider and model. `bild info gemini` shows the options this model accepts, such as `--aspect-ratio` and `--input-media`, and the values each one allows. Neither command needs a key or makes a generation request.

You have installed Bildomat, connected a provider, and generated an image. Next, [generate and edit your first image](../how-to/first-image.md) to change an image's shape and edit it, or [find a model](../how-to/find-a-model.md) from another provider.

Revised 2026-10-02
