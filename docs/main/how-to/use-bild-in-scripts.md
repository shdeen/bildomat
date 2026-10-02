# Process images in scripts

Use a loop to turn product photographs into consistently styled catalog images. Bildomat handles generation and saving; the shell chooses inputs, output names, and whether to continue after a failure.

## Prepare the batch

Put JPEG source photographs in `products/`. Set an OpenAI credential and inspect the model:

```sh
bild info openai/gpt-image-2
```

The following is a Bash script. It processes each `.jpg` file, prints successful output paths, continues after a failed request, and exits unsuccessfully if any request failed. Filenames containing commas cannot be passed as local input media.

```bash
#!/usr/bin/env bash
shopt -s nullglob
source_images=(products/*.jpg)
if (( ${#source_images[@]} == 0 )); then
  printf '%s\n' 'No JPEG files found in products/' >&2
  exit 1
fi
batch_status=0
for source_image in "${source_images[@]}"; do
  source_name=${source_image##*/}
  if bild --model openai/gpt-image-2 --input-media "$source_image" --num-images 1 \
    --size 1024x1024 --print-filename --output-path "./catalog/${source_name%.jpg}.png" \
    "Preserve the product shape, color, and markings. Photograph it on a white studio background with soft, even lighting"; then
    :
  else
    printf 'Generation failed for %s\n' "$source_image" >&2
    batch_status=1
  fi
done
exit "$batch_status"
```

Existing output images are preserved under numbered names. Repeated runs do not replace previously approved assets. If your website requires one stable filename, select the generated file and publish it through your normal asset workflow.

## Capture structured results

```sh
bild --model openai/gpt-image-2 --json --save-results ./reports/product.json --print-filename --output-path ./catalog/product.png "A studio photograph of a ceramic radio"
```

JSON goes to the report file. Standard output contains one absolute path per successfully saved file. The report file is opened before generation and replaces existing contents, so choose a distinct report path for each run when you need history.

Without `--save-results`, `--json` writes to standard output. Adding `--print-filename` in that combination discards the JSON and prints only paths. The error message of a failed run is discarded with the JSON, so only the exit status reports the failure. Configuration warnings can still appear on standard error. See [JSON output](../reference/json-output.md).

## Check success before using files

Check the exit status and, when using JSON, its `status`. Exit status 0 also covers a declined one-word confirmation, whose JSON status is `canceled`. Bildomat asks for that confirmation only when standard input and standard error are both terminals. Use a complete multiword prompt for unattended generation.

A failure can leave completed files on disk and report their paths. A thoughts sidecar is also a saved file and appears in the path list. Do not treat every returned path as an image or assume one path when a model can return several outputs.

## Compare models or run on a schedule

In Bash, use a fully qualified model identifier for each request:

```bash
prompt='A house built into a chalk cliff, architectural photograph'
for model_id in openai/gpt-image-2 google/gemini-3-pro-image bfl/flux-2-flex; do
  if bild --model "$model_id" --print-filename --output-path "./comparisons/${model_id//\//-}.png" "$prompt"; then
    :
  else
    printf 'Generation failed for %s\n' "$model_id" >&2
  fi
done
```

Each provider needs its own credential. Use your scheduler to invoke a script daily or weekly; Bildomat has no built-in scheduling command. Set the working directory and credentials explicitly in that environment. Use distinct output and report names, and select the generated asset before publishing it.

Revised 2026-10-01
