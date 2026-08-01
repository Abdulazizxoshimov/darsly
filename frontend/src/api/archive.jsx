import { api } from './api'

// Dars arxivi (kontrakt №20) — video + chat + materiallar BITTA so'rovda.
//
// Nega bitta: chat xabarining `offset_sec` i videoning `started_at` iga tayanadi.
// Uch alohida so'rovda ular boshqa-boshqa o'qishdan kelib, "vaqtni bosganda
// video boshqa joyga sakraydi" degan sezilmas xatoni tug'dirardi.

/**
 * Server javobini KO'RINISH modeliga aylantiradi (adapter).
 *
 * View'lar shu shaklga tayanadi, server shakliga emas: maydon nomi o'zgarsa
 * yoki `chat`/`materials` `null` bilan kelsa tuzatish FAQAT shu yerda bo'ladi.
 * Shu bilan birga uchta kafolat beriladi va ular pleyer mantig'i uchun shart:
 *  · `chat` — `offset_sec` bo'yicha o'sish tartibida (highlight qidiruvi
 *    ro'yxat tartiblangan deb hisoblaydi);
 *  · `offset_sec` — butun va manfiy emas (video `currentTime` manfiy bo'lolmaydi);
 *  · massivlar hech qachon `null` emas.
 */
export function adaptArchive(raw) {
  const chat = (raw?.chat || []).map((m) => ({
    id: m.id,
    sender_identity: m.sender_identity || '',
    sender_name: m.sender_name || m.sender_identity || 'Noma’lum',
    body: m.body || '',
    // Shaxsiy xabar bayrog'i — view `to_identity` ning o'zi bilan ishlamasin.
    is_private: !!m.to_identity,
    to_identity: m.to_identity || null,
    file: m.file || null,
    created_at: m.created_at || null,
    offset_sec: Math.max(0, Math.floor(Number(m.offset_sec) || 0)),
  }))
  // DIQQAT: `Date.parse(x || 0)` YARAMAYDI — `Date.parse(0)` aslida
  // `Date.parse("0")` bo'lib 2000-yilni beradi, 0 ni emas. Sanasiz xabar
  // shunda 2000-yilgi xabardek tartiblanib qolardi.
  const ts = (v) => Date.parse(v) || 0
  chat.sort((a, b) => a.offset_sec - b.offset_sec || ts(a.created_at) - ts(b.created_at))

  const rec = raw?.recording
  return {
    lesson: raw?.lesson || null,
    recording: rec
      ? {
          id: rec.id,
          status: rec.status || 'failed',
          duration_sec: Number(rec.duration_sec) || 0,
          size_bytes: Number(rec.size_bytes) || 0,
          // Havola FAQAT `ready` da keladi; boshqa holatda `null` bo'lishiga
          // ishonamiz-u, baribir statusga qarab kesamiz — bo'sh `src` bilan
          // `<video>` joriy sahifani yuklashga urinadi (brauzer xatosi).
          url: rec.status === 'ready' ? rec.url || null : null,
          expires_at: rec.expires_at || null,
        }
      : null,
    chat,
    materials: (raw?.materials || []).map((m) => ({
      name: m.name || 'fayl',
      size: Number(m.size) || 0,
      mime: m.mime || '',
      url: m.url || '',
      created_at: m.created_at || null,
    })),
  }
}

export async function getLessonArchive(lessonId) {
  return adaptArchive(await api.get(`/lessons/${lessonId}/archive`))
}

// Chat transkripti (№21) — TXT yoki HTML fayl. Javob JSON emas: `{blob, filename}`.
export function getChatTranscript(lessonId, format = 'txt') {
  return api.blob(`/lessons/${lessonId}/chat/transcript?format=${encodeURIComponent(format)}`)
}
