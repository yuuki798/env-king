// 环境配置（参考 blome）

export const config = {
  dev: {
    apiBaseUrl: '/api', // 开发时走 Vite proxy -> localhost:8080
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
