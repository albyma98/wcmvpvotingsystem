import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const backendTarget = (env.VITE_API_BASE_URL || 'http://localhost:3000')
    .replace(/\/api\/?$/, '')
    .replace(/\/$/, '')

  return {
    plugins: [vue()],
    server: {
      host: '0.0.0.0',
      port: 5173,
      proxy: {
        '/api': {
          target: backendTarget,
          changeOrigin: true,
        },
      },
    },
    build: {
      // defineAsyncComponent already splits admin chunks via dynamic import.
      // manualChunks is intentionally omitted to avoid Vite adding
      // <link rel="modulepreload"> for admin files in the user-facing HTML.
    },
  }
})
