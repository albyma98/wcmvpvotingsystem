import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { adminV2Controllers } from './build/adminV2Controllers.js'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const remoteAPI = /^https?:\/\//.test(env.VITE_API_BASE_URL || '')
  const backendTarget = (remoteAPI ? env.VITE_API_BASE_URL : 'http://localhost:3000')
    .replace(/\/api\/?$/, '')
    .replace(/\/$/, '')
  const apiProxy = {
    target: backendTarget,
    changeOrigin: true,
    rewrite: remoteAPI ? undefined : path => path.replace(/^\/api/, ''),
  }

  return {
    plugins: [adminV2Controllers(), vue()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      host: '0.0.0.0',
      port: 5173,
      proxy: {
        '^/api/v1/clubs/[^/]+/matches/stream$': {
          ...apiProxy,
          ws: true,
          rewriteWsOrigin: true,
        },
        '/api': apiProxy,
      },
    },
    build: {
      rollupOptions: {
        input: {
          main: fileURLToPath(new URL('./index.html', import.meta.url)),
          arena: fileURLToPath(new URL('./arena-demo/index.html', import.meta.url)),
        },
      },
      // defineAsyncComponent already splits admin chunks via dynamic import.
      // manualChunks is intentionally omitted to avoid Vite adding
      // <link rel="modulepreload"> for admin files in the user-facing HTML.
    },
  }
})
