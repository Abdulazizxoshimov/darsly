import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'node:path'

// Dev-server 3000-portda (backend CORS FRONTEND_BASE_URL shunga sozlangan).
// /api va /ws → backend :8087 ga proksilanadi (CORS/mixed muammosi bo'lmaydi, WS bir origin).
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': path.resolve(process.cwd(), 'src') },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': { target: 'http://localhost:8087', changeOrigin: true, ws: true },
    },
  },
})
