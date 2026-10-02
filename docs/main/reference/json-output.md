# JSON Output

Pass `--json` or `-j` to generation, `list`, `info`, or `search`. Bildomat writes exactly one indented JSON document followed by a newline to the result destination. Angle brackets and ampersands appear as themselves rather than as HTML escape sequences.

`bild help` is text-only and accepts no options, so `bild help --json` is a usage error. So is `--help` or `--version` beside `--json`, because each of those flags must stand alone.

## Generation Document

A completed generation document has this shape:

```json
{
  "timestamp": "2026-08-28T14:32:10.123456Z",
  "durationMs": 2841,
  "status": "completed",
  "provider": "Google",
  "model": "gemini-3.1-flash-image",
  "prompt": "A graphite drawing of a glass greenhouse",
  "flags": {
    "aspect-ratio": "16:9",
    "model": "gemini",
    "output-path": "greenhouse.png",
    "resolution": "1K"
  },
  "artifacts": [
    {
      "path": "/work/greenhouse.png",
      "bytes": 418276
    }
  ]
}
```

| Field | Type | Presence | Meaning |
| --- | --- | --- | --- |
| `timestamp` | String | Always | UTC time at which the run started, after the prompt was checked and before any one-word confirmation, in RFC 3339 format with available fractional seconds. |
| `durationMs` | Integer | Always | Nonnegative elapsed duration in milliseconds, measured from `timestamp`, so it includes time spent answering a confirmation. |
| `status` | String | Always | `completed`, `canceled`, or `failed`. |
| `provider` | String | After provider resolution | Provider display name. |
| `model` | String | After model resolution | Canonical model identifier without the provider prefix. |
| `prompt` | String | Always | Submitted prompt. |
| `flags` | Object | Always | Explicitly supplied `model`, `output-path`, and model options, keyed by long option name without leading hyphens. No other option appears. |
| `adjustments` | Array | When values changed or were omitted | Changes to model options made while the request was prepared, including changes found while conforming an input image downloaded from a URL. |
| `artifacts` | Array | When files were saved | Saved media and thought sidecars in save order. |
| `notices` | Array of strings | When notices exist | Notices after generation, such as a returned file extension differing from the requested extension. Parameter changes belong in `adjustments`. |
| `errors` | Array of strings | When errors exist | User-facing failure messages. |

The values in `flags` retain their JSON type. A repeatable option is an array of strings, with one element for each source after a comma-separated local list is split; other model options are strings, numbers, integers, or Booleans according to their declared type. A supplied number that is not finite has no JSON form and is left out of `flags`; it appears in `adjustments` as an omitted value.

### Adjustment Object

An adjustment has this shape:

```json
{
  "flag": "duration",
  "submitted": "6",
  "used": "8",
  "notice": "Explanation of the duration adjustment"
}
```

| Field | Type | Presence | Meaning |
| --- | --- | --- | --- |
| `flag` | String | Always | Public name of the command flag without leading hyphens. |
| `submitted` | String | When the change replaced or rejected a supplied value that has a text form | Value supplied on the command line. |
| `used` | String | When a replacement value is used | Value used after adjustment. |
| `notice` | String | Always | Complete user-facing explanation of the change. |

An ignored option has neither `submitted` nor `used`; its `notice` says why it was ignored. A rejected value keeps `submitted` and omits `used`. A derived value can omit `submitted`.

### Saved File Object

Each object in `artifacts` describes a generated media file or thought sidecar:

| Field | Type | Meaning |
| --- | --- | --- |
| `path` | String | Absolute path of the saved file. |
| `bytes` | Integer | File size in bytes. |

A model lookup failure during generation retains the timestamp, duration, prompt, supplied flags, status, and errors. It omits `provider` and `model` when those identities could not be resolved.

Some usage errors occur after the generation document exists: no `--model` and no keyed provider with a default, or a blank reply when Bildomat asks for a more specific model. These produce a complete `failed` generation document and exit with status `2`.

An operational failure after model resolution retains the resolved provider and model identity. Error and notice text in these examples is illustrative; wording is not a stable interface:

