# Save a model’s thoughts

On supporting Google image models, `--include-thoughts` saves returned thought text in a Markdown sidecar beside the generated media.

```sh
bild --model google/gemini-3-pro-image --include-thoughts --thinking-level high --output-path ./concept.png "An information-dense transit map for an imaginary tidal city"
```

The supporting models are `google/gemini-3-pro-image`, `google/gemini-3.1-flash-image`, `google/gemini-3.1-flash-lite-image`, and `google/gemini-nano-banana-2.1`. An unsupported model ignores the flag with a notice. `--thinking-level` alone does not request a sidecar; Sourceful's Riverflow 2.5 models also accept that option but do not support `--include-thoughts`.

## Find and read the sidecar

If the image is saved as `concept.png`, the sidecar is `concept.md`. An occupied image or sidecar name causes the run to choose a numbered stem. Bildomat reports both paths, including in `--print-filename` output and JSON `artifacts`.

The sidecar contains the prompt, model, timestamp, adjusted generation options, input sources, and returned thought blocks. Its body is empty when the model returns no thought text. See [output files](../reference/output-files.md#thoughts-sidecar) for the exact fields and collision behavior.

Revised 2026-10-07
