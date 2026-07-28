import { afterEach } from 'vitest'
import { cleanup } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'

// MAJBURIY: `globals: true` yoqilmagani uchun Testing Library o'zining
// avtomatik tozalashini ro'yxatdan o'tkaza olmaydi. Usiz DOM testlar orasida
// TO'PLANIB boradi va so'rovlar "bir nechta element topildi" deb yiqiladi —
// bu esa test emas, konfiguratsiya xatosi (va uni tashxislash qiyin).
afterEach(cleanup)

// jsdom'da mavjud bo'lmagan, lekin xona komponentlariga kerak bo'ladigan API'lar.
// Ularni bu yerda bir marta stub qilamiz — har testda takrorlamaslik uchun.

// `scrollIntoView` — chat ro'yxati oxirgi xabarga suradi.
if (!Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = () => {}
}

// `crypto.randomUUID` — doska fon bo'laklariga ID beradi.
if (!globalThis.crypto?.randomUUID) {
  globalThis.crypto = { ...globalThis.crypto, randomUUID: () => Math.random().toString(36).slice(2) }
}
