import { describe, expect, it } from 'vitest'
import { ApiError } from './api'
import { ALLOWED_EXTENSIONS, fileRejectReason, MAX_FILE_BYTES, uploadErrorText } from './chat'

// Klient tomondagi tekshiruv SERVERNIKINI ALMASHTIRMAYDI — u faqat 20 MB'ni
// bekorga yuklab, so'ng 400 olishdan qutqaradi (sekin mobil internetda bu
// bir necha daqiqa isrof). Ro'yxat backend bilan bir xil bo'lishi shart.

const file = (name, size = 1024) => ({ name, size })

describe('fileRejectReason', () => {
  it('ruxsat etilgan turlar o‘tadi', () => {
    for (const ext of ALLOWED_EXTENSIONS) {
      expect(fileRejectReason(file(`x.${ext}`))).toBeNull()
    }
  })

  it('kengaytma katta-kichik harfdan qat’i nazar tanib olinadi', () => {
    expect(fileRejectReason(file('RASM.PNG'))).toBeNull()
  })

  it('ruxsat etilmagan tur rad etiladi', () => {
    expect(fileRejectReason(file('virus.exe'))).toMatch(/qabul qilinmaydi/i)
    expect(fileRejectReason(file('sahifa.html'))).toMatch(/qabul qilinmaydi/i)
    expect(fileRejectReason(file('kengaytmasiz'))).toMatch(/qabul qilinmaydi/i)
  })

  it('20 MB chegarasi: aynan chegara o‘tadi, undan kattasi yo‘q', () => {
    expect(fileRejectReason(file('a.pdf', MAX_FILE_BYTES))).toBeNull()
    expect(fileRejectReason(file('a.pdf', MAX_FILE_BYTES + 1))).toMatch(/20 MB/)
  })

  it('bo‘sh fayl yuborilmaydi', () => {
    expect(fileRejectReason(file('a.pdf', 0))).toMatch(/bo‘sh/i)
  })
})

describe('uploadErrorText', () => {
  it('server sababini o‘zbekchaga aylantiradi (umumiy «xatolik» emas)', () => {
    expect(uploadErrorText(new ApiError('BAD_REQUEST', 'file is too large', 400))).toMatch(/20 MB/)
    expect(
      uploadErrorText(new ApiError('BAD_REQUEST', 'file content does not match its extension', 400)),
    ).toMatch(/mos emas/i)
  })

  it('tezlik cheklovi aniq sabab bilan', () => {
    expect(uploadErrorText(new ApiError('RATE_LIMITED', 'too many', 429))).toMatch(/5 ta/)
  })

  it('bekor qilish xato sifatida ko‘rsatilmaydi', () => {
    expect(uploadErrorText(new ApiError('ABORTED', 'x', 0))).toMatch(/bekor/i)
  })

  it('noma’lum xatoda ham harakat taklif qilinadi', () => {
    expect(uploadErrorText(new Error('boom'))).toMatch(/qayta urinib/i)
  })
})
