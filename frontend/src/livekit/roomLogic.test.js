import { describe, expect, it } from 'vitest'
import { ConnectionQuality } from 'livekit-client'
import {
  applyHandEvent,
  isChatVisible,
  galleryOrder,
  galleryPage,
  GALLERY_PAGE_SIZE,
  linkView,
  localSignature,
  nextPipState,
  participantSignature,
  qualityLabel,
  rateLimiter,
} from './roomLogic'
import { acceptData, decodeData, encodeData, isHostParticipant } from './messaging'

const P = (over = {}) => ({
  identity: 'u1',
  name: 'Ali',
  speaking: false,
  micMuted: true,
  canPublish: false,
  camTrack: null,
  screenTrack: null,
  ...over,
})

describe('participantSignature', () => {
  it('bir xil holat uchun bir xil imzo beradi', () => {
    expect(participantSignature([P()])).toBe(participantSignature([P()]))
  })

  // Har maydon uchun alohida holat: imzoga kirmay qolgan maydon = ekranda
  // eskirgan qiymat, va buni brauzerda tutish deyarli imkonsiz.
  const cases = [
    ['ism', { name: 'Vali' }],
    ['gapirish', { speaking: true }],
    ['mikrofon', { micMuted: false }],
    ['publish huquqi', { canPublish: true }],
    ['host belgisi', { isHost: true }],
    ['kamera treki', { camTrack: { sid: 'TR_1' } }],
    ['ekran treki', { screenTrack: { sid: 'TR_2' } }],
  ]
  it.each(cases)('%s o‘zgarsa imzo ham o‘zgaradi', (_label, over) => {
    expect(participantSignature([P(over)])).not.toBe(participantSignature([P()]))
  })

  it('ishtirokchi qo‘shilsa/chiqsa imzo o‘zgaradi', () => {
    const one = participantSignature([P()])
    const two = participantSignature([P(), P({ identity: 'u2', name: 'Vali' })])
    expect(one).not.toBe(two)
  })

  it('trek obyekti almashsa ham SID bir xil bo‘lsa imzo o‘zgarmaydi', () => {
    // Muhim: LiveKit ba'zan bir xil trek uchun yangi wrapper beradi — bunda
    // qayta render qilishning ma'nosi yo'q.
    const a = participantSignature([P({ camTrack: { sid: 'TR_1' } })])
    const b = participantSignature([P({ camTrack: { sid: 'TR_1' } })])
    expect(a).toBe(b)
  })
})

describe('localSignature', () => {
  const L = (over = {}) => ({ identity: 'me', micOn: false, camOn: false, screenOn: false, canPublish: false, ...over })
  it.each([
    ['mikrofon', { micOn: true }],
    ['kamera', { camOn: true }],
    ['ekran', { screenOn: true }],
    ['ruxsat', { canPublish: true }],
  ])('%s o‘zgarsa imzo o‘zgaradi', (_l, over) => {
    expect(localSignature(L(over))).not.toBe(localSignature(L()))
  })
})

describe('qualityLabel', () => {
  it('excellent va good — ikkalasi ham "good"', () => {
    expect(qualityLabel(ConnectionQuality.Excellent)).toBe('good')
    expect(qualityLabel(ConnectionQuality.Good)).toBe('good')
  })
  it('poor va lost ajratiladi', () => {
    expect(qualityLabel(ConnectionQuality.Poor)).toBe('poor')
    expect(qualityLabel(ConnectionQuality.Lost)).toBe('lost')
  })
  it('noma’lum qiymat "unknown"', () => {
    expect(qualityLabel(undefined)).toBe('unknown')
    expect(qualityLabel(ConnectionQuality.Unknown)).toBe('unknown')
  })
})

