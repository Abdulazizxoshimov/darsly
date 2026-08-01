import { describe, expect, it } from 'vitest'
import {
  expiresInDays,
  formatBytes,
  formatExpiry,
  formatOffset,
  RECORDING_STATUS_UZ,
} from './format'

// Yozuv 30 kun saqlanadi. «Qancha qoldi» hisobi UI'ning yagona joyida bo'lishi
// va CHEGARALARDA to'g'ri ishlashi kerak: "bugun o'chadi" bilan "1 kundan keyin"
// ni aralashtirib yuborish foydalanuvchini muhim yozuvdan ayiradi.
const NOW = Date.parse('2026-07-31T12:00:00Z')
const at = (h) => new Date(NOW + h * 3_600_000).toISOString()

describe('formatExpiry / expiresInDays', () => {
  it('muddat yo‘q bo‘lsa null (masalan `processing` yozuv)', () => {
    expect(expiresInDays(null, NOW)).toBeNull()
    expect(formatExpiry(undefined, NOW)).toBeNull()
  })

  it('bir necha kun qolganda kun soni aytiladi', () => {
    expect(formatExpiry(at(24 * 12), NOW)).toBe('12 kundan keyin o‘chadi')
  })

  it('bugun/ertaga chegaralari alohida', () => {
    expect(formatExpiry(at(20), NOW)).toBe('Ertaga o‘chadi') // 0.83 kun → 1
    expect(formatExpiry(at(2), NOW)).toBe('Ertaga o‘chadi')
    expect(formatExpiry(at(-1), NOW)).toBe('Bugun o‘chadi') // muddat o'tib ketgan
  })

  it('yaxlitlash YUQORIGA — qolgan vaqtni kam ko‘rsatmaydi', () => {
    // 3 kun 1 soat → "4 kun" emas, lekin 3 kunni ham yo'qotmaydi.
    expect(expiresInDays(at(24 * 3 + 1), NOW)).toBe(4)
    expect(expiresInDays(at(24 * 3), NOW)).toBe(3)
  })

  it('buzuq sanadan yiqilmaydi', () => {
    expect(expiresInDays('shunchaki matn', NOW)).toBeNull()
  })
})

describe('formatBytes', () => {
  it('kichik fayl KB da (0.0 MB emas)', () => {
    expect(formatBytes(184_320)).toBe('180 KB')
    expect(formatBytes(512)).toBe('512 B')
  })
  it('katta fayl MB da', () => {
    expect(formatBytes(5 * 1024 * 1024)).toBe('5.0 MB')
  })
  it('bo‘sh/noto‘g‘ri qiymat 0 B', () => {
    expect(formatBytes(undefined)).toBe('0 B')
    expect(formatBytes(null)).toBe('0 B')
  })
})

describe('RECORDING_STATUS_UZ', () => {
  it('`expired` holati ham tarjima qilingan', () => {
    expect(RECORDING_STATUS_UZ.expired).toBe('Muddati tugagan')
  })
})

// Pleyer vaqt belgisi — chat xabari yonidagi tugmadagi matn. U video
// shkalasidagi vaqt bilan bir xil ko'rinishda bo'lishi kerak.
describe('formatOffset', () => {
  it('soatgacha m:ss', () => {
    expect(formatOffset(0)).toBe('0:00')
    expect(formatOffset(5)).toBe('0:05')
    expect(formatOffset(125)).toBe('2:05')
    expect(formatOffset(3599)).toBe('59:59')
  })
  it('soatdan oshsa h:mm:ss', () => {
    expect(formatOffset(3600)).toBe('1:00:00')
    expect(formatOffset(3725)).toBe('1:02:05')
  })
  it('yaroqsiz qiymat 0:00', () => {
    expect(formatOffset(-10)).toBe('0:00')
    expect(formatOffset(undefined)).toBe('0:00')
    expect(formatOffset('salom')).toBe('0:00')
  })
})
