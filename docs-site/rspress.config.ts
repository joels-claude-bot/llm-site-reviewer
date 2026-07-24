import * as path from 'node:path';
import { defineConfig } from '@rspress/core';
import pluginMermaid from 'rspress-plugin-mermaid';
import pluginContentPadding from 'rspress-plugin-content-padding';
import pluginFocusMode from 'rspress-plugin-focus-mode';

export default defineConfig({
  root: path.join(__dirname, 'docs'),
  lang: 'en',
  title: 'llm-site-reviewer',
  description: 'Review the rendered site, not the source.',
  search: { codeBlocks: true },
  plugins: [pluginMermaid(), pluginContentPadding(), pluginFocusMode()],
  themeConfig: {
    socialLinks: [
      {
        icon: 'github',
        mode: 'link',
        content: 'https://github.com/joel/llm-site-reviewer',
      },
    ],
  },
});
