// Nega tizimdan chiqarildik — login sahifasiga OLIB O'TILADIGAN yagona xabar.
//
// Chiqarish `window.location.href = '/auth'` bilan bo'ladi, ya'ni React holati
// butunlay yo'qoladi. Sababni saqlashning yagona yo'li — sessionStorage
// (localStorage EMAS: sabab shu tab uchun, bir martalik).
//
// Nega umuman kerak: backend `SESSION_REVOKED` kodini beradi va u aksariyat
// hollarda «boshqa qurilmada kirildi» degani (bitta akkaunt = bitta sessiya
// mahsulot qoidasi). Foydalanuvchiga «sessiya tugadi» deyish ADASHTIRADI —
// u parolni yoki internetni ayblab, akkaunti ulashilganini bilmay qoladi.

const KEY = 'darsly.logout_reason'

const TEXT = {
  session_revoked:
    'Boshqa qurilmada kirildi. Bitta hisobdan bir vaqtda faqat bitta qurilmada foydalanish mumkin.',
  expired: 'Sessiya muddati tugadi — qaytadan kiring.',
}

export function setLogoutReason(reason) {
  try {
    if (reason && TEXT[reason]) sessionStorage.setItem(KEY, reason)
  } catch {
    /* sessionStorage yopiq (private rejim) — sabab ko'rsatilmaydi, xolos */
  }
}

// Bir martalik o'qish: ko'rsatilgach o'chiriladi (sahifa yangilansa qayta chiqmasin).
export function consumeLogoutReason() {
  try {
    const r = sessionStorage.getItem(KEY)
    if (!r) return null
    sessionStorage.removeItem(KEY)
    return TEXT[r] || null
  } catch {
    return null
  }
}
