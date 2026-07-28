import { describe, expect, it } from 'vitest'
import { ConnectionQuality } from 'livekit-client'
import {
  applyHandEvent,
  linkView,
  localSignature,
  nextPipState,
  participantSignature,
  qualityLabel,
  rateLimiter,
} from './roomLogic'
import { decodeData, encodeData } from './messaging'

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
