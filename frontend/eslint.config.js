import js from '@eslint/js'
import globals from 'globals'
import react from 'eslint-plugin-react'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'

// ESLint (flat config).
//
// Loyihada lint UMUMAN yo'q edi: ishlatilmagan o'zgaruvchilar, o'lik kod va —
// eng muhimi — yetishmayotgan `useEffect`/`useCallback` bog'liqliklari hech kim
// tomonidan tutilmasdi. Oxirgisi React'dagi eng ko'p uchraydigan xato manbai
// va u aynan real-time xonada og'riydi: eskirgan closure ushlab qolingan
// `room` yoki `localId` bilan xabar noto'g'ri filtrlanadi, obuna tozalanmaydi.
export default [
  {
    ignores: [
      'dist/**',
      'playwright-report/**',
      'test-results/**',
      'coverage/**',
      // MSW tomonidan generatsiya qilingan — qo'lda tahrirlanmaydi.
      'public/mockServiceWorker.js',
    ],
  },

  js.configs.recommended,

  {
    files: ['**/*.{js,jsx}'],
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'module',
      globals: { ...globals.browser, ...globals.es2021 },
      parserOptions: { ecmaFeatures: { jsx: true } },
    },
    settings: { react: { version: 'detect' } },
    plugins: {
      react,
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,

      // JSX'da ishlatilgan komponentni "ishlatilmagan" deb hisoblamaslik uchun.
      // Busiz `no-unused-vars` har bir JSX importini yolg'ondan xato deb
      // belgilaydi (199 ta soxta xato — lint umuman foydasiz bo'lardi).
      'react/jsx-uses-react': 'error',
      'react/jsx-uses-vars': 'error',

      // Hook bog'liqliklari — OGOHLANTIRISH emas, XATO.
      //
      // Bu qoida bo'yicha "keyin tuzatamiz" degan narsa yo'q: yetishmayotgan
      // bog'liqlik jimgina eskirgan qiymat ishlatadi va uni brauzerda tutish
      // deyarli imkonsiz. Ataylab chetlab o'tish kerak bo'lgan joylar
      // allaqachon satr-satr `eslint-disable-next-line` bilan belgilangan
      // (masalan bir marta ishlashi kerak bo'lgan boshlang'ich effektlar).
      'react-hooks/exhaustive-deps': 'error',

      // `_` bilan boshlangan argument — ataylab ishlatilmagan.
      'no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],

      // Bo'sh `catch {}` bloki ataylab ishlatiladi (izoh bilan) — masalan
      // ixtiyoriy tozalash. Bo'sh boshqa bloklar esa xato.
      'no-empty': ['error', { allowEmptyCatch: true }],

      // `react-hooks` v7 da paydo bo'lgan yangi qoida. Bu yerdagi 6 ta holat —
      // TASHQI holatni React holatiga ko'chirish (LiveKit hodisasi, so'rov
      // natijasi, `PIP` qo'llab-quvvatlashi). Ular qasddan shunday yozilgan va
      // ularni qayta yozish real-time xonaning eng nozik joylariga tegadi.
      // Shuning uchun hozircha OGOHLANTIRISH: signal ko'rinib turadi, lekin CI
      // ni bloklamaydi. Har birini alohida ko'rib chiqish — alohida ish.
      'react-hooks/set-state-in-effect': 'warn',

      'react-refresh/only-export-components': 'off',
    },
  },

  // Node skriptlari (`final-sweep.mjs` — brauzersiz ishlaydigan yakuniy oqim
  // sinovi). Ular `**/*.{js,jsx}` naqshiga tushmagani uchun HECH QANDAY global
  // olmasdi va har `console`/`process` "aniqlanmagan" deb belgilanardi —
  // natijada lint chiqishi 14 ta soxta xato bilan to'lib, haqiqiylarini
  // ko'rsatmay qo'ygan edi.
  {
    files: ['**/*.mjs'],
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'module',
      globals: { ...globals.node },
    },
  },

  // Testlar va Playwright/Vitest konfiguratsiyasi — Node + test globallari.
  {
    files: [
      '**/*.test.{js,jsx}',
      'e2e/**/*.js',
      'src/test/**/*.{js,jsx}',
      '*.config.js',
      'playwright.config.js',
    ],
    languageOptions: {
      globals: { ...globals.node, ...globals.browser },
    },
  },
]
