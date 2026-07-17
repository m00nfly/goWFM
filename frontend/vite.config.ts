import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '^/share/[^/]+/info(?:\\?.*)?$': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '^/share/[^/]+/files/[^/]+/download-link(?:\\?.*)?$': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '^/share/download/[^/]+/[^/]+(?:\\?.*)?$': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: resolve(__dirname, '../backend/internal/web-dist'),
    emptyOutDir: true,
  },
})
