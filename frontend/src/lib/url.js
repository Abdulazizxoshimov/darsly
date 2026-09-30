// Serverdan kelgan havolalarni `href`/`src` ga qo'yishdan oldingi tekshiruv.
//
// `file.url` va `rec.url` presigned havolalar — ular ishonchli manbadan
// keladi, lekin `<a href>` ga tekshiruvsiz qo'yilgan satr `javascript:` yoki
// `data:` bo'lsa bosilganda kod bajariladi. Faqat `http(s):` o'tadi; `blob:`
// ham ruxsat etilgan — u shu origin'ning o'zi yaratgan obyektga ishora
// qiladi (mock rejimi shundan foydalanadi) va tashqi manzil bo'la olmaydi.
const SAFE_SCHEME = /^(https?:|blob:)/i

/** `href`/`src` uchun xavfsizmi. Bo'sh/yo'q qiymat ham `false`. */
export function isSafeUrl(url) {
  return typeof url === 'string' && SAFE_SCHEME.test(url.trim())
}

/** Xavfsiz bo'lsa havolaning o'zi, aks holda `null`. */
export function safeUrl(url) {
  return isSafeUrl(url) ? url : null
}
