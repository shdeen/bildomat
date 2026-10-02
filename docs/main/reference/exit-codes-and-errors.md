# Exit Codes and Errors

| Code | Meaning | Examples |
| --- | --- | --- |
| `0` | Command finished successfully, or an interactive prompt was declined | Generation completed; help/catalog displayed; search found no matches; one-word confirmation declined. |
| `1` | Operational failure or cancellation during generation | Unknown or ambiguous model; missing credential; invalid input file; an input URL that cannot be reached or is neither an image nor a video; conflicting frames; no dimensions that satisfy a model's size bounds; provider rejection; timeout; an empty or unavailable download; output write failure; a results file that cannot be opened or written; failure to remove a temporary download; a search term leaves more than 100 matches. |
| `2` | Command-line usage error | Missing or blank prompt on a model that reads one; no model given and no keyed provider with a default; a second positional argument to generation; missing required command argument; a search with neither term, or with an empty term; extra catalog arguments; unknown help topic; more than one argument or any option given to `bild help`; a help or version flag beside other input; a flag before a command word; unknown flag; invalid numeric syntax; invalid search expression; blank response to model disambiguation. |

When one run reports both a usage error and another failure, the exit code is 2. For example, a `--save-results` file that cannot be opened, on a command line that also has an unknown flag, exits 2. Bildomat opens the results file before it checks the prompt and the positional arguments. When the file cannot be opened, those checks do not run, so a missing prompt or an extra argument goes unreported and the run exits 1.

Adjustment notices alone do not fail a run. Unsupported options or values can be omitted, clamped, or replaced before generation. Review [parameter adjustment](parameter-adjustment.md) when a result uses different settings from those supplied.

## Diagnose a Failure

For model lookup failures, run `bild search` and use the exact key from `bild list --models`. For a credential failure, check the selected provider's key and configuration precedence. For media failures, check file existence, format, size, source count, and [frame rules](input-media.md).

A provider can reject a combination that passes local checks. Inspect `bild info MODEL --json` for declared constraints, then use the reported provider message to correct the request. A request can also fail because the provider returns no usable media, no job identity, an unknown job state, an invalid response, a completed job without its result, or an unavailable download.

While waiting on a submitted job, Bildomat retries through network errors, failed response reads, and HTTP status 429, 502, 503, and 504 until the job's time limit runs out. A polling response larger than 64 MiB ends the wait at once. A timeout names the model and the provider's job. See [network behavior](configuration.md#network-behavior).

For file errors, inspect the reported destination and permissions. Completed media files are retained when a later file or report fails, and a failure writing the report or the printed paths lists the saved files on standard error. `--save-results` may already have replaced an earlier report before the failure.

Errors normally use concise messages on standard error. With `--json`, the failure goes into the JSON document instead, for every command: generation failures appear in `errors`, and early errors use the smaller failure document. Parsing stops at the first flag it cannot read, so a flag error produces a JSON document only when `--json` comes before the bad flag; otherwise the error is printed as text. Destination failures can still use standard error. Do not match prose error wording as a stable interface; inspect the exit code and [JSON fields](json-output.md).

## Cancellation

Declining the terminal's one-word prompt confirmation exits 0. JSON records `status: "canceled"` without an error. A script must distinguish that from `completed` if standard input and standard error can both be terminals.

Bildomat handles Ctrl-C from the moment the output directory exists until it has saved the files. By then Bildomat has asked for any one-word confirmation, resolved the model, read the local input files that the model will use, and created the output directory. The handled span covers printing the provider and model, identifying input URLs, adjusting the options, checking the credential, submitting, waiting, downloading, and saving the files.

Within that span, Ctrl-C stops the network request or wait in progress, records a cancellation error, and exits 1. Pressing Ctrl-C again within the span has no further effect. JSON records `status: "canceled"`. Ctrl-C pressed during a local step, such as adjusting the options, takes effect at the next network request. If the run fails at a local step first, for example on a missing credential, it reports that failure instead, with JSON `status: "failed"`. After the result has been received, including any download, no network request remains, so Ctrl-C does not cancel the run: Bildomat saves the files and reports the run as completed, with exit status 0. Cancellation does not guarantee that a submitted remote job stops or that the provider reverses charges.

Outside that span, Ctrl-C ends Bildomat at once, without a report or a JSON document; a shell reports exit status 130. That includes Ctrl-C during the final report, after the files are saved, so the report or JSON document can be missing or incomplete. Files saved before a failure remain, so do not use cancellation as a promise that no output file exists.

Revised 2026-10-01
