import assert from 'node:assert/strict';
import { mkdtemp, readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import process from 'node:process';
import { pathToFileURL } from 'node:url';
import { captureSegments } from '../src/segment.mjs';

const baseURL = process.env.VISUAL_MODEL_BASE_URL ?? 'http://127.0.0.1:9177/v1';
const model = process.env.VISUAL_MODEL ?? 'qwen3-vl';
const output = await mkdtemp(path.join(tmpdir(), 'visual-model-smoke-'));
const fixture = path.resolve('test/fixtures/colours.html');
const manifest = await captureSegments(pathToFileURL(fixture).href, output, { settleMs: 10 });

const content = [{
  type: 'text',
  text: 'List every coloured block visible across these frames as COLOUR: LABEL. Do not infer unseen blocks. Return only the list.',
}];
for (const frame of manifest.frames) {
  const data = (await readFile(path.join(output, frame.filename))).toString('base64');
  content.push({ type: 'image_url', image_url: { url: `data:image/png;base64,${data}` } });
}

const response = await fetch(`${baseURL.replace(/\/$/, '')}/chat/completions`, {
  method: 'POST',
  headers: {
    'content-type': 'application/json',
    ...(process.env.VISUAL_MODEL_API_KEY ? { authorization: `Bearer ${process.env.VISUAL_MODEL_API_KEY}` } : {}),
  },
  body: JSON.stringify({ model, messages: [{ role: 'user', content }], temperature: 0 }),
});
assert.equal(response.ok, true, await response.text());
const body = await response.json();
const answer = body.choices?.[0]?.message?.content?.toUpperCase() ?? '';
for (const expected of ['RED: ALPHA', 'GREEN: BRAVO', 'BLUE: CHARLIE', 'YELLOW: DELTA', 'PURPLE: ECHO']) {
  assert.match(answer.replaceAll(' — ', ': ').replaceAll(' - ', ': '), new RegExp(expected));
}
console.log(answer);
