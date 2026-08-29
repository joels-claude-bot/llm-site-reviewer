import assert from 'node:assert/strict';
import { mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { parseVisualResponse, reviewURL, visualPrompt } from '../src/review.mjs';

test('visual prompt limits the model to rendered evidence and deduplication', () => {
  const prompt = visualPrompt('https://example.test/docs', 4);
  assert.match(prompt, /visible in the pixels/);
  assert.match(prompt, /report each defect once/);
  assert.match(prompt, /4 overlapping screenshots/);
});

test('parseVisualResponse normalises success and rejects invented categories', () => {
  assert.deepEqual(parseVisualResponse('```json\n{"success":false,"findings":[{"category":"TRUNCATION","notes":"cut off","frame":2}]}\n```', 'https://example.test'), {
    url: 'https://example.test', success: false,
    findings: [{ category: 'TRUNCATION', notes: 'cut off', frame: 2 }],
  });
  assert.throws(() => parseVisualResponse('{"findings":[{"category":"MADE_UP"}]}', 'x'), /unknown category/);
});

test('reviewURL sends every captured frame to the vision endpoint', async () => {
  const outputDir = await mkdtemp(path.join(tmpdir(), 'visual-review-test-'));
  await writeFile(path.join(outputDir, 'one.png'), Buffer.from('png-one'));
  await writeFile(path.join(outputDir, 'two.png'), Buffer.from('png-two'));
  let request;
  const result = await reviewURL('https://example.test/docs', {
    outputDir,
    capture: async () => ({ frames: [{ filename: 'one.png' }, { filename: 'two.png' }] }),
    fetch: async (url, options) => {
      request = { url, body: JSON.parse(options.body) };
      return new Response(JSON.stringify({
        choices: [{ message: { content: '{"findings":[]}' } }],
      }), { status: 200, headers: { 'content-type': 'application/json' } });
    },
  });
  assert.equal(request.url, 'http://127.0.0.1:9177/v1/chat/completions');
  assert.equal(request.body.messages[0].content.filter(block => block.type === 'image_url').length, 2);
  assert.deepEqual(result, { url: 'https://example.test/docs', success: true, findings: [] });
});
