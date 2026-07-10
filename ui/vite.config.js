import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [
    vue(),
    VitePWA({
      registerType: 'autoUpdate',
      manifest: {
        name: 'ComicHero',
        short_name: 'ComicHero',
        description: 'Browse, organize, and track your comic collection.',
        theme_color: '#b64237',
        background_color: '#eef2f1',
        display: 'standalone',
        start_url: '/',
        scope: '/',
      },
      workbox: {
        cleanupOutdatedCaches: true,
        navigateFallback: '/index.html',
        navigateFallbackDenylist: [/^\/api\//, /^\/covers\//],
      },
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/covers': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
