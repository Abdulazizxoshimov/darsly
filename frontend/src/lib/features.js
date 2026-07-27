// Feature-bayroqlar (build vaqtida Vite env'dan). Backend sozlamasi bilan mos bo'lishi kerak.
//
// EMAIL_ENABLED: backend'dagi EMAIL_ENABLED bilan mos. SMTP o'chiq bo'lganda (default)
// parol-tiklash xatlari jimgina tashlanadi — shuning uchun "Parolni unutdim" oqimi
// UI'da yashiriladi. SMTP yoqilganda `VITE_EMAIL_ENABLED=true` qilinsa oqim qaytadi.
export const EMAIL_ENABLED = import.meta.env.VITE_EMAIL_ENABLED === 'true'
