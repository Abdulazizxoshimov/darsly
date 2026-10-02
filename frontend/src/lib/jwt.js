// JWT `exp` ni o'qish — FAQAT rejalashtirish uchun (imzo TEKSHIRILMAYDI, u
// serverning ishi). Xona tokeni (`roomToken`) va WS kanali (`ws`) shu bilan
// "muddati tugayapti / tugagan" degan qarorni chiqaradi.

/** `exp` millisekundda yoki `null` (token yo'q / buzuq / exp yo'q). */
export function jwtExpiry(jwt) {
  try {
    const payload = String(jwt || '').split('.')[1]
    if (!payload) return null
    const b64 = payload.replace(/-/g, '+').replace(/_/g, '/')
    const padded = b64 + '='.repeat((4 - (b64.length % 4)) % 4)
    const { exp } = JSON.parse(atob(padded))
    return typeof exp === 'number' && Number.isFinite(exp) ? exp * 1000 : null
  } catch {
    return null
  }
}

/** Muddati o'tganmi. `exp` o'qilmasa `false` — tokenni serverga qoldiramiz. */
export function jwtExpired(jwt, now = Date.now()) {
  const exp = jwtExpiry(jwt)
  return exp !== null && exp <= now
}
