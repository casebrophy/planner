import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'
import { resolve } from 'path'

const isCapacitor = process.env.CAPACITOR_BUILD === 'true'

export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    // Disable PWA service worker for Capacitor builds — native app serves files locally
    ...(!isCapacitor
      ? [
          VitePWA({
            registerType: 'autoUpdate',
            manifest: false, // we provide our own in public/
            workbox: {
              globPatterns: ['**/*.{js,css,html,svg,png,woff2}'],
              navigateFallback: 'index.html',
              navigateFallbackDenylist: [/^\/api\//],
            },
          }),
        ]
      : []),
  ],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