describe('linkView', () => {
  it('qayta ulanish sifatdan USTUN — sifat "good" bo‘lsa ham ogohlantiradi', () => {
    expect(linkView(true, 'good')).toMatchObject({ bad: true, label: 'Ulanmoqda…' })
  })
  it('zaif aloqada tejamkor rejimni taklif qiladi', () => {
    const v = linkView(false, 'poor')
    expect(v.bad).toBe(true)
    expect(v.hint).toMatch(/tejamkor/i)
  })
  it('o‘lchanmagan sifatni "Yaxshi" deb ATAMAYDI', () => {
    // Eski xatti-harakat aynan shu edi: har doim "Yaxshi" — ya'ni yolg'on.
    expect(linkView(false, 'unknown').label).toBe('Ulandi')
    expect(linkView(false, 'good').label).toBe('Yaxshi')
  })
})

describe('applyHandEvent', () => {
  const raise = (id, name, at) => ({ kind: 'hand', identity: id, name, raised: true, at })

  it('qo‘l ko‘tariladi va tushiriladi', () => {
    let m = applyHandEvent(new Map(), raise('u1', 'Ali', 1))
    expect(m.has('u1')).toBe(true)
    m = applyHandEvent(m, { kind: 'hand', identity: 'u1', raised: false })
    expect(m.has('u1')).toBe(false)
  })

  it('NAVBAT tartibi ko‘tarilish ketma-ketligi bo‘yicha saqlanadi', () => {
    let m = new Map()
    m = applyHandEvent(m, raise('u1', 'Ali', 1))
    m = applyHandEvent(m, raise('u2', 'Vali', 2))
    m = applyHandEvent(m, raise('u3', 'Guli', 3))
    expect([...m.keys()]).toEqual(['u1', 'u2', 'u3'])
  })

  it('takroriy ko‘tarish navbatdagi o‘rinni O‘ZGARTIRMAYDI', () => {
    let m = new Map()
    m = applyHandEvent(m, raise('u1', 'Ali', 1))
    m = applyHandEvent(m, raise('u2', 'Vali', 2))
    const same = applyHandEvent(m, raise('u1', 'Ali', 9))
    // Aynan o'sha Map — ya'ni React qayta render ham qilmaydi.
    expect(same).toBe(m)
    expect([...same.keys()]).toEqual(['u1', 'u2'])
  })

  it('yo‘q qo‘lni tushirish holatni o‘zgartirmaydi (bekorga render yo‘q)', () => {
    const m = new Map()
    expect(applyHandEvent(m, { kind: 'hand', identity: 'yoq', raised: false })).toBe(m)
  })

  it('lower_all hammasini tozalaydi, bo‘sh holatda esa tegmaydi', () => {
    let m = applyHandEvent(new Map(), raise('u1', 'Ali', 1))
    m = applyHandEvent(m, { act: 'lower_all' })
    expect(m.size).toBe(0)
    expect(applyHandEvent(m, { act: 'lower_all' })).toBe(m)
  })

  it('identity’siz xabar e’tiborsiz qoldiriladi', () => {
    const m = new Map()
    expect(applyHandEvent(m, { kind: 'hand', raised: true })).toBe(m)
    expect(applyHandEvent(m, null)).toBe(m)
  })

  it('ismi o‘zgarsa yangilanadi (qayta ulangan ishtirokchi)', () => {
    let m = applyHandEvent(new Map(), raise('u1', 'Ali', 1))
    m = applyHandEvent(m, raise('u1', 'Ali Valiyev', 2))
    expect(m.get('u1').name).toBe('Ali Valiyev')
  })
})

describe('rateLimiter', () => {
  it('oraliq tugamaguncha rad etadi', () => {
    let t = 1000
    const allow = rateLimiter(1500, () => t)
    expect(allow()).toBe(true) // birinchisi doim o'tadi
    expect(allow()).toBe(false)
    t += 1499
    expect(allow()).toBe(false)
    t += 1
    expect(allow()).toBe(true)
  })

  it('birinchi chaqiruv soat 0 bo‘lsa ham o‘tadi', () => {
    const allow = rateLimiter(1000, () => 0)
    expect(allow()).toBe(true)
  })
})

