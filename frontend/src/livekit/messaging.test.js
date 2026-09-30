import { describe, expect, it } from 'vitest'
import { acceptData, chatEntryFromServer, decodeData, encodeData } from './messaging'

// ISHONCH MODELI (xavfsizlik chegarasi):
// `CanPublishData` xonadagi HAMMADA yoqilgan (doska va so'rovnoma uchun), ya'ni
// har qanday mehmon brauzer konsolidan istalgan payload yubora oladi. LiveKit
// har paketga uni KIM yuborganini o'zi biriktiradi — soxtalashtirib bo'lmaydi:
//
//   participant yo'q → serverdan → ishonchli
//   participant bor  → klientdan → faqat HOST, faqat `wb` / `poll`

const pack = (obj) => encodeData(obj)
const HOST = { identity: 'mentor1', name: 'Ustoz', metadata: JSON.stringify({ role: 'host' }) }
const GUEST = { identity: 'guest_9', name: 'Bezori', metadata: JSON.stringify({ role: 'participant' }) }

describe('decodeData', () => {
  it('backend ChatMessage faylni ham olib o‘tadi', () => {
    const msg = decodeData(
      pack({
        id: 'c1',
        sender_name: 'Ali',
        sender_identity: 'guest_1',
        body: 'Uy ishi',
        to_identity: null,
        created_at: '2026-07-31T09:12:00Z',
        file: { name: 'uy_ishi.pdf', size: 184320, mime: 'application/pdf', url: 'https://minio/x' },
      }),
    )
    expect(msg.kind).toBe('chat')
    expect(msg.file.name).toBe('uy_ishi.pdf')
  })

  it('yangi server turlari tanib olinadi', () => {
    expect(decodeData(pack({ kind: 'chat_deleted', id: 'c1', lesson_id: 'l1' })).kind).toBe('chat_deleted')
    expect(decodeData(pack({ kind: 'poll_published', results: { total: 3 } })).kind).toBe('poll_published')
  })

  it('buzuq payload null qaytaradi (yiqilmaydi)', () => {
    expect(decodeData(new Uint8Array([1, 2, 3]))).toBeNull()
  })
})

// SHAKL tekshiruvi: avval faqat `kind` tekshirilardi. Buzuq host `poll`
// xabari (masalan `options` yo'q) HAR BIR o'quvchi panelini `options.map` da
// bir vaqtda yiqitardi — bitta buzilgan payload butun sinfni.
describe('decodeData — shakl tekshiruvi', () => {
  const POLL = { id: 'p1', question: 'Savol?', options: ['Ha', "Yo'q"], results_visibility: 'public' }

  it('to‘g‘ri poll open/close qabul qilinadi', () => {
    expect(decodeData(pack({ kind: 'poll', action: 'open', poll: POLL }))).toMatchObject({ action: 'open' })
    expect(decodeData(pack({ kind: 'poll', action: 'close', poll: { id: 'p1' } }))).toMatchObject({ action: 'close' })
  })

  it('poll open: variantlarsiz / savolsiz / poll‘siz → null', () => {
    expect(decodeData(pack({ kind: 'poll', action: 'open', poll: { id: 'p1', question: 'S' } }))).toBeNull()
    expect(decodeData(pack({ kind: 'poll', action: 'open', poll: { id: 'p1', options: ['a'] } }))).toBeNull()
    expect(decodeData(pack({ kind: 'poll', action: 'open', poll: { ...POLL, options: [1, 2] } }))).toBeNull()
    expect(decodeData(pack({ kind: 'poll', action: 'open' }))).toBeNull()
    expect(decodeData(pack({ kind: 'poll', action: 'boshqa', poll: POLL }))).toBeNull()
  })

  it('poll_published: results obyekt bo‘lishi shart, ichidagi poll tekshiriladi', () => {
    expect(decodeData(pack({ kind: 'poll_published', results: { poll: POLL, counts: [1, 0], total: 1 } }))).toBeTruthy()
    expect(decodeData(pack({ kind: 'poll_published' }))).toBeNull()
    expect(decodeData(pack({ kind: 'poll_published', results: 'x' }))).toBeNull()
    expect(decodeData(pack({ kind: 'poll_published', results: { poll: { id: 'p1' } } }))).toBeNull()
  })

  it('chat_deleted id‘siz, reaction emoji‘siz, hand identity‘siz, wb act‘siz → null', () => {
    expect(decodeData(pack({ kind: 'chat_deleted' }))).toBeNull()
    expect(decodeData(pack({ kind: 'reaction', name: 'Ali' }))).toBeNull()
    expect(decodeData(pack({ kind: 'hand', raised: true }))).toBeNull()
    expect(decodeData(pack({ kind: 'wb', pts: [] }))).toBeNull()
    expect(decodeData(pack({ kind: 'wb', act: 'clear' }))).toMatchObject({ act: 'clear' })
  })

  it('massiv / satr / null payload → null', () => {
    expect(decodeData(pack([1, 2]))).toBeNull()
    expect(decodeData(pack('salom'))).toBeNull()
    expect(decodeData(pack(null))).toBeNull()
  })

  it('kind‘siz, lekin ChatMessage bo‘lmagan obyekt → null', () => {
    expect(decodeData(pack({ sender_name: 'Ali' }))).toBeNull()
    expect(decodeData(pack({ id: 'c1', sender_name: 'Ali', body: 5 }))).toBeNull()
  })
})

