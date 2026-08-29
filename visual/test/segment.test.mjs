import assert from 'node:assert/strict';
import { mkdtemp, readFile, stat } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { pathToFileURL } from 'node:url';
import { captureSegments, scrollPositions } from '../src/segment.mjs';

test('scrollPositions includes the exact bottom without duplicates', () => {
  assert.deepEqual(scrollPositions(3100, 900), [0, 630, 1260, 1890, 2200]);
  assert.deepEqual(scrollPositions(2792, 900), [0, 630, 1260, 1892]);
  assert.deepEqual(scrollPositions(600, 900), [0]);
});

test('captureSegments covers every fixture marker with overlapping frames', async () => {
  const fixture = path.resolve('test/fixtures/colours.html');
  const output = await mkdtemp(path.join(tmpdir(), 'visual-segments-'));
  const manifest = await captureSegments(pathToFileURL(fixture).href, output, { settleMs: 10 });

  assert.ok(manifest.frames.length > 1);
  assert.equal(manifest.frames[0].y, 0);
  assert.equal(manifest.frames.at(-1).y, manifest.pageHeight - manifest.viewport.height);
  for (let index = 1; index < manifest.frames.length; index++) {
    const movement = manifest.frames[index].y - manifest.frames[index - 1].y;
    assert.ok(movement > 270, `frame ${index} moves enough to add useful content`);
    assert.ok(movement < 900, `frame ${index} still overlaps its predecessor`);
  }

  const seen = new Set(manifest.frames.flatMap(frame => frame.markers));
  assert.deepEqual([...seen].sort(), [
    'blue-charlie', 'green-bravo', 'purple-echo', 'red-alpha', 'yellow-delta',
  ]);
  for (const frame of manifest.frames) {
    const screenshotPath = path.join(output, frame.filename);
    assert.ok((await stat(screenshotPath)).size > 1000);
    const png = await readFile(screenshotPath);
    assert.equal(png.readUInt32BE(16), manifest.viewport.width);
    assert.equal(png.readUInt32BE(20), manifest.viewport.height);
  }
  assert.equal(JSON.parse(await readFile(path.join(output, 'manifest.json'), 'utf8')).frames.length, manifest.frames.length);
});