describe('messaging (wire format)', () => {
  // Bu shakl SERVER bilan shartnoma (`usecase/roomstate.handMsg`) — u o'zgarsa
  // qo'llar jimgina ko'rinmay qo'yadi, shuning uchun aynan shu yerda qotirilgan.
  it('server yasagan "hand" xabari tanib olinadi', () => {
    const fromServer = { kind: 'hand', identity: 'u1', name: 'Ali', raised: true, at: 1730000000000 }
    expect(decodeData(encodeData(fromServer))).toMatchObject({ kind: 'hand', identity: 'u1', raised: true })
  })

  it('server yasagan "reaction" xabarida identity bo‘ladi (o‘z echo‘sini filtrlash uchun)', () => {
    const fromServer = { kind: 'reaction', emoji: '🎉', name: 'Ali', identity: 'u1' }
    expect(decodeData(encodeData(fromServer))).toMatchObject({ kind: 'reaction', identity: 'u1' })
  })

  it('backend ChatMessage (kind’siz) chat sifatida tanib olinadi', () => {
    const backend = {
      id: 'm1',
      sender_name: 'Ustoz',
      body: 'Salom',
      created_at: '2026-07-27T10:00:00Z',
    }
    const out = decodeData(encodeData(backend))
    expect(out).toMatchObject({ kind: 'chat', name: 'Ustoz', body: 'Salom' })
    expect(out.ts).toBe(Date.parse(backend.created_at))
  })

  it('notanish yoki buzuq data null qaytaradi', () => {
    expect(decodeData(encodeData({ kind: 'boshqa' }))).toBeNull()
    expect(decodeData(new Uint8Array([1, 2, 3]))).toBeNull()
  })

  it('server yasagan "lower_all" xabari tanib olinadi', () => {
    expect(decodeData(encodeData({ kind: 'hand', act: 'lower_all' }))).toMatchObject({ act: 'lower_all' })
  })
})

describe('nextPipState', () => {
  it('ekran ulashish boshlanganda o‘zi ochiladi', () => {
    expect(nextPipState('idle', 'share_start')).toBe('open')
  })

  it('ustoz yopsa qayta ochilmaydi', () => {
    const s = nextPipState('open', 'user_close')
    expect(s).toBe('dismissed')
    // Ulashish davom etyapti, lekin biz o'zimizcha ochmaymiz.
    expect(nextPipState(s, 'user_close')).toBe('dismissed')
  })

  it('ulashish tugagach "yopgan edi" xotirasi unutiladi', () => {
    let s = nextPipState('open', 'user_close') // dismissed
    s = nextPipState(s, 'share_stop')
    expect(s).toBe('idle')
    // Keyingi ulashishda yana ochiladi — bu asosiy talab.
    expect(nextPipState(s, 'share_start')).toBe('open')
  })

  it('yopilgan holatda ham yangi ulashish oynani ochadi', () => {
    expect(nextPipState('dismissed', 'share_start')).toBe('open')
  })

  it('qo‘lda ochish har qanday holatdan ishlaydi (auto-ochilish rad etilgan holat)', () => {
    expect(nextPipState('dismissed', 'user_open')).toBe('open')
    expect(nextPipState('idle', 'user_open')).toBe('open')
  })

  it('notanish hodisa holatni o‘zgartirmaydi', () => {
    expect(nextPipState('open', 'nimadir')).toBe('open')
  })
})

