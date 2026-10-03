# Configuration, Environment, and Network Limits

Bildomat reads `~/.bildomat/config.yml`. The file is optional and has three supported top-level keys:

```yaml
default-model: google/gemini-3.1-flash-image
output-dir: ~/Pictures/bild
api-keys:
  google: your-google-api-key
```

| Key | Type | Default and effect |
| --- | --- | --- |
| `default-model` | String | Unset; without it, a run without `--model` uses the default model of the first keyed provider. `bild help` shows the model in effect under `-m, --model`. Accepts the same identifiers as `--model`. The value is not checked when the file is read: `bild help` shows an unknown value as the default, and a run without `--model` then fails as an unknown model, exit code 1. |
| `output-dir` | String | Unset; output otherwise goes to the working directory. Applies only when `--output-path` is omitted. |
| `api-keys` | Mapping from provider ID to string | Unset; nonempty keys override the corresponding environment variable. |

An explicit `--model` wins over configuration. With neither, the model is the declared default of the first provider, in the order `bild list` uses, whose API key is available; generation exits 2 when no provider's key is available. See [model names and aliases](model-specifiers.md#defaults).

Any `--output-path` wins over `output-dir`: `-o hero.png` and `-o ./hero.png` both save in the working directory. A relative configured directory resolves against the working directory, and saved paths are reported absolute either way. A leading `~/`, or `~` alone, expands in output paths and `output-dir`.

There is no `--output-dir` flag and no alternate configuration-file flag. Empty values behave as unset.

A missing configuration file is silent. An unreadable or invalid file produces a warning on standard error and is ignored. When the home directory cannot be determined, for example because `HOME` is unset, Bildomat warns that it cannot locate the file and continues without it. An `--output-path` or `--save-results` path that begins with `~/`, or is `~` alone, then fails with exit code 1. A path such as `~name/x` is never expanded and is used as written. Unknown top-level keys are warned about while recognized settings remain usable. An unknown provider ID in `api-keys` with a nonempty value is warned about and ignored; one with an empty value is ignored silently. Configuration warnings do not themselves change the exit code; they also use standard error during JSON commands.

Bildomat reads the file for generation, `bild help`, and the catalog commands, so warnings appear with all of them. A standalone `bild --version` does not read the file. These runs also end before the file is read and show no configuration warning: an unknown flag or a flag value of the wrong type, a flag placed before a command word, `--help` or `--version` beside other input, and a `--save-results` file that cannot be opened. Other usage errors, such as a missing prompt, an extra argument, or an unknown help topic, occur after the file is read, so its warnings appear before the error.

## Credentials

| Provider ID | Environment variable |
| --- | --- |
| `openai` | `OPENAI_API_KEY` |
| `xai` | `XAI_API_KEY` |
| `openrouter` | `OPENROUTER_API_KEY` |
| `google` | `GOOGLE_API_KEY` |
| `bfl` | `BFL_API_KEY` |
| `sourceful` | `SOURCEFUL_API_KEY` |
| `recraft` | `RECRAFT_API_KEY` |
| `kling` | `KLING_API_KEY` |

A nonempty configured key wins over the environment variable. If neither supplies a key for the selected provider, generation fails with exit code 1. Other providers' keys are not required. OpenRouter uses its own key even for a model from a vendor that also has a direct integration. Kling accepts the bearer credential supplied as its key; Bildomat does not assemble a token from separate access and secret keys.

Catalog commands need no credential. Generation requires network access and whatever model access the provider grants to the account.

## Network Behavior

`HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` configure proxy use. Their lowercase forms, `http_proxy`, `https_proxy`, and `no_proxy`, also work; a nonempty uppercase value takes precedence. Proxy addresses accept a URL or `host:port`. Localhost and loopback destinations bypass proxies. A nonempty `REQUEST_METHOD` marks a CGI environment. There, when `HTTP_PROXY` or `http_proxy` is set, a request to an `http://` address fails instead of using the proxy or connecting directly; `HTTPS_PROXY` still applies to `https://` addresses. On Unix systems other than macOS, `SSL_CERT_FILE` and `SSL_CERT_DIR` select the trusted certificate authorities; macOS uses the system trust store. Individual HTTP requests have an eight-minute timeout, which includes the time spent downloading a result. Response bodies read into memory, and local input files, are limited to 64 MiB. Completed media downloads stream to disk and do not use that response-body limit.

Waiting on an asynchronous job has a 30-minute limit. Bildomat checks at these intervals:

| Service | Interval |
| --- | --- |
| OpenAI and xAI video | 5 seconds |
| OpenRouter video | 30 seconds |
| Google Veo jobs | 15 seconds |
| Google Gemini result file, until it is ready to download | 5 seconds |
| Black Forest Labs | 2 seconds |
| Sourceful | 3 seconds |
| Kling | 5 seconds |

Gemini models answer in a single response rather than through a job. When that response points to a result file instead of containing the media, Bildomat waits for the file to become ready. That wait has its own 30-minute limit, which starts when the response arrives.

While Bildomat waits on a submitted job, it retries through network errors, failures reading a response, and HTTP status 429, 502, 503, and 504, within the job's time limit. A polling response larger than 64 MiB is not retried. The submission itself is never retried, and neither is the download of a finished result, which must complete within its eight-minute request limit. Other provider error responses, failed job states, and malformed responses end polling immediately. A timeout reports the model and the provider's job, and does not prove that the provider stopped the job. See [cancellation and errors](exit-codes-and-errors.md).

Input URLs must be HTTP or HTTPS. Bildomat requests each one that the model will use to identify its media type, and some providers receive media that Bildomat downloads, so a URL must be accessible to Bildomat as well as to the service. Bildomat does not attach a provider key to input-URL requests; see [input media](input-media.md). Provider credentials on result downloads are restricted to the configured service origin where required. A request follows at most nine redirects and fails at the tenth. Credentials are not forwarded to another host or scheme.

## Home and Temporary Files

On Unix systems, `HOME` determines the home directory used for configuration and home-relative output paths. `TMPDIR` selects the system temporary directory; without it, the system default applies. On Windows, `USERPROFILE` determines the home directory, so the configuration file is `%USERPROFILE%\.bildomat\config.yml`, and `TMP` or `TEMP` selects the temporary directory. Bildomat expands a leading `~/` or `~` alone on every system, and on Windows a leading `~\` as well.

Downloaded results first occupy temporary files in that directory, using names beginning `bild-dl-`. Bildomat copies each downloaded result into its output file and then removes the temporary file, so both the temporary directory and the destination need free space. A failed run also removes its temporary downloads. If a removal fails, the run fails with exit code 1 and names the file; media already saved are kept and reported. An abrupt process termination can leave files behind.

Revised 2026-10-01
