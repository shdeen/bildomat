# Quickstart

Install `bild`, add an API key from a supported provider, and generate an image. You need Internet access and a key from at least one provider. Generation requests may incur provider charges.

## Install `bild`

On macOS or Linux, run:

```sh
curl -fsSL https://bildomat.com/install.sh | sh
```

On Windows, run in PowerShell:

```powershell
irm https://bildomat.com/install.ps1 | iex
```

To install with Homebrew or Go, or to download a binary, see [install and configure Bildomat](install-and-configure.md#install).

## Get a Key

Create an API key with any supported provider, for example at [Google AI Studio](https://aistudio.google.com/apikey), [OpenAI](https://platform.openai.com/api-keys), or [xAI](https://console.x.ai/team/default/api-keys). [Install and configure Bildomat](install-and-configure.md#add-a-provider-key) lists every supported provider, where to create its key, and the variable and configuration entry that hold the key.

## Add Your Key

Set your provider's environment variable. For example, for an OpenAI key, on macOS or Linux run the following command in your terminal:

```sh
export OPENAI_API_KEY='YOUR_API_KEY'
```

On Windows, run the following in PowerShell:

```powershell
$env:OPENAI_API_KEY = 'YOUR_API_KEY'
```

For a key from another provider, use that provider's variable from the [provider table](install-and-configure.md#add-a-provider-key). The variable lasts for this terminal session. To keep the key, put it in `~/.bildomat/config.yml` or export it from your shell configuration file as shown in [install and configure Bildomat](install-and-configure.md#use-the-configuration-file). A key in the config file takes precedence over an environment variable.

## Generate an Image

```sh
bild --model openai/gpt-image-2 --output-path ./lighthouse.png "A lighthouse on a rocky coast at dawn, watercolor"
```

Use a model from the provider whose key you added. `bild list` shows every provider's models. Bildomat reports the provider and model, waits for the result, and prints the path of the saved file. Open it.

Quote a prompt that contains spaces. Options can come before or after the prompt. Bildomat never replaces an existing file, so running the command again saves `lighthouse-02.png`. If the model returns a different format, the extension changes to match. The path that Bildomat prints is always the file it saved.

## See What Else Is Available

```sh
bild list
bild info openai/gpt-image-2
```

`bild list` shows every provider and model. `bild info` followed by a model shows the options that the model accepts, such as `--aspect-ratio` and `--input-media`, and the values each one allows. Neither command needs a key or makes a generation request.

You have installed Bildomat, connected a provider, and generated an image. Next, [generate and edit your first image](../how-to/first-image.md) to change an image's shape and edit it, or [find a model](../how-to/find-a-model.md) from another provider.

Revised 2026-10-05
