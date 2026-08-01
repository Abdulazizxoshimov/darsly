// Himoyalangan endpoint'dan olingan faylni (Blob) diskka saqlash.
//
// Nega `<a href>` emas: transkript endpoint'i JWT talab qiladi va brauzer
// oddiy havolaga `Authorization` header'ini qo'sha olmaydi. Fayl `api.blob`
// bilan olinadi, so'ng shu yerda vaqtinchalik object-URL orqali saqlanadi.
//
// `revokeObjectURL` DARHOL chaqirilmaydi: ba'zi brauzerlar `click()` dan keyin
// yuklashni asinxron boshlaydi va URL o'sha zahoti bekor qilinsa fayl bo'sh
// tushadi. Bir soniya — yetarli va xotira ham oqmaydi.
export function saveBlob(blob, filename) {
  if (!blob || typeof URL?.createObjectURL !== 'function') return false
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename || 'fayl'
  a.style.display = 'none'
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
  return true
}
