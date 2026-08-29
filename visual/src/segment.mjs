import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { chromium } from 'playwright';

export const DEFAULT_VIEWPORT = { width: 1280, height: 900 };
export const DEFAULT_STEP_RATIO = 0.7;

export function scrollPositions(pageHeight, viewportHeight, stepRatio = DEFAULT_STEP_RATIO) {
  if (pageHeight <= 0 || viewportHeight <= 0) {
    throw new Error('page and viewport heights must be positive');
  }
  if (stepRatio <= 0 || stepRatio > 1) {
    throw new Error('step ratio must be greater than 0 and no more than 1');
  }

  const bottom = Math.max(0, pageHeight - viewportHeight);
  const step = Math.max(1, Math.floor(viewportHeight * stepRatio));
  const positions = [];
  for (let y = 0; y < bottom; y += step) positions.push(y);
  const minimumUsefulMove = Math.max(1, viewportHeight - step);
  if (positions.length > 1 && bottom - positions.at(-1) < minimumUsefulMove) {
    positions.pop();
  }
  positions.push(bottom);
  return [...new Set(positions)];
}

export async function captureSegments(url, outputDir, options = {}) {
  const viewport = options.viewport ?? DEFAULT_VIEWPORT;
  const stepRatio = options.stepRatio ?? DEFAULT_STEP_RATIO;
  const executablePath = options.executablePath ?? process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH;
  const browser = await chromium.launch({ headless: true, executablePath });

  try {
    const page = await browser.newPage({ viewport });
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.goto(url, { waitUntil: 'domcontentloaded' });
    await page.evaluate(() => document.fonts.ready);
    await mkdir(outputDir, { recursive: true });

    // A real scroll pass hydrates native lazy images and scroll-triggered content.
    let pageHeight = await page.evaluate(() => document.documentElement.scrollHeight);
    let positions = scrollPositions(pageHeight, viewport.height, stepRatio);
    for (const y of positions) {
      await page.evaluate(scrollY => window.scrollTo(0, scrollY), y);
      await page.waitForTimeout(options.settleMs ?? 100);
    }

    // Content may grow during hydration, so calculate the final capture plan afterward.
    pageHeight = await page.evaluate(() => document.documentElement.scrollHeight);
    positions = scrollPositions(pageHeight, viewport.height, stepRatio);
    const frames = [];

    for (const [index, requestedY] of positions.entries()) {
      await page.evaluate(scrollY => window.scrollTo(0, scrollY), requestedY);
      await page.waitForTimeout(options.settleMs ?? 100);
      const state = await page.evaluate(() => {
        const y = Math.round(window.scrollY);
        const markers = [...document.querySelectorAll('[data-review-marker]')]
          .filter(element => {
            const rect = element.getBoundingClientRect();
            return rect.bottom > 0 && rect.top < window.innerHeight;
          })
          .map(element => element.getAttribute('data-review-marker'));
        return { y, markers };
      });
      const filename = `frame-${String(index + 1).padStart(3, '0')}.png`;
      await page.screenshot({
        path: path.join(outputDir, filename),
        animations: 'disabled',
        caret: 'hide',
      });
      frames.push({ filename, requestedY, y: state.y, markers: state.markers });
    }

    const manifest = { url, viewport, stepRatio, pageHeight, frames };
    await writeFile(path.join(outputDir, 'manifest.json'), `${JSON.stringify(manifest, null, 2)}\n`);
    return manifest;
  } finally {
    await browser.close();
  }
}
