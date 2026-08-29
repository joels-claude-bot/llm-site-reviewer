import path from 'node:path';
import process from 'node:process';
import { captureSegments } from './segment.mjs';

const [url, output = 'generated/visual'] = process.argv.slice(2);
if (!url) {
  console.error('usage: npm run capture -- <url> [output-directory]');
  process.exit(2);
}

const manifest = await captureSegments(url, path.resolve(output));
console.log(JSON.stringify(manifest, null, 2));
