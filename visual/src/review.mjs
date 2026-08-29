import { mkdtemp, readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { captureSegments } from './segment.mjs';

export const visualCategories = [
  'RENDERED_BROKEN', 'DIAGRAM_SYNTAX_ERROR', 'TRUNCATION', 'TEXT_LEGIBILITY',
  'FLAT_COLOUR', 'WEAK_HIERARCHY', 'HIGH_DENSITY', 'IMAGE_MISMATCH',
];

export function visualPrompt(url, frames) {
  return `Review the rendered documentation page at ${url} from ${frames} overlapping screenshots in top-to-bottom order.
Report only defects visible in the pixels. Do not review prose style, links, or facts that require outside knowledge.
Allowed categories: ${visualCategories.join(', ')}.
Overlapping screenshots may repeat content: report each defect once.
Return JSON only: {"url":"...","success":true,"findings":[{"category":"CATEGORY","notes":"specific visible evidence","frame":1}]}.`; 
}

export function parseVisualResponse(raw, url) {
  const cleaned = raw.trim().replace(/^```(?:json)?\s*/i, '').replace(/\s*```$/, '');
  const parsed = JSON.parse(cleaned);
  if (!Array.isArray(parsed.findings)) throw new Error('visual response has no findings array');
  for (const finding of parsed.findings) {
    if (!visualCategories.includes(finding.category)) {
      throw new Error(`visual response returned unknown category ${finding.category}`);
    }
  }
  return { url, success: parsed.findings.length === 0, findings: parsed.findings };
}

export async function reviewURL(url, options = {}) {
  const outputDir = options.outputDir ?? await mkdtemp(path.join(tmpdir(), 'visual-review-'));
  const manifest = await (options.capture ?? captureSegments)(url, outputDir, options.captureOptions);
  const content = [{ type: 'text', text: visualPrompt(url, manifest.frames.length) }];
  for (const frame of manifest.frames) {
    const data = (await readFile(path.join(outputDir, frame.filename))).toString('base64');
    content.push({ type: 'image_url', image_url: { url: `data:image/png;base64,${data}` } });
  }

  const baseURL = options.baseURL ?? process.env.VISUAL_MODEL_BASE_URL ?? process.env.LITELLM_BASE_URL ?? 'http://127.0.0.1:9177/v1';
  const model = options.model ?? process.env.VISUAL_MODEL ?? 'qwen3-vl';
  const response = await (options.fetch ?? fetch)(`${baseURL.replace(/\/$/, '')}/chat/completions`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      ...((process.env.VISUAL_MODEL_API_KEY ?? process.env.LITELLM_MASTER_KEY) ? {
        authorization: `Bearer ${process.env.VISUAL_MODEL_API_KEY ?? process.env.LITELLM_MASTER_KEY}`,
      } : {}),
    },
    body: JSON.stringify({ model, messages: [{ role: 'user', content }], temperature: 0 }),
  });
  if (!response.ok) throw new Error(`visual model request failed: ${response.status} ${await response.text()}`);
  const body = await response.json();
  return parseVisualResponse(body.choices?.[0]?.message?.content ?? '', url);
}
