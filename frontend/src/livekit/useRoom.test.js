import { describe, expect, it } from 'vitest'
import { DisconnectReason } from 'livekit-client'
import { canPublishCameraOf, endedByServer, endedReason, mediaErrorClass } from './roomLogic'

// F-1 — `useRoom` ning SOF klassifikatorlari (xavfsizlik/UX qoidalari).
//
// Bular avval `useRoom` ichida yashiringan edi va faqat real LiveKit hodisasi
// bilan, brauzerda tekshirilardi — ya'ni amalda hech qachon. Xatti-harakatni
// o'zgartirmasdan `roomLogic.js` ga ko'chirildi; bu yerda har shox sinaladi.

// ─── canPublishCameraOf — o'quvchi kamerasi ruxsati (XAVFSIZLIK) ──────────────
describe('canPublishCameraOf', () => {
  // Bug: ruxsatsiz o'quvchida `true` qaytsa, ustoz ruxsat bermay turib kamera yonadi.
  it('canPublish=false bo‘lsa kamera YO‘Q (o‘quvchi umuman publish qila olmaydi)', () => {
    expect(canPublishCameraOf({ canPublish: false })).toBe(false)
    expect(canPublishCameraOf({ canPublish: false, canPublishSources: [1] })).toBe(false)
  })

  // Bug: perms yo'q holatda tashlab yuborilsa (throw) xona ekrani oq bo'lardi.
  it('perms yo‘q/undefined — xavfsiz tomonga: kamera YO‘Q', () => {
    expect(canPublishCameraOf(undefined)).toBe(false)
    expect(canPublishCameraOf(null)).toBe(false)
    expect(canPublishCameraOf({})).toBe(false)
  })

  // Bug: host (bo'sh ro'yxat = hammasi) kamerasi bloklansa ustoz video bera olmaydi.
  it('canPublish=true + bo‘sh/yo‘q manba ro‘yxati = HAMMASI (host)', () => {
    expect(canPublishCameraOf({ canPublish: true })).toBe(true)
    expect(canPublishCameraOf({ canPublish: true, canPublishSources: [] })).toBe(true)
  })

  // Bug: server CAMERA'ni raqam (protokol enum=1) yuboradi; faqat stringni
  // tekshirsak ustoz ruxsat bergan o'quvchi kamerasi baribir o'chiq qolardi.
  it('CAMERA manbasini raqam (1) ham, string (\'camera\') ham tan oladi', () => {
    expect(canPublishCameraOf({ canPublish: true, canPublishSources: [1] })).toBe(true)
    expect(canPublishCameraOf({ canPublish: true, canPublishSources: ['camera'] })).toBe(true)
    expect(canPublishCameraOf({ canPublish: true, canPublishSources: [2, 1] })).toBe(true)
  })

  // Bug: FAQAT mikrofon (default o'quvchi) ruxsatida kamera tugmasi ochiq
  // ko'rinsa — bosilganda LiveKit rad etadi va o'quvchi sababini bilmaydi.
  it('faqat mikrofon (kamera YO‘Q manbalar) — kamera bloklangan', () => {
    expect(canPublishCameraOf({ canPublish: true, canPublishSources: [2] })).toBe(false)
    expect(canPublishCameraOf({ canPublish: true, canPublishSources: ['microphone'] })).toBe(false)
  })
})

