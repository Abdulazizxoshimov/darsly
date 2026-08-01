// Bir martalik UI eslatmalari — «boshqa ko'rsatilmasin» belgilari.
//
// Nega localStorage (sessiya emas): eslatma foydalanuvchi BIR MARTA o'rgangan
// narsa haqida. Uni har dars boshida qayta ko'rsatish — o'rgangan odamni
// jazolash degani. Mobil ilovadagi `PrefsUiPrefs` bilan bir xil naqsh.
//
// Yozuv BLOKLAMAYDI: Safari private rejimida `localStorage` yozish istisno
// beradi — bunda eslatma shunchaki keyingi safar yana chiqadi (xato emas).

const KEY = 'jonly.ui.prefs'

/** «Butun ekranni emas, bitta oynani ulashing» eslatmasi. */
export const SHARE_WINDOW_TIP = 'shareWindowTip'

function read() {
  try {
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : null
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    // Storage o'chirilgan yoki buzilgan JSON — sukut: hech narsa yodda emas.
    return {}
  }
}

export function isTipMuted(name) {
  return read()[name] === true
}

export function muteTip(name) {
  try {
    const prefs = read()
    prefs[name] = true
    localStorage.setItem(KEY, JSON.stringify(prefs))
  } catch {
    // Yozib bo'lmadi — eslatma keyingi safar yana chiqadi. Bu to'xtatuvchi emas.
  }
}
