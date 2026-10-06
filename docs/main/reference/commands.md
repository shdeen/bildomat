# Commands and Parsing

```sh
bild [OPTIONS] "prompt"
bild list [OPTIONS]
bild info [OPTIONS] PROVIDER_OR_MODEL
bild search [OPTIONS] [TERM]
bild help [COMMAND]
```

Generation is the root command. Catalog commands read Bildomat's included catalog without contacting a generation service. Their results describe the installed version; provider access still depends on your account.

## Generation

The single positional argument is the complete prompt. Generation flags can come before or after it. Quote a multiword prompt: a second positional argument is a usage error, exit code 2. Use `--` before a prompt beginning with a dash; every argument after `--` is positional. A token that names a subcommand selects that command; neither shell quotes nor a preceding `--` changes that.

```sh
bild --model google/gemini-3.1-flash-image --output-path ./lantern.png "A brass lantern on a wooden table"
bild "A brass lantern on a wooden table" --model gemini --output-path ./lantern.png
bild --model gemini -- "-- a dash-shaped sculpture"
```

A missing prompt, or one that is empty after surrounding whitespace is removed, is a usage error, exit code 2. A few models do not require a prompt, and a run on one of them needs no prompt argument; `bild info MODEL` marks such a model. When standard input and standard error are both terminals, a one-word prompt asks for confirmation on standard error. After surrounding whitespace is removed, `n` or `no` in any letter case declines; any other reply, including an empty one, proceeds. Declining exits 0 without generating. When either standard input or standard error is not a terminal, Bildomat skips the confirmation.

| Flag | Value and default | Effect |
| --- | --- | --- |
| `--model`, `-m` | Model identifier; the default that `bild help` shows under `-m, --model` | Select a model. |
| `--output-path`, `-o` | Path; unset | Choose media directory, filename, or stem. |
| `--print-filename`, `-p` | Boolean; false | Print absolute saved paths on standard output and suppress regular results there. |
| `--save-results` | Path; unset | Write regular results or JSON to a file, replacing existing contents. |
| `--json`, `-j` | Boolean; false | Write a JSON result document. |
| `--help`, `-h` | Boolean; false | Print text help and exit. Must stand alone. |
| `--version`, `-v` | Boolean; false | Print the version and exit. Must stand alone. |

`--help` and `--version` must stand alone: either one given beside another flag or a positional argument is a usage error, exit code 2. The same rule applies to `--help` on `list`, `info`, and `search`.

Without `--model`, the model is the configured `default-model`. Without that setting, it is the declared default model of the first provider, in the order `bild list` shows providers, whose API key is available. When no provider qualifies, generation ends in a usage error, exit code 2, and names `--model`. `bild help` shows the model in effect under `-m, --model`. [Generation flags](generation-flags.md) lists model parameters. [Output files](output-files.md) defines destination precedence and how the three output controls combine.

Before submission, Bildomat asks for confirmation of a one-word prompt, resolves the model, reads the local input files that the model will use, and resolves the output path and creates its directory. In text mode it then prints the provider and model. When the provider's API key is available, it contacts each input URL to identify its media type. It adjusts parameters, checks the provider credential, submits, waits, and saves returned files. Adjustment notices can therefore precede a missing-credential failure, and a failed run can leave a newly created, empty output directory.

Regular results include provider and model identity and saved paths with file sizes. A progress animation appears only when standard output is a terminal and none of `--json`, `--print-filename`, or `--save-results` is active. When the animation is shown and generation succeeds, a completion line with the elapsed time follows it. Notices and failures use standard error. [JSON output](json-output.md) defines structured reporting instead.

## Values and Repeated Flags

A value flag accepts `--model gemini` or `--model=gemini`. It always takes the next argument as its value, even one that begins with a dash: `bild --model --json "a lighthouse"` looks for a model named `--json`. Repeating a single-value flag uses the last value. `--input-media` accumulates values. A Boolean flag alone means true; use `--flag=false` to disable it explicitly. A separate `false` is a positional argument, not a Boolean value. Without a prompt it becomes the prompt; with a prompt it is a second argument, which is a usage error.

Integer flags accept whole numbers, including base-prefixed forms accepted by the parser, such as `0x10`. Use decimal without a leading zero for ordinary counts. Number flags accept decimal or exponent notation. Invalid numeric syntax is a usage error; parseable nonfinite numbers are omitted during adjustment. Supported values and omission rules are in [parameter adjustment](parameter-adjustment.md).