// ─── endedByServer / endedReason — kick va sessiya-dublikati XONA ICHIDA ──────
describe('endedByServer', () => {
  // Bug: yakuniy sabab "vaqtinchalik" deb qaralsa, chiqarilgan/xona yopilgan
  // foydalanuvchi jimgina qayta ulanishga urinib, xato bannerini ko'rmaydi.
  it('server yopgan/chiqargan sabablar YAKUNIY', () => {
    expect(endedByServer(DisconnectReason.ROOM_DELETED)).toBe(true)
    expect(endedByServer(DisconnectReason.ROOM_CLOSED)).toBe(true)
    expect(endedByServer(DisconnectReason.PARTICIPANT_REMOVED)).toBe(true)
    expect(endedByServer(DisconnectReason.DUPLICATE_IDENTITY)).toBe(true)
  })

  // Bug: vaqtinchalik uzilish (metro/lift/zaif Wi-Fi) YAKUNIY deb qaralsa,
  // guest sessiyasi bilan bosh sahifaga uloqtiriladi — LiveKit qayta ulana turib.
  it('vaqtinchalik/klient tashabbusidagi uzilishlar YAKUNIY EMAS', () => {
    expect(endedByServer(DisconnectReason.CLIENT_INITIATED)).toBe(false)
    expect(endedByServer(DisconnectReason.SIGNAL_CLOSE)).toBe(false)
    expect(endedByServer(DisconnectReason.SERVER_SHUTDOWN)).toBe(false)
    expect(endedByServer(DisconnectReason.CONNECTION_TIMEOUT)).toBe(false)
    expect(endedByServer(undefined)).toBe(false)
  })
})

describe('endedReason', () => {
  // Bug: kick va xona-yopilishi bir xil ko'rsatilsa, chiqarilgan foydalanuvchi
  // "dars tugadi" deb o'ylab, aslida u chiqarilganini bilmaydi.
  it('PARTICIPANT_REMOVED → \'removed\' (ustoz chiqarib yubordi)', () => {
    expect(endedReason(DisconnectReason.PARTICIPANT_REMOVED)).toBe('removed')
  })

  // Bug: sessiya-dublikati (boshqa qurilmada kirildi) "xona yopildi" deb
  // ko'rsatilsa, foydalanuvchi akkaunti ulashilganini payqamaydi.
  it('DUPLICATE_IDENTITY → \'duplicate\' (boshqa joyda ulandi)', () => {
    expect(endedReason(DisconnectReason.DUPLICATE_IDENTITY)).toBe('duplicate')
  })

  it('ROOM_DELETED va ROOM_CLOSED → \'room_deleted\'', () => {
    expect(endedReason(DisconnectReason.ROOM_DELETED)).toBe('room_deleted')
    expect(endedReason(DisconnectReason.ROOM_CLOSED)).toBe('room_deleted')
  })

  // Bug: vaqtinchalik uzilishda banner ko'rsatilsa ("dars tugadi"), foydalanuvchi
  // aslida qayta ulanayotgan xonadan chiqib ketadi.
  it('vaqtinchalik uzilish → null (banner YO‘Q, qayta ulanadi)', () => {
    expect(endedReason(DisconnectReason.CLIENT_INITIATED)).toBeNull()
    expect(endedReason(DisconnectReason.SIGNAL_CLOSE)).toBeNull()
    expect(endedReason(undefined)).toBeNull()
  })
})

// ─── mediaErrorClass — kamera/mikrofon nosozligini AJRATISH ───────────────────
describe('mediaErrorClass', () => {
  // Bug: kamera rad etilib mikrofon ishlaganda umumiy "xatolik" ko'rsatilsa,
  // foydalanuvchi ovoz eshitilayotganini bilmay darsdan chiqib ketadi.
  it('faqat kamera yiqilsa → \'camera\'', () => {
    expect(mediaErrorClass(true, false)).toBe('camera')
  })
  it('faqat mikrofon yiqilsa → \'mic\'', () => {
    expect(mediaErrorClass(false, true)).toBe('mic')
  })
  it('ikkalasi ham yiqilsa → \'both\'', () => {
    expect(mediaErrorClass(true, true)).toBe('both')
  })
  // Bug: hech nima yiqilmaganda banner chiqsa — sof yolg'on ogohlantirish.
  it('ikkalasi ham ishlasa → null (banner YO‘Q)', () => {
    expect(mediaErrorClass(false, false)).toBeNull()
  })
})
