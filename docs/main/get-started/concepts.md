# Concepts

Image and video models differ in the shapes they accept, the number and kinds of references they use, and how they interpret a request. Bildomat provides one vocabulary for asking for those things. The selected model still determines which capabilities are available.

## Consistent commands, different constraints

`--aspect-ratio 16:9` expresses the same desired shape across models. One model accepts a ratio directly, another requires pixel dimensions, and another offers a short list of sizes. Bildomat uses the model’s declared constraints to translate that request or choose a supported value.

A fully qualified model identifier includes the provider because the provider affects the request. Direct Google and OpenRouter versions of a model can have different options, credentials, and input rules. Switching a model is a command-line change, but it is not a promise of identical behavior or output.

## Visible adjustments

When a value has a meaningful nearest alternative, Bildomat can choose it: the closest ratio, duration, resolution, or a numeric bound. A categorical value with no accepted match can instead be omitted. Unsupported options are ignored. Notices describe those changes, and JSON records adjustments before submission separately from later output notices.

For exploratory work, these adjustments let a request continue. For a workflow that requires a particular setting, inspect the adjustments before accepting the result. Exit status 0 also covers declining the interactive prompt. It does not establish that every requested value was used unchanged or that the visual result meets the task.

Some values are derived even when you did not supply them. For example, direct Veo uses an eight-second duration with input media, and Sora can choose a video size from a reference image. Other omissions leave the provider to choose its own default. See [parameter adjustment](../reference/parameter-adjustment.md).

## Files make the command useful to other tools

Bildomat waits for generation, saves media, and reports the files it produced. A shell can process a directory of inputs. An agent can inspect the catalog, request an edit, read its JSON result, and use the saved image in a page. Both use the same command surface.

Generated files are preserved rather than overwritten. That makes comparisons straightforward, but a website that expects a fixed asset name still needs an explicit selection and publication step. A daily refresh is a script or scheduler workflow around Bildomat; it is not a separate generation mode.

## Frame placement is a model capability

Opening and closing images constrain the ends of a clip. Timed keyframes constrain particular positions within it. A model that only supports endpoints cannot preserve arbitrary timestamps: Bildomat converts numeric times into an available endpoint and reports the conversion. A model without frame support uses an ordinary reference where input media is accepted.

The [keyframe guide](../how-to/use-keyframes.md) shows both kinds of control. Inspect the exact model before depending on one.

Revised 2026-10-01
