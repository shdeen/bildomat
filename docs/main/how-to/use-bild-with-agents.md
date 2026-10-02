# Use Bildomat with an agent

Give a terminal-capable agent access to `bild` so it can create images and videos as part of a design task. The agent can inspect supported models and options through the command instead of constructing provider requests and polling jobs itself.

## Prepare the environment

Install `bild` in the environment where the agent runs and configure the required provider credentials there, as described in [install and configure Bildomat](../get-started/install-and-configure.md). A remote agent or sandbox may have a different `PATH`, home directory, and configuration file from your interactive shell.

Confirm catalog access:

```sh
bild --version
bild list --providers --json
```

Credentials are required for generation, but not for those checks.

## Give the agent a concrete task

For example:

> Use `bild` to generate three candidate hero images for this product page. Use the supplied product photograph as a reference, preserve the product’s markings, and create wide compositions with space for text on the left. Inspect the chosen model’s accepted options first. Save candidates under ./assets/candidates/ and a separate JSON report for each request under ./reports/. Report the actual saved paths and any adjusted settings. Let me choose the image to publish.

Specify source files, output dimensions or aspect ratio, destination, and any details that must remain consistent. If a particular provider is required, include its fully qualified model identifier.

## Discover, inspect, and generate

The agent can find a model and inspect its public constraints:

```sh
bild search --models --json gemini
bild info google/gemini-3.1-flash-image --json
```

Then it can submit an edit:

```sh
bild --model google/gemini-3.1-flash-image --input-media product.jpg --aspect-ratio 16:9 --json --save-results ./reports/hero-01.json --output-path ./assets/candidates/hero-01.png "Preserve the product and its markings. Place it on the right of a quiet studio composition with open space on the left"
```

The command waits for completion and saves the result. The JSON report records the submitted options, adjustments, saved files, and any error. The agent should check the command’s exit status and require JSON `status` to be `completed` before presenting a successful generation.

It can read the report with ordinary file tools. If `jq` is installed, saved paths can be extracted with:

```sh
jq -r '.artifacts[]?.path' ./reports/hero-01.json
```

An adjustment may mean the returned asset differs from the requested dimensions or other constraints. The agent should inspect the report and the resulting media before using the asset. JSON `flags` records submitted values; it is not a complete record of provider defaults.

## Include motion in the design task

Give the agent opening and closing images and ask it to use a model that supports [keyframes](use-keyframes.md). Exact intermediate timestamps require a model that accepts timed keyframes; numeric prefixes on endpoint-only models are converted to opening or closing positions.

For regular refreshes, have the agent prepare a [script](use-bild-in-scripts.md) and invoke it with your scheduler. Generating, choosing, and publishing an asset are separate actions. State which of those actions the agent should perform.

Revised 2026-10-02
