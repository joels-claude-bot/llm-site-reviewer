import { reviewURL } from './review.mjs';

const urls = process.argv.slice(2);
if (urls.length === 0) {
  console.error('usage: node src/review-cli.mjs <url> [url...]');
  process.exit(2);
}

const results = [];
for (const url of urls) results.push(await reviewURL(url));
console.log(JSON.stringify(results, null, 2));
