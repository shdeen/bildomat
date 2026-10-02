# Choose where files are saved

Use `--output-path` to choose a directory, filename, or filename stem. Bildomat creates missing directories and reports the absolute path of each saved file.

## Save to a directory

```sh
bild --output-path ./assets/concepts/ "A ceramic radio photographed on a white background"
```

The trailing separator makes this a directory, even if it does not exist. An existing directory without an extension also works. A nonexistent extensionless path without a trailing separator names a file stem instead.

## Name the file

```sh
bild --output-path ./assets/hero.png "A coastal observatory at sunrise"
```

A relative output path is relative to the working directory, whether or not it includes a directory: `hero.png` and `./hero.png` both save in the working directory. Absolute and home-relative output paths are also accepted. A configured default output directory does not apply when you give `--output-path`.

With no output path, Bildomat saves in the configured default output directory, or in the working directory when none is configured. It names the file `bild-image` for an image or `bild-video` for a video, adds the returned format’s extension, and appends `-02`, `-03`, and so on when that name is taken.

## Preserve existing files

Bildomat never replaces an existing file. If `hero.png` exists, another run uses `hero-02.png`, then `hero-03.png`. Several returned artifacts receive zero-based suffixes such as `hero-0.png` and `hero-1.png`.

The extension can change to match the returned format. Use the saved path rather than assuming the requested filename was produced.

## Get paths or save a report

```sh
bild --print-filename --output-path ./assets/ "A coastal observatory at sunrise"
bild --json --save-results ./reports/hero.json --print-filename --output-path ./assets/ "A coastal observatory at sunrise"
```

The first command prints only saved paths on standard output. The second saves JSON in the report file and prints paths on standard output. Reports replace existing contents; generated media follow the collision rules above. If the report file cannot be opened, Bildomat prints a text error on standard error, even with `--json`, and generates nothing. `--json --print-filename` without `--save-results` prints paths and discards the JSON document, so the error message of a failed run in that combination is discarded with it; check the exit status. Configuration warnings can still appear on standard error.

See [output files](../reference/output-files.md) for the complete naming and routing rules and [scripts](use-bild-in-scripts.md) for status handling.

Revised 2026-10-01
