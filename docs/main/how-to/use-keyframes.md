# Control video with keyframes

Use input images to choose where a clip starts and ends, or to place specific images at times within a clip when the selected model supports it. Endpoint frames and timed keyframes are different capabilities.

## Control the beginning and end

Prepare `paper.png`, showing a sheet of paper, and `crane.png`, showing the folded crane. Supply them to a model that accepts opening and closing frames:

```sh
bild --model google/veo-3.1-generate-preview --duration 8 --input-media first:paper.png --input-media last:crane.png --output-path ./paper-to-crane.mp4 "A sheet of paper folds itself into the supplied origami crane in one continuous shot"
```

The saved clip connects the supplied endpoints. Direct Veo requires an opening frame when a closing frame is present. Its standard and fast models accept up to three inputs; Lite accepts only one and therefore cannot use this two-image example.

The same syntax works for a wide seascape and a lighthouse interior: supply the wide view as `first:` and the lantern room as `last:`, then describe the intended camera movement. The prompt directs the transition; keyframes do not guarantee a particular intermediate motion.

## Place an image at a particular time

Prepare an intermediate image, `half-folded.png`. Direct BFL FLUX.3 accepts numeric keyframe times:

```sh
bild --model bfl/flux-3-video --duration 8 --input-media 0:paper.png --input-media 4:half-folded.png --input-media 8:crane.png --output-path ./folding-sequence.mp4 "The paper folds continuously through the supplied stages into an origami crane"
```

Numeric times require a duration, must be nonnegative, must increase strictly in input order, and cannot exceed the adjusted duration. Direct FLUX.3 accepts up to ten inputs. Its OpenRouter counterpart does not declare `--input-media`; use the direct BFL model for this example.

For predictable timing, give every image an explicit time. Mixed timed and untimed inputs follow the [BFL timing rules](../reference/input-media.md#bfl-timed-keyframes).

## Use another supporting model

Inspect the exact provider/model pair:

```sh
bild info kling/kling-3.0
bild info openrouter/google/veo-3.1
```

Supporting direct Kling and OpenRouter models use opening and closing positions. Numeric prefixes on these models are converted into those positions, not retained as arbitrary timestamps. For example, a single numeric time at or before half the selected duration becomes the opening frame when both positions are free. A later time becomes the closing frame.

A model that accepts media but no frame placement removes the prefix with a notice and uses an ordinary reference; the BFL video tools reject a prefix instead. A model that does not accept input media ignores the inputs without reading them. Check [the complete frame rules](../reference/input-media.md#frame-prefixes), including duplicate positions, input caps, and unprefixed references.

Revised 2026-10-06