// O'qilmagan badge'ining butun ma'nosi shu funksiyada: u "chat ustozning ko'z
// oldidami?" degan savolga javob beradi, "panel ochiqmi?" ga emas.
describe('isChatVisible — o‘qilmagan sanog‘i qoidasi', () => {
  it('suzuvchi oyna yopiq: asosiy panel hal qiladi', () => {
    expect(isChatVisible({ panel: 'chat', pipOpen: false, pipChatOpen: false })).toBe(true)
    expect(isChatVisible({ panel: 'participants', pipOpen: false, pipChatOpen: false })).toBe(false)
    expect(isChatVisible({ panel: 'none', pipOpen: false, pipChatOpen: false })).toBe(false)
  })

  it('suzuvchi oyna ochiq: asosiy oynadagi chat HISOBGA OLINMAYDI', () => {
    // Ustoz ekran ulashyapti va PDF oynasida — brauzerdagi ochiq panel
    // unga hech narsa ko'rsatmayapti, ya'ni xabar O'QILMAGAN.
    expect(isChatVisible({ panel: 'chat', pipOpen: true, pipChatOpen: false })).toBe(false)
  })

  it('suzuvchi oynadagi chat ochiq bo‘lsa xabar o‘qilgan hisoblanadi', () => {
    expect(isChatVisible({ panel: 'none', pipOpen: true, pipChatOpen: true })).toBe(true)
  })
})

// ─── C-1: data-channel ishonch modeli ────────────────────────────────────────
// Xona ichidagi ENG ARZON hujum — mehmon konsoldan ustoz nomidan xabar yuborishi.
// `CanPublishData` hammada yoqilgan, shuning uchun yagona to'siq shu filtr.
describe('acceptData — data-channel ishonch modeli', () => {
  const HOST = { identity: 'mentor-uuid', name: 'Ustoz Ali', metadata: '{"role":"host"}' }
  const GUEST = { identity: 'guest-1', name: 'Mehmon', metadata: '{"role":"participant"}' }

  it('serverdan kelgan chat/hand/reaction qabul qilinadi (participant yo‘q)', () => {
    for (const kind of ['chat', 'hand', 'reaction', 'poll']) {
      expect(acceptData({ kind }, undefined)).toMatchObject({ kind })
    }
  })

  it('mehmon ustoz nomidan chat yubora OLMAYDI', () => {
    const soxta = { kind: 'chat', name: 'Ustoz Ali', senderIdentity: 'mentor-uuid', body: 'Imtihon bekor' }
    expect(acceptData(soxta, GUEST)).toBeNull()
  })

  it('mehmon so‘rovnoma ocha olmaydi va doskani buza olmaydi', () => {
    expect(acceptData({ kind: 'poll', action: 'open', poll: {} }, GUEST)).toBeNull()
    expect(acceptData({ kind: 'wb', act: 'clear' }, GUEST)).toBeNull()
  })

  it('mehmon boshqaning qo‘lini tushira olmaydi (hand faqat serverdan)', () => {
    expect(acceptData({ kind: 'hand', identity: 'boshqa', raised: false }, GUEST)).toBeNull()
  })

  it('host doska va so‘rovnoma yubora oladi', () => {
    expect(acceptData({ kind: 'wb', act: 'clear' }, HOST)).toMatchObject({ kind: 'wb', act: 'clear' })
    expect(acceptData({ kind: 'poll', action: 'open' }, HOST)).toMatchObject({ kind: 'poll' })
  })

  it('host ham chat/hand/reaction ni klientdan yubora olmaydi (faqat server yo‘li)', () => {
    expect(acceptData({ kind: 'chat', body: 'x' }, HOST)).toBeNull()
    expect(acceptData({ kind: 'reaction', emoji: '👍' }, HOST)).toBeNull()
  })

  it('klient xabarida muallif payload‘dan emas, participant‘dan olinadi', () => {
    const out = acceptData({ kind: 'wb', act: 'stroke', name: 'Soxta', identity: 'birov' }, HOST)
    expect(out.name).toBe('Ustoz Ali')
    expect(out.identity).toBe('mentor-uuid')
  })

  it('metadata yo‘q / buzuq / soxta rol — host emas', () => {
    expect(isHostParticipant({ identity: 'x' })).toBe(false)
    expect(isHostParticipant({ identity: 'x', metadata: 'buzuq{' })).toBe(false)
    expect(isHostParticipant({ identity: 'x', metadata: '{"role":"participant"}' })).toBe(false)
    expect(isHostParticipant(null)).toBe(false)
    // Metadata tokendan keladi (server imzolagan) — buni klient o'zgartira olmaydi.
    expect(isHostParticipant(HOST)).toBe(true)
  })

  it('null xabar va notanish tur rad etiladi', () => {
    expect(acceptData(null, undefined)).toBeNull()
    expect(acceptData({ kind: 'nimadir' }, undefined)).toBeNull()
  })

  it('to‘liq zanjir: decode → accept (mehmon soxta backend-shaklidagi chat yuboradi)', () => {
    const wire = encodeData({ sender_name: 'Ustoz Ali', body: 'Havolani bosing', sender_identity: 'mentor-uuid' })
    expect(decodeData(wire)).toMatchObject({ kind: 'chat', name: 'Ustoz Ali' }) // decode ishlaydi…
    expect(acceptData(decodeData(wire), GUEST)).toBeNull() // …lekin filtr uzadi
  })
})

