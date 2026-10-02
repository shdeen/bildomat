# Output files and reports

Every returned image or video is saved to disk. Output controls change how the result is reported; they do not turn generation into a stream of media bytes on standard output.

## Location and path interpretation

Without `--output-path`, files use the configured `output-dir`, or the working directory when none is configured. Any `--output-path` overrides `output-dir`, including a bare filename such as `hero.png`.

`--output-path`, `-o`, is interpreted in this order:

1. A trailing path separator means a directory.
2. A final path component of `.`, `..`, or `~` means a directory.
3. An existing directory whose final component has no extension means that directory.
4. A `.png`, `.jpg`, `.jpeg`, `.webp`, or `.mp4` suffix, in any letter case, means a filename with a requested extension.
5. Any other extension is discarded, retaining the filename stem. A final component that is only a dot and an extension, such as `.cover`, leaves no stem, so the default stem applies.
6. Otherwise the final component is a stem.

Thus `-o ./out/` chooses a directory, `-o ./out/hero` chooses a stem, and `-o ./hero.gif` saves `hero` with the returned format’s extension. An `.svg` suffix is not an output-format request; vector output that the provider declares as SVG still receives `.svg`. Rule 5 also applies to a dot inside a name: `-o ./hero-v1.5` saves `hero-v1` with the returned format’s extension.

Relative paths, including a bare filename, resolve against the working directory; absolute directories remain absolute. A leading `~/`, or `~` alone, expands to the home directory. Missing parent directories are created before the request is submitted, so a run that fails afterward can leave a new, empty directory. Backslashes and control characters, including DEL, in a requested filename become hyphens. Reported saved paths are absolute.

## Names and collisions

The default stem is `bild-image` for an image and `bild-video` for a video.

A stem, requested or default, is used bare when every file it would produce is free. A collision adds a suffix starting at `02`: `hero.png`, `hero-02.png`, `hero-03.png`, and likewise `bild-image.png`, `bild-image-02.png`. Bildomat never replaces an existing file, whether or not Bildomat created it.

When the provider returns multiple media artifacts, suffixes start at zero: `hero-0.png`, `hero-1.png`, or `bild-image-0.png`, `bild-image-1.png`. This depends on the number actually returned, not merely on `--num-images`. When such a set collides, the collision suffix comes before the index: `hero-02-0.png`, `hero-02-1.png`. A thoughts sidecar does not make a single image use a numbered media suffix.

Collision checks include all media files and the thoughts sidecar. Bildomat writes the sidecar only after every media file has been saved; a media write failure leaves no sidecar. Bildomat creates each file only if its name is still free at the moment of writing, so simultaneous runs do not overwrite each other. If another process takes a later file’s name during the run, that file is saved under the first free two-digit suffix, such as `hero-1-02.png` or `hero-02.md`, and the report shows that name. A later write failure ends the run with exit code 1; completed files remain and are reported. Bildomat attempts to remove an incomplete file after a write failure and reports a cleanup failure. Do not assume every failed run leaves no files.

## Format and extension

A declared returned media type determines the extension, with content detection when needed, then a provider fallback when the format cannot be identified.

| Image provider | Fallback extension |
| --- | --- |
| OpenAI, OpenRouter, Google, Recraft, Kling | `.png` |
| xAI, Black Forest Labs | `.jpg` |
| Sourceful | `.webp` |

Video fallback is `.mp4`. A returned vector image is saved as `.svg` when the provider's response declares the SVG type; content detection does not recognize SVG, so an undeclared vector image receives the provider's fallback extension. Sourceful can also supply the result’s MIME type to identify the format.

On models accepting `--output-format`, an image extension in `--output-path` sets that option before adjustment, overriding a differing format flag. `.jpg` and `.jpeg` request JPEG. `.mp4` requests no model format.

After generation, a requested extension is used only when it agrees with the returned format, and it is then kept as typed: `-o hero.PNG` saves `hero.PNG`. Otherwise the actual format’s extension is used with a notice. JPEG uses `.jpg` unless `.jpeg` was requested. When `--output-path` names a file without an extension, or with an extension that Bildomat discards, an accepted output-format value supplies the preferred extension, subject to the same check. Bildomat does not transcode generated media to satisfy an extension request.

## Thoughts sidecar

`--include-thoughts` writes one `<stem>.md` file beside the media on supported Gemini image models. The catalog identifies those models; Gemini 2.5 Flash Image does not support this option in Bildomat. Unsupported use is omitted with a notice.

The sidecar starts with YAML front matter containing the prompt, model ID, the time of writing as UTC in RFC 3339 form, and parameters after adjustment. The `input-media` value lists the input sources that Bildomat sent, after input capping and prefix removal, as one quoted value. The body contains returned thought texts separated by `---`. When no thoughts are returned, the body is empty. The sidecar participates in collision checks and all saved-file reports.

## Result destinations

| Options | Standard output | Results file |
| --- | --- | --- |
| None | Regular results | None |
| `--json` | One JSON document | None |
| `--print-filename` | Absolute saved paths | None |
| `--save-results PATH` | No regular results | Regular results |
| `--json --save-results PATH` | No regular results | JSON document |
| `--save-results PATH --print-filename` | Absolute saved paths | Regular results |
| `--json --save-results PATH --print-filename` | Absolute saved paths | JSON document |
| `--json --print-filename` | Absolute saved paths; JSON is discarded | None |

`--print-filename` has the short form `-p`.

`--save-results` expands a leading `~/`, or `~` alone, creates missing parent directories, and opens the file before prompt validation. It **truncates an existing report**. The media collision policy does not apply to reports. An open, write, or close failure exits 1 and is reported as text on standard error. When the command line also has a flag error, such as an unknown flag or invalid numeric syntax, the exit code is 2. Other usage errors, such as a missing prompt, are not checked after an open failure. An open failure produces no JSON document, even with `--json`, and ends the run before anything is generated. A `--save-results` placed after a flag that cannot be read is not read: the file is not opened, and the usage error goes where it would go without `--save-results`. When writing the report or the printed paths fails after generation, standard error also lists every saved file with its size.

Regular results begin with the provider and model, written before generation starts, and end with the saved files. With `--save-results`, both parts go into the results file. When standard output is a terminal and none of `--json`, `--print-filename`, or `--save-results` is given, a progress animation runs during generation, followed by a completion line with the elapsed time when generation succeeds. Adjustment notices go to standard error before submission. Changes found later while the request is prepared, such as the conformance of an image downloaded from a URL, are reported on standard error after generation.

Regular saved-file reports include an absolute path and byte count. When standard output is a terminal and neither `--save-results` nor `--print-filename` is given, sizes use readable decimal units. `--print-filename` prints only each successfully saved path, one per line, including a thoughts sidecar. An early failure prints no paths. JSON stores paths and sizes in `artifacts`. Because `--json --print-filename` without `--save-results` discards the JSON document, the error message of a failed run in that combination is discarded with it, and only the exit status reports the failure.

Normal notices and errors use standard error. With `--json`, adjustments and failures go into the JSON document instead; configuration warnings and output-destination failures can still appear on standard error. The one-word confirmation, the list of candidates for an ambiguous model, and the request for a replacement model also use standard error. [Commands](commands.md#generation) gives the conditions for the confirmation, and [model names and aliases](model-specifiers.md#ambiguous-names) gives them for the other two. See [JSON output](json-output.md) and [exit codes](exit-codes-and-errors.md).

Revised 2026-10-01
