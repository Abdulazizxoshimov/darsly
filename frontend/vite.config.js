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
  // Vitest — birlik testlari FAQAT src/ ichida. `e2e/` Playwright'niki: uni bu yerga
  // qo'shsak `test()` ikki xil freymvork ostida chaqirilib xato beradi.
  test: {
    include: ['src/**/*.{test,spec}.{js,jsx}'],
    // `happy-dom` — komponent testlari uchun. `jsdom` EMAS: uning bog'liqligi
    // (`html-encoding-sniffer`) shu Node versiyasida `ERR_REQUIRE_ESM` beradi.
    // Sof mantiq testlari ham shu muhitda ishlaydi (ular DOM'ga tegmaydi).
    environment: 'happy-dom',
    setupFiles: ['./src/test/setup.js'],
  },
})