// ─── Galereya sahifalash (Zoom andozasi: 9 plitka/sahifa) ─────────────────────

const G = (identity, over = {}) => ({
  identity,
  name: identity,
  isLocal: false,
  isHost: false,
  speaking: false,
  ...over,
})

describe('galleryOrder', () => {
  it('ustoz → o‘zim → gapirayotganlar → qolganlar tartibida', () => {
    const items = [
      G('men', { isLocal: true }),
      G('a'),
      G('b', { speaking: true }),
      G('ustoz', { isHost: true }),
      G('c'),
    ]
    expect(galleryOrder(items).map((p) => p.identity)).toEqual(['ustoz', 'men', 'b', 'a', 'c'])
  })

  it('barqaror: bir darajadagilar kelish tartibini saqlaydi', () => {
    const items = [G('a'), G('b'), G('c'), G('d')]
    expect(galleryOrder(items).map((p) => p.identity)).toEqual(['a', 'b', 'c', 'd'])
  })

  it('kirish massivini o‘zgartirmaydi', () => {
    const items = [G('a'), G('ustoz', { isHost: true })]
    galleryOrder(items)
    expect(items[0].identity).toBe('a')
  })
})

describe('galleryPage', () => {
  const many = (n) => Array.from({ length: n }, (_, i) => G('p' + i))

  it('9 tagacha bitta sahifa', () => {
    const { items, page, total } = galleryPage(many(9), 0)
    expect(items).toHaveLength(9)
    expect(page).toBe(0)
    expect(total).toBe(1)
  })

  it('10 kishida 2 sahifa: birinchisida 9, ikkinchisida 1', () => {
    expect(galleryPage(many(10), 0).items).toHaveLength(9)
    const p2 = galleryPage(many(10), 1)
    expect(p2.items).toHaveLength(1)
    expect(p2.total).toBe(2)
    expect(p2.items[0].identity).toBe('p9')
  })

  it('chegaradan tashqari sahifa OXIRGI mavjud sahifaga qisiladi (ishtirokchi chiqib ketsa)', () => {
    // 3-sahifada turgan edik, odamlar chiqib 1 sahifa qoldi.
    const { page, items } = galleryPage(many(5), 7)
    expect(page).toBe(0)
    expect(items).toHaveLength(5)
  })

  it('manfiy sahifa 0 ga qisiladi va bo‘sh ro‘yxatda ham yiqilmaydi', () => {
    expect(galleryPage(many(3), -2).page).toBe(0)
    expect(galleryPage([], 0)).toEqual({ items: [], page: 0, total: 1 })
  })

  it('sahifa hajmi standarti 9 (Zoom 3×3)', () => {
    expect(GALLERY_PAGE_SIZE).toBe(9)
  })
})
