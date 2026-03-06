import { defineConfig } from 'vitepress'

export default defineConfig({
  lang: 'zh-CN',
  title: 'Env King',
  description: '面向开发团队的智能环境管家与 DevOps Agent',
  themeConfig: {
    nav: [
      { text: '首页', link: '/' },
      { text: '指南', link: '/guide/getting-started' },
      { text: '架构', link: '/architecture/overview' },
      { text: '路线图', link: '/roadmap' },
    ],
    sidebar: {
      '/guide/': [
        {
          text: '指南',
          items: [
            { text: '快速开始', link: '/guide/getting-started' },
            { text: '流水线', link: '/guide/pipeline' },
            { text: '智能 Agent', link: '/guide/agent' },
            { text: 'MCP & Skills', link: '/guide/skills' },
          ],
        },
      ],
      '/architecture/': [
        {
          text: '架构',
          items: [{ text: '总览', link: '/architecture/overview' }],
        },
      ],
      '/': [
        {
          text: '项目',
          items: [
            { text: '首页', link: '/' },
            { text: '路线图', link: '/roadmap' },
          ],
        },
      ],
    },
    outline: [2, 3],
  },
})