describe('chatEntryFromServer', () => {
  it('ChatMessage → panel yozuvi (identity, shaxsiy, fayl bilan)', () => {
    const e = chatEntryFromServer({
      id: 'c1',
      sender_identity: 'guest_1',
      sender_name: 'Ali',
      body: 'Salom',
      to_identity: 'host',
      file: { name: 'a.pdf', size: 1, mime: 'application/pdf', url: 'https://x' },
      created_at: '2026-07-31T09:12:00Z',
    })
    expect(e).toEqual({
      id: 'c1',
      name: 'Ali',
      body: 'Salom',
      senderIdentity: 'guest_1',
      toIdentity: 'host',
      file: { name: 'a.pdf', size: 1, mime: 'application/pdf', url: 'https://x' },
      ts: Date.parse('2026-07-31T09:12:00Z'),
    })
  })

  it('bo‘sh to_identity → null, fayl yo‘q → null, identity yo‘q → bo‘sh satr', () => {
    const e = chatEntryFromServer({ id: 'c1', sender_name: 'Ali', body: '', to_identity: '' })
    expect(e).toMatchObject({ toIdentity: null, file: null, senderIdentity: '' })
  })

  it('majburiy maydonsiz → null', () => {
    expect(chatEntryFromServer(null)).toBeNull()
    expect(chatEntryFromServer({ sender_name: 'Ali', body: 'x' })).toBeNull()
  })
})

describe('acceptData — kim nima yubora oladi', () => {
  it('serverdan kelgan chat_deleted QABUL qilinadi', () => {
    const msg = decodeData(pack({ kind: 'chat_deleted', id: 'c1' }))
    expect(acceptData(msg, undefined)).toBeTruthy()
  })

  it('serverdan kelgan poll_published QABUL qilinadi', () => {
    const msg = decodeData(pack({ kind: 'poll_published', results: { total: 3 } }))
    expect(acceptData(msg, undefined)).toBeTruthy()
  })

  // Eng muhim ikki holat: bularsiz istalgan mehmon boshqalarning xabarini
  // ekrandan o'chirib (jimgina senzura) yoki soxta natija ko'rsatib bo'lardi.
  it('MEHMON yuborgan chat_deleted RAD etiladi', () => {
    const msg = decodeData(pack({ kind: 'chat_deleted', id: 'c1' }))
    expect(acceptData(msg, GUEST)).toBeNull()
  })

  it('USTOZ yuborgan chat_deleted ham RAD etiladi — bu server ishi', () => {
    // O'chirish DB'da bo'ladi va server tarqatadi. Klientdan qabul qilsak,
    // xabar faqat EKRANDAN yo'qolardi va tarixda qolib ketardi.
    const msg = decodeData(pack({ kind: 'chat_deleted', id: 'c1' }))
    expect(acceptData(msg, HOST)).toBeNull()
  })

  it('MEHMON yuborgan poll_published RAD etiladi', () => {
    const msg = decodeData(pack({ kind: 'poll_published', results: { total: 999 } }))
    expect(acceptData(msg, GUEST)).toBeNull()
  })

  // Ovoz siyosati — SERVER e'lon qiladi. Avval `policy` host klientidan
  // kutilardi va serverdan kelgan xabar (`participant` yo'q) jimgina tashlab
  // yuborilardi: telefondan o'tilgan darsda o'quvchining mikrofon tugmasi
  // yolg'on ko'rsatardi.
  it('SERVERDAN kelgan policy QABUL qilinadi', () => {
    const msg = decodeData(pack({ kind: 'policy', mute_on_entry: true, allow_self_unmute: false }))
    const out = acceptData(msg, undefined)
    expect(out).toBeTruthy()
    expect(out.allow_self_unmute).toBe(false)
    expect(out.mute_on_entry).toBe(true)
  })

  it('KLIENTdan kelgan policy RAD etiladi — ustoznikidan ham', () => {
    const msg = decodeData(pack({ kind: 'policy', allow_self_unmute: true }))
    expect(acceptData(msg, GUEST)).toBeNull()
    // Ustoz ham: taqiqni server qo'yadi, u ham e'lon qiladi. Klientdan qabul
    // qilsak, mehmon "endi ochsa bo'ladi" degan soxta xabar bilan tugmani
    // yoqib qo'yardi (server esa jimgina qayta mute qilardi).
    expect(acceptData(msg, HOST)).toBeNull()
  })

  it('ustozning doska xabari QABUL qilinadi va muallif qayta yoziladi', () => {
    const msg = decodeData(pack({ kind: 'wb', act: 'clear', name: 'Soxta ism' }))
    const out = acceptData(msg, HOST)
    expect(out.identity).toBe('mentor1')
    expect(out.name).toBe('Ustoz')
  })
})
