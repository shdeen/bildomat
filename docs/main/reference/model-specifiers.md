# Model names and aliases

`--model` accepts a full key such as `google/gemini-3.1-flash-image`, a bare model ID, or a declared alias such as `gemini`.

```sh
bild info gemini
bild info google/gemini-3.1-flash-image --json
bild list --models
```

Resolution is case-sensitive and follows this order:

1. A provider ID followed by `/` and that provider’s model ID or alias.
2. A unique alias across the catalog.
3. A bare model ID across the catalog, including IDs that themselves contain slashes.

An unsuccessful provider-qualified lookup still falls through to the alias and bare-ID lookups. Use the full key from `bild list --models` to identify both the model and the service handling the request. That listing omits aliases; `bild list --models --aliases` and `bild info MODEL` show them.

OpenRouter IDs include a vendor component. For example, `openrouter/google/gemini-3.1-flash-image` selects OpenRouter, while `google/gemini-3.1-flash-image` selects Google directly. These use different credentials and can expose different options. Inspect the exact key you will run.

Because the provider-qualified step comes first, a bare OpenRouter ID whose vendor component is also a Bildomat provider ID selects that provider when the provider has a matching model ID or alias. `google/veo-3.1` selects Google's `veo-3.1-generate-preview` through its alias `veo-3.1`, and `openai/gpt-image-2` selects OpenAI directly. Use `openrouter/google/veo-3.1` or `openrouter/openai/gpt-image-2` for the OpenRouter models.

## Defaults

An explicit `--model` wins over `default-model` in `~/.bildomat/config.yml`. Without either, Bildomat takes the declared default model of the first provider, in the order `bild list` uses, whose API key is available from the configuration file or the environment. Every provider declares one; the table in [providers and models](providers-and-models.md) lists them, and `bild info PROVIDER --json` reports the same value under `defaultModel`. When no provider's key is available, generation exits 2 and asks for `--model`; catalog commands are unaffected.

`bild help` shows the model in effect under `-m, --model`, and shows no default when none resolves. Because the choice follows the keys that are present, adding or removing a credential can change which model an unqualified run uses. Name the model explicitly when that matters. See [configuration](configuration.md).

## Ambiguous names

When generation finds several matches and standard input and standard error are both terminals, it lists the candidates on standard error and asks for a more specific name, with or without `--json`. A blank reply or end of input exits 2. A replacement that cannot be resolved exits 1, and a replacement that is also ambiguous leads to another request.

When either standard input or standard error is not a terminal, ambiguity exits 1 without asking. Text mode lists the candidates on standard error first; `--json` mode does not list them.

`bild info` always reports ambiguity and exits 1; it never asks interactively. Use full keys in scripts and agent instructions to avoid ambiguity as the catalog grows.

An unknown model exits 1. Use [find a model](../how-to/find-a-model.md) to locate an available identifier.

Revised 2026-10-01
