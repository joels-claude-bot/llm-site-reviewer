# Visual capture spike

This package captures a rendered page as overlapping viewport-sized frames for
vision review. It scrolls the real page first so lazy and scroll-triggered content
has a chance to render, then records each frame and its scroll position in a JSON
manifest.

```bash
npm install
npm run capture -- http://localhost:9811 generated/visual
npm test
```

Run the rendered-page review against an OpenAI-compatible vision endpoint with:

```bash
VISUAL_MODEL=qwen3-vl npm run review -- http://localhost:9811/guide/
```

`npm run test:model` is an opt-in end-to-end smoke test. It captures the colour
fixture and asks an OpenAI-compatible vision endpoint to identify every block.
It defaults to `http://127.0.0.1:9177/v1` and model `qwen3-vl`; override those with
`VISUAL_MODEL_BASE_URL`, `VISUAL_MODEL`, and optionally `VISUAL_MODEL_API_KEY`.
The base URL and key also fall back to the repository's existing
`LITELLM_BASE_URL` and `LITELLM_MASTER_KEY` variables.
