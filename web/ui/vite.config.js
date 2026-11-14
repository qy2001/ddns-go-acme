import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const backend = 'http://127.0.0.1:9876'

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: '../dist',
    emptyOutDir: true
  },
  server: {
    port: 5173,
    proxy: {
      '/api': backend,
      '/save': backend,
      '/loginFunc': backend,
      '/logout': backend,
      '/logs': backend,
      '/clearLog': backend,
      '/setLang': backend,
      '/webhookTest': backend,
      '/static': backend,
      '/favicon.ico': backend
    }
  }
})
