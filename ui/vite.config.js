import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [
    tailwindcss(),
    vue(),
    VitePWA({
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
        globPatterns: ['**/*'],
        navigateFallback: '/index.html',
        navigateFallbackDenylist: [/^\/api\//, /^\/covers\//],
        runtimeCaching: [
          {
            // Cover images rarely change once fetched — serve from cache
            // first so previously-viewed covers work offline, and only
            // fall back to the network for ones not seen yet.
            urlPattern: ({ url }) => url.pathname.startsWith('/covers/'),
            handler: 'CacheFirst',
            options: {
              cacheName: 'comic-covers',
              expiration: {
                maxEntries: 2000,
                maxAgeSeconds: 60 * 60 * 24 * 30,
              },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
          {
            // Library data: prefer a live response, but fall back to the
            // last-seen cached response when offline or slow, so reading
            // orders/comics/series/etc. already viewed remain browsable
            // without a connection.
            urlPattern: ({ url }) =>
              /^\/api\/(reading-orders|comics|comic|series|characters|arcs|collections|dashboard)(\/.*)?$/.test(
                url.pathname,
              ),
            handler: 'NetworkFirst',
            options: {
              cacheName: 'api-data',
              networkTimeoutSeconds: 4,
              expiration: {
                maxEntries: 500,
                maxAgeSeconds: 60 * 60 * 24 * 7,
              },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
          {
            // Marking an issue read/unread while offline: queue the
            // request and replay it automatically once connectivity
            // returns, instead of just failing. The frontend applies the
            // change optimistically (see useComics.js/useDashboard.js) so
            // the UI reflects it immediately either way.
            urlPattern: ({ url }) => /^\/api\/comic\/\d+\/read$/.test(url.pathname),
            method: 'PATCH',
            handler: 'NetworkOnly',
            options: {
              backgroundSync: {
                name: 'comic-read-status-queue',
                options: {
                  maxRetentionTime: 24 * 60,
                },
              },
            },
          },
        ],
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
