# Find a model

Use Bildomat’s local catalog to find a model and inspect its accepted options. These commands do not contact generation services or require provider credentials.

## Narrow the catalog

```sh
bild list --models --image
bild list --models --video
bild search flux --models
```

`--models` prints fully qualified identifiers that you can use in scripts. `--providers` lists the providers that own the selected models. With neither selector, or both, Bildomat groups models under providers. The image/video pair works the same way: neither or both includes both media types. Add `--aliases` to show each model's aliases; listings omit them otherwise.

Ordinary search ignores case and matches the start of each slash-delimited part of a model key and each alias. Search matches aliases even when `--aliases` is not given, so a result can match through an alias that the listing does not show. Use a regular expression to match elsewhere in a name, or across the slash between a provider and a model; a plain term containing a slash matches nothing:

```sh
bild search --regex 'image-(1|2)' --models
bild search --regex 'google/gemini' --models
```

Regular expressions use RE2 syntax and are case-sensitive unless the expression changes that. More than 100 matches for a search term is an error; narrow the term or add a media filter.

Remove models you do not want with `--exclude`, beside a search term or on its own:

```sh
bild search --models kling --exclude kling-2
bild search --models --exclude openrouter
```

An exclusion term matches the way a search term does, so plain text removes only models whose key part or alias begins with it. Use `--regex` to remove by any part of a name; both terms are then expressions:

```sh
bild search --models kling --regex --exclude 'turbo|omni'
```

An exclusion used without a search term has no match limit, so it can pare down the whole catalog.

## Inspect the result

```sh
bild info google/gemini-3.1-flash-image
bild info google --video
```

A model page shows its aliases, accepted options, required values, bounds, input count, and special rules. A provider page compares the options of its models. The aggregator OpenRouter receives a summary in text output instead; `--json` still returns its full selected model details.

Media filters narrow a provider page. They do not hide a model that you name explicitly.

## Select the model

Use a fully qualified identifier when the provider matters:

```sh
bild --model openai/gpt-image-2 --output-path ./poster.png "A geometric travel poster for a coastal railway"
```

`openai/gpt-image-2` and `openrouter/openai/gpt-image-2` use different providers and credentials. They can expose different options even when their names describe the same underlying model. Aliases and model identifiers are case-sensitive; search is more permissive.

Set the selected provider’s credential using [configuration or its environment variable](../reference/configuration.md#credentials). See [model specifiers](../reference/model-specifiers.md) for exact lookup and ambiguity rules.

For machine-readable discovery:

```sh
bild search --json --video veo
bild info google/veo-3.1-generate-preview --json
```

The [provider reference](../reference/providers-and-models.md) lists the full documented catalog.

Revised 2026-10-01