A subcommand's options follow its name, before or after its own argument. A flag placed before the command word, as in `bild --json list`, is a usage error, exit code 2. Short names have command-specific meanings, and they cannot be combined: write `-p -m`, not `-pm`, which is an unknown flag and exits 2.

## List

`bild list` takes no positional arguments.

| Flag | Default | Selection |
| --- | --- | --- |
| `--aliases`, `-a` | false | Include model aliases in text and JSON. |
| `--providers`, `-p` | false | Providers only when used alone. |
| `--models`, `-m` | false | Model keys only when used alone. |
| `--image`, `-i` | false | Image models only when used alone. |
| `--video`, `-v` | false | Video models only when used alone. |
| `--json`, `-j` | false | JSON catalog selection. |
| `--help`, `-h` | false | Text help. Must stand alone. |

Both selector pairs are inclusive: neither flag or both flags includes both alternatives. With providers and models selected, text groups models by provider and medium. Providers-only output excludes providers with no models of the selected medium. Models-only text lists one full key per line. Without `--aliases`, no listing shows aliases; with it, text shows each model's aliases beside the model.

Every listing, in text and JSON, orders providers alphabetically by display name, ignoring case, with aggregators such as OpenRouter last. Within each provider, models are in alphabetical order of model ID.

## Info

`bild info PROVIDER_OR_MODEL` takes exactly one argument. It checks provider IDs first, then [model specifiers](model-specifiers.md).

| Flag | Default | Effect |
| --- | --- | --- |
| `--image`, `-i` | false | Filter a provider page to image models. |
| `--video`, `-v` | false | Filter a provider page to video models. |
| `--json`, `-j` | false | Full catalog detail for the selection. |
| `--help`, `-h` | false | Text help. Must stand alone. |

Both media flags, or neither, includes both media. Media filters do not exclude an explicitly selected model. An unknown or ambiguous model exits 1; ambiguity reports candidate keys on standard error, also with `--json`, without asking for a choice.

A model card shows its provider, identity, aliases, medium, and every supported option with constraints. It marks a model that does not require a prompt. Model cards and provider pages both carry a documentation address for the provider and name the environment variable and the `api-keys` configuration entry that supply its API key. A provider page groups shared constraints and identifies exceptions. An aggregator receives a text summary instead; JSON still includes full model details. The examples in an aggregator summary can differ between runs.

## Search

`bild search` takes a search term as its single positional argument, an exclusion term as the value of `--exclude`, or both. Neither term, or either one typed as an empty string, exits 2. A second positional argument also exits 2. Search supports the same options as `list`, including `--aliases`, plus two of its own.

| Flag | Default | Effect |
| --- | --- | --- |
| `--exclude`, `-x` | unset | Drop the models that this term matches. |
| `--regex`, `-r` | false | Read both terms as regular expressions. |

By default, matching ignores case and compares a term with the beginning of each slash-separated part of a full model key and each alias. Aliases are matched even when `--aliases` is not given. Search does not look at descriptions or match arbitrary substrings. A plain term that contains a slash matches nothing, because each part is compared separately; `bild search google/gemini` finds no models. Use `--regex` to match across parts.

With `--regex`, an expression uses RE2 syntax and matches anywhere in the full key, in one of its slash-separated parts, or in an alias. Matching is case-sensitive unless the expression changes that, for example `(?i)veo`. An expression that does not compile exits 2, whichever term carries it.

The search term chooses the models; the exclusion term then removes from that choice. Without a search term, every model of the selected media is a candidate for removal.

```sh
bild search --models flux
bild search --video --regex 'veo|sora'
bild search kling --exclude kling-2
bild search --models --exclude openrouter
```

With a search term, more than 100 remaining models after the exclusion and the media filter exits 1. Narrow the term or filter the medium. An exclusion term used alone has no such limit. No matches exits 0: text output is empty and JSON contains an empty `providers` array.

## Help and Version

`bild help` prints root help; `bild help list` prints a command's help. An unknown command name exits 2, and so does more than one argument. The help command accepts no options, so `bild help --json` exits 2. The root help page ends with usage tips whose examples come from one provider, chosen afresh on each run.

`bild --version` or `bild -v` prints the version, as in `bild version 0.0.5`. Help and version are text only and cannot be combined with `--json`, since each must stand alone.

Short names differ between commands. On catalog commands, `-v` means video and `-i` means image. On `list` and `search`, `-p` means providers, `-m` means models, and `-a` means aliases. On `search`, `-r` means regex and `-x` means exclude. On generation, `-v` means version, `-i` input media, `-p` print filename, `-m` model, `-a` aspect ratio, and `-r` resolution.

Revised 2026-10-06
