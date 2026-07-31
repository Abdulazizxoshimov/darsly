import { describe, expect, it } from 'vitest'
import { acceptData, decodeData, encodeData } from './messaging'

// ISHONCH MODELI (xavfsizlik chegarasi):
// `CanPublishData` xonadagi HAMMADA yoqilgan (doska va so'rovnoma uchun), ya'ni
// har qanday mehmon brauzer konsolidan istalgan payload yubora oladi. LiveKit
// har paketga uni KIM yuborganini o'zi biriktiradi — soxtalashtirib bo'lmaydi:
//
//   participant yo'q → serverdan → ishonchli
//   participant bor  → klientdan → faqat HOST, faqat `wb` / `poll` / `policy`

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

  it('ustozning doska xabari QABUL qilinadi va muallif qayta yoziladi', () => {
    const msg = decodeData(pack({ kind: 'wb', act: 'clear', name: 'Soxta ism' }))
    const out = acceptData(msg, HOST)
    expect(out.identity).toBe('mentor1')
    expect(out.name).toBe('Ustoz')
  })
})
