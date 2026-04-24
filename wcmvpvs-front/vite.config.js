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
      rollupOptions: {
        output: {
          manualChunks: {
            'admin': [
              './src/components/AdminPortal.vue',
              './src/components/AdminLottery.vue',
              './src/components/MasterPortal.vue',
              './src/components/PartnerPortal.vue',
              './src/components/TicketValidationView.vue',
            ],
            'shop-admin': [
              './src/components/shop/ShopAdminPortal.vue',
            ],
          },
        },
      },
    },
  }
})
