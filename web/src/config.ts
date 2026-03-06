// 环境配置（参考 blome）

export const config = {
  dev: {
    // 开发时直连后端，避免 Vite 未代理时返回 index.html
    apiBaseUrl: import.meta.env.VITE_API_URL || 'http://localhost:8080/api',
  },
  prod: {
    apiBaseUrl: import.meta.env.VITE_API_URL || '/api', // 生产同源用 /api，跨域需配置完整 URL
  },
}

export const getConfig = () => {
  const mode = import.meta.env.MODE || 'development'
  return mode === 'production' ? config.prod : config.dev
}

export default getConfig()