```json
{
  "timestamp": "2026-08-28T17:51:53.419427Z",
  "durationMs": 0,
  "status": "failed",
  "provider": "Google",
  "model": "gemini-3.1-flash-image",
  "prompt": "A small cabin under an aurora",
  "flags": {},
  "errors": [
    "Provider credential is missing"
  ]
}
```

Declining an interactive one-word prompt produces `status: "canceled"`, has no error, and exits with status `0`. Cancellation handled during generation also produces `status: "canceled"`, records a cancellation error, and exits with status `1`. An interrupt before Bildomat has created the output directory ends Bildomat without any document, and an interrupt after the files are saved can leave the document missing or incomplete; see [cancellation](exit-codes-and-errors.md#cancellation).

## Early Error Document

An error that occurs before a generation or catalog document exists uses this smaller shape:

```json
{
  "status": "failed",
  "errors": [
    "prompt missing"
  ]
}
```

`status` is always `failed`. `errors` is an array of one or more user-facing messages. Like every JSON document, it goes to the result destination, and the error is not also printed on standard error. This holds for catalog commands and command-line usage errors as well as generation, with one exception: parsing stops at the first flag it cannot read, so a flag error produces the document only when `--json` comes before the bad flag. Otherwise the error is printed as text on standard error. `--save-results` works the same way: placed after the bad flag, it is not read, the results file is not opened, and the document or text error goes where it would go without `--save-results`. The document does not change the exit status: a command-line usage error exits with status `2`, while an operational lookup or catalog failure exits with status `1`. A failure to open the `--save-results` file produces no document; Bildomat reports it as text on standard error and exits with status `1`, or `2` when the command line also has a flag error, such as an unknown flag.

## Catalog Document

`list --json`, `search --json`, and `info --json` return a catalog selection. This abbreviated `info` example shows one parameter and its flag definition; a real model result includes all its parameters:

```json
{
  "providers": [
    {
      "id": "google",
      "displayName": "Google",
      "apiKeyEnvVar": "GOOGLE_API_KEY",
      "apiKeyConfigKey": "api-keys.google",
      "defaultModel": "gemini-3.1-flash-image",
      "docsURL": "https://ai.google.dev/gemini-api/docs/image-generation",
      "models": [
        {
          "id": "gemini-3.1-flash-image",
          "name": "Gemini 3.1 Flash Image",
          "media": "image",
          "aliases": [
            "gemini",
            "gemini-3.1",
            "gemini-3.1-flash",
            "gemini-flash",
            "nano-banana"
          ],
          "params": [
            {
              "flagID": "input-media",
              "maxMultiple": 14
            }
          ]
        }
      ]
    }
  ],
  "flags": [
    {
      "flagID": "input-media",
      "flagName": "Input media",
      "dataType": "string",
      "aliases": [
        "i"
      ],
      "description": "File path or HTTP(S) URL of an image or video to edit or use as a reference.",
      "comment": "Multiple local sources may be provided with repeated flags or in a comma-separated list. Repeat the flag for URLs.",
      "textHint": "media-file",
      "allowMultiple": true
    }
  ]
}
```

`list --json` and `search --json` contain provider and model identity records and omit `flags` because they do not include model parameter details. `info --json` includes each selected model's public parameter constraints and the referenced public flag definitions. `list --providers --json` and `search --providers --json` omit `models`.

In `list` and `search` documents, the `providers` array is in alphabetical order of display name, ignoring case, with aggregators last, and each `models` array is in alphabetical order of model ID. An `info` document for a provider keeps that provider's catalog order of models. Catalog documents contain user-facing identifiers and constraints. They do not contain names of fields sent to a provider or transport settings.

### Provider Object

| Field | Type | Presence | Meaning |
| --- | --- | --- | --- |
| `id` | String | Always | Identifier for the provider in the catalog. |
| `displayName` | String | Always | Provider display name. |
| `apiKeyEnvVar` | String | Always | Environment variable read for the provider credential. |
| `apiKeyConfigKey` | String | Always | Configuration file entry that supplies the provider credential, in the form `api-keys.<provider ID>`. |
| `aggregator` | Boolean | When true | The provider represents models from several vendors. |
| `defaultModel` | String | When declared, as every catalog provider declares one | Model used for a run without `--model` when this provider is the first keyed one, written without the provider prefix. |
| `docsURL` | String | When declared | Address of the provider's own documentation. |
| `models` | Array | When models are selected | Selected model records. |

### Model Object

| Field | Type | Presence | Meaning |
| --- | --- | --- | --- |
| `id` | String | Always | Canonical model identifier within the provider. |
| `name` | String | Always | Published model name. |
| `description` | String | When declared | Published model description. |
| `media` | String | Always | `image` or `video`. |
| `family` | String | When declared | Model family. |
| `aliases` | Array of strings | When declared; in `list` and `search`, only with `--aliases` | Alternate values accepted by `--model`. |
| `promptIgnored` | Boolean | When true | The model reads no prompt, so a run on it needs no prompt argument. |
| `docsURL` | String | When declared | Address of the model's own documentation. A text page falls back to the provider's address; this field does not. |
| `params` | Array | In `info` results | Public records for the model's options. |

### Model Parameter Object

| Field | Type | Presence | Meaning |
| --- | --- | --- | --- |
| `flagID` | String | Always | Public name of the command flag without leading hyphens. |
| `required` | Boolean | When true | The catalog declares the option required for the model. Bildomat does not check for the option before submission. |
| `allowedValues` | Array of strings | When declared | Closed set of accepted values. |
| `minValue` | Number | When declared | Minimum numeric value. |
| `maxValue` | Number | When declared | Maximum numeric value. |
| `maxMultiple` | Integer | When declared | Maximum accepted values for a repeatable option. |
| `customSize` | Object | When declared | Dimension limits. |
| `ruleDescription` | String | When declared | Rule that can adjust the value. |
| `modelInfoComment` | String | When declared | Additional model-specific guidance. |

The `customSize` object can contain these fields:

| Field | Meaning |
| --- | --- |
| `maxRatio` | Maximum ratio between the longer and shorter edge. |
| `minEdge` | Minimum edge length in pixels. |
| `maxEdge` | Maximum edge length in pixels. |
| `minPx` | Minimum total pixel count. |
| `maxPx` | Maximum total pixel count. |
| `edgeIncrem` | Required pixel increment for each edge. |
| `longEdge` | Long edge used when Bildomat derives dimensions. |

Zero-valued fields inside `customSize` are omitted. A declared numeric `minValue` or `maxValue` of zero is retained. Undeclared constraints are omitted, not encoded as `null`.

### Flag Definition Object

| Field | Type | Presence | Meaning |
| --- | --- | --- | --- |
| `flagID` | String | Always | Long command-line name without leading hyphens. |
| `flagName` | String | Always | Display name. |
| `dataType` | String | Always | `string`, `number`, `integer`, or `boolean`. |
| `aliases` | Array of strings | When declared | Short names without a leading hyphen. |
| `description` | String | Always | Option description. |
| `exampleValues` | Array of strings | When declared | Example command-line values. |
| `comment` | String | When declared | Additional general guidance. |
| `textHint` | String | When declared | Value placeholder used in help. |
| `allowMultiple` | Boolean | When true | The option can be repeated. |

## Result Routing

Without `--save-results`, the JSON document is written to standard output. With `--save-results PATH`, it is written to that file and replaces the file's previous contents. `--save-results` and `--print-filename` are generation options; `list`, `info`, and `search` do not accept them, and giving one there is a usage error, exit code 2.

When a script also needs saved paths on standard output, combine `--json --save-results PATH --print-filename`. On an early failure or a run that saves no file, `--print-filename` leaves standard output empty; the saved JSON document still records the result when the results file could be opened.

`--print-filename` without `--save-results` suppresses the regular result destination. Therefore, `--json --print-filename` prints saved paths but does not also print a JSON document. The error message of a failed run in that combination is discarded with the JSON document, and only the exit status reports the failure.

Configuration warnings and failures writing or closing a result destination can still appear on standard error. After a failure writing the destination, standard error also lists every saved file. A failure to open the results file produces no JSON document, only a text error on standard error. A JSON result does not make exit-code checks optional. Files saved before a later failure remain in `artifacts`. Defaults not explicitly supplied do not appear in `flags`; use `adjustments` to inspect derived values.

Revised 2026-10-01
