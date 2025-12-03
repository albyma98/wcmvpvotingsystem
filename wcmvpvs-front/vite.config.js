import { defineConfig } from 'vite'
import path from 'path'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@vueuse/motion': path.resolve(__dirname, './src/vendor/motion-stub.js'),
      'gsap': path.resolve(__dirname, './src/vendor/gsap-stub.js'),
    }
  },
  server: {
    host: '0.0.0.0',
    port: 5173  // o quello che preferisci
  }
})
