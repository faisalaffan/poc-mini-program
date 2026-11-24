import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  base: '/webview/',
  plugins: [vue()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    assetsInlineLimit: 0,
    cssCodeSplit: false,
    target: 'safari17'
  },
  test: {
    environment: 'jsdom',
    globals: true
  }
})
