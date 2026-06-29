import * as path from 'node:path';
import { defineConfig } from '@rspress/core';
import pluginMermaid from 'rspress-plugin-mermaid';

export default defineConfig({
  root: path.join(__dirname, 'docs'),
  lang: 'en',
  title: 'llm-site-reviewer',
  description: 'Review the rendered site, not the source.',
  search: { codeBlocks: true },
  plugins: [pluginMermaid()],
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
