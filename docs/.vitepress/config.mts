import { defineConfig } from 'vitepress';
import llmstxt from 'vitepress-plugin-llms';
import { copyOrDownloadAsMarkdownButtons } from 'vitepress-plugin-llms';

export default defineConfig({
  title: ' ',
  description: 'A CLI tool for developing, deploying, and interacting with XRPL smart contracts',

  lang: 'en-US',
  base: '/bedrock/',

  head: [
    ['link', { rel: 'icon', type: 'image/png', href: '/bedrock/favicon.png' }],
    ['link', { rel: 'preconnect', href: 'https://fonts.googleapis.com' }],
    ['link', { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' }],
    [
      'link',
      {
        href: 'https://fonts.googleapis.com/css2?family=Unbounded:wght@400;500;600;700;800&family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap',
        rel: 'stylesheet',
      },
    ],
  ],

  themeConfig: {
    logo: '/commons_ligth_logo.png',

    nav: [
      {
        text: 'Documentation',
        items: [
          { text: 'Introduction', link: '/' },
          { text: 'Getting Started', link: '/guide/getting-started' },
          { text: 'Commands Reference', link: '/guide/commands-reference' },
        ],
      },
      {
        text: 'Links',
        items: [
          { text: 'GitHub', link: 'https://github.com/XRPL-Commons/bedrock' },
          { text: 'XRPL Commons', link: 'https://www.xrpl-commons.org' },
          { text: 'XRPL Docs', link: 'https://xrpl.org/' },
        ],
      },
    ],

    sidebar: [
      {
        text: 'Start Here',
        items: [
          { text: 'Introduction', link: '/' },
          { text: 'Getting Started', link: '/guide/getting-started' },
          { text: 'Quick Reference', link: '/quick-reference' },
        ],
      },
      {
        text: 'Smart Contracts',
        items: [
          { text: 'Building Contracts', link: '/guide/building-contracts' },
          { text: 'ABI Generation', link: '/guide/abi-generation' },
          { text: 'Deploying & Calling', link: '/guide/deployment-and-calling' },
        ],
      },
      {
        text: 'Smart Escrows',
        items: [{ text: 'Smart Escrows', link: '/guide/smart-escrows' }],
      },
      {
        text: 'Smart Vaults',
        items: [{ text: 'Smart Vaults', link: '/guide/smart-vaults' }],
      },
      {
        text: 'Infrastructure',
        items: [
          { text: 'Local Node', link: '/guide/local-node' },
          { text: 'Wallet Management', link: '/guide/wallet' },
        ],
      },
      {
        text: 'Reference',
        items: [{ text: 'Commands Reference', link: '/guide/commands-reference' }],
      },
    ],

    socialLinks: [{ icon: 'github', link: 'https://github.com/XRPL-Commons/bedrock' }],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2025 XRPL Commons',
    },

    search: {
      provider: 'local',
    },
  },

  markdown: {
    lineNumbers: true,
    config(md) {
      md.use(copyOrDownloadAsMarkdownButtons);
    },
  },

  vite: {
    plugins: [
      llmstxt({
        generateLLMsFullTxt: true,
        ignoreFiles: [],
      }),
    ],
  },

  srcExclude: ['**/README.md', 'assets/**'],
});
