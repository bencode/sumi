import { fileURLToPath } from 'node:url'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { defineConfig } from 'vite'

const proxy = {
  '/api': 'http://127.0.0.1:8080',
  '/healthz': 'http://127.0.0.1:8080',
}

export default defineConfig({
  plugins: [svelte()],
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
    proxy,
    fs: {
      allow: [
        fileURLToPath(new URL('.', import.meta.url)),
        fileURLToPath(new URL('../docs/examples', import.meta.url)),
        fileURLToPath(new URL('../docs/compatibility.md', import.meta.url)),
      ],
    },
  },
  preview: { host: '127.0.0.1', port: 4173, strictPort: true, proxy },
})
