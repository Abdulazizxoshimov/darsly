import { describe, expect, it } from 'vitest'
import { adaptArchive } from './archive'

// Adapter — view va server o'rtasidagi yagona chegara. U uchta narsani
// KAFOLATLAYDI, chunki pleyer mantig'i shularga tayanadi:
//   · chat `offset_sec` bo'yicha tartiblangan (highlight chiziqli qidiruv qiladi);
//   · `offset_sec` butun va manfiy emas (`video.currentTime` manfiy bo'lolmaydi);
//   · massivlar hech qachon `null` emas.

describe('adaptArchive', () => {
  it('bo‘sh/yaroqsiz javobda ham xavfsiz shakl qaytaradi', () => {
    const a = adaptArchive(null)
    expect(a).toEqual({ lesson: null, recording: null, chat: [], materials: [] })
  })

  it('chatni offset bo‘yicha tartiblaydi va manfiy offsetni 0 ga qisadi', () => {
    const a = adaptArchive({
      chat: [
        { id: 'b', offset_sec: 300, created_at: '2026-08-01T09:05:00Z' },
        { id: 'a', offset_sec: -40, created_at: '2026-08-01T08:59:00Z' },
        { id: 'c', offset_sec: 12.9, created_at: '2026-08-01T09:00:12Z' },
      ],
    })
    expect(a.chat.map((m) => m.id)).toEqual(['a', 'c', 'b'])
    expect(a.chat[0].offset_sec).toBe(0)
    expect(a.chat[1].offset_sec).toBe(12)
  })

  it('shaxsiy xabarni bayroq bilan belgilaydi', () => {
    const a = adaptArchive({ chat: [{ id: 'x', to_identity: 'guest_1' }, { id: 'y' }] })
    expect(a.chat.find((m) => m.id === 'x').is_private).toBe(true)
    expect(a.chat.find((m) => m.id === 'y').is_private).toBe(false)
  })

  it('ismi yo‘q xabarda identity’ga tushadi (bo‘sh joy qolmasin)', () => {
    const a = adaptArchive({ chat: [{ id: 'x', sender_identity: 'guest_7' }] })
    expect(a.chat[0].sender_name).toBe('guest_7')
  })

  // `ready` BO'LMAGAN yozuvda havola bo'lmasligi kerak. Agar server xato bilan
  // eski havolani yuborsa ham `<video src>` ga tushmasligi shart: buzuq havola
  // "video umuman yo'q" degan taassurot berardi va tiklash tugmasini yashirardi.
  it('`ready` bo‘lmagan yozuvda url har doim null', () => {
    const a = adaptArchive({ recording: { id: 'r1', status: 'archived', url: 'https://old' } })
    expect(a.recording.url).toBeNull()
    expect(a.recording.status).toBe('archived')
  })

  it('`ready` yozuvda url va expires_at o‘tadi', () => {
    const a = adaptArchive({
      recording: { id: 'r1', status: 'ready', url: 'https://x.mp4', duration_sec: 3600, size_bytes: 10, expires_at: '2026-09-01T00:00:00Z' },
    })
    expect(a.recording).toMatchObject({ url: 'https://x.mp4', duration_sec: 3600, expires_at: '2026-09-01T00:00:00Z' })
  })
})
