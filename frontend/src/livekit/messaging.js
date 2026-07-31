// LiveKit data-channel xabar sxemasi (client yuboradigan).
// Backend host-chat'ni entity.ChatMessage shaklida (kind'siz) broadcast qiladi — uni ajratib olamiz.

const enc = new TextEncoder()
const dec = new TextDecoder()

export function encodeData(msg) {
  // new Uint8Array(...) — ArrayBuffer bilan (publishData tipi uchun)
  return new Uint8Array(enc.encode(JSON.stringify(msg)))
}

// Kelgan data'ni normalizatsiya qiladi. Qaytadi: {kind:'chat'|'reaction'|'hand'|'poll'|'wb', ...} yoki null.
export function decodeData(payload) {
  try {
    const obj = JSON.parse(dec.decode(payload))
    if (obj && typeof obj === 'object') {
      if (
        obj.kind === 'chat' ||
        obj.kind === 'chat_deleted' ||
        obj.kind === 'reaction' ||
        obj.kind === 'hand' ||
        obj.kind === 'poll' ||
        obj.kind === 'poll_published' ||
        obj.kind === 'wb' ||
        obj.kind === 'policy'
      ) {
        return obj
      }
      // Backend ChatMessage (kind yo'q, sender_name + body bor).
      // Fayl biriktirilgan bo'lsa `file` ham olib o'tiladi (nom/hajm/mime/url).
      // `sender_identity` va `to_identity` ham olib o'tiladi: klient shular
      // asosida "bu meniki" va "bu shaxsiy" degan qarorni chiqaradi. Ularsiz
      // o'z xabaringiz begonanikidek ko'rinardi.
      if (typeof obj.sender_name === 'string' && typeof obj.body === 'string') {
        return {
          kind: 'chat',
          id: obj.id,
          name: obj.sender_name,
          body: obj.body,
          senderIdentity: obj.sender_identity || '',
          toIdentity: obj.to_identity || null,
          file: obj.file || null,
          ts: Date.parse(obj.created_at) || Date.now(),
        }
      }
    }
  } catch {
    /* e'tiborsiz */
  }
  return null
}

// Qaysi turlar SERVERDAN keladi (backend `SendData`), qaysilari HOST klientidan.
// Ro'yxatda yo'q kombinatsiya — rad etiladi.
// `chat_deleted` va `poll_published` — FAQAT serverdan: birinchisi begona
// klientga istalgan xabarni ekrandan o'chirish imkonini berardi (jimgina
// senzura), ikkinchisi esa soxta natija ko'rsatardi.
const SERVER_KINDS = new Set(['chat', 'chat_deleted', 'reaction', 'hand', 'poll', 'poll_published'])
// 'policy' — ustoz klienti dars ovoz siyosatini (allow_self_unmute) tarqatadi:
// backend buni push qilmaydi, o'quvchi esa mikrofon tugmasi holatini bilishi kerak.
// Server baribir ENFORCE qiladi — bu faqat UI ko'rsatkichi, xavfsizlik chegarasi emas.
const HOST_KINDS = new Set(['wb', 'poll', 'policy'])

/**
 * Ishtirokchi HOST'mi — token metadata'si bo'yicha.
 *
 * Metadata'ni LiveKit **tokendan** oladi, token esa backend tomonidan imzolangan
 * (`livekit/token.go` → `RoleMetadata`). Klient uni o'zgartira olmaydi, shuning
 * uchun "kim ustoz" savolining yagona ishonchli javobi shu. `RoomAdmin` grant'i
 * bu ish uchun yaramaydi — u boshqa klientlarga umuman ko'rinmaydi.
 */
export function isHostParticipant(p) {
  if (!p || !p.metadata) return false
  try {
    return JSON.parse(p.metadata)?.role === 'host'
  } catch {
    return false
  }
}

/**
 * Data-channel xabarini ishonch modeli bo'yicha filtrlaydi. Qaytadi: qabul
 * qilingan xabar (kerak bo'lsa tuzatilgan) yoki `null`.
 *
 * ## Nega kerak
 * `CanPublishData` XONADAGI HAMMADA yoqilgan (busiz doska va so'rovnoma ishlamaydi).
 * Ya'ni har qanday mehmon brauzer konsolidan istalgan payload yubora oladi.
 * Payload ichidagi `name` / `sender_identity` — shunchaki mehmon yozgan matn;
 * unga ishonish "Ustoz Ali: imtihon bekor, mana havola…" degan soxta xabarni
 * butun sinfga ko'rsatish demakdir (xabar serverda saqlanmagani uchun izi ham
 * qolmaydi). Xuddi shu yo'l bilan soxta so'rovnoma ochish va doskani buzish mumkin.
 *
 * ## Qoida
 * LiveKit har paketga uni KIM yuborganini o'zi biriktiradi (`participant`) — bu
 * maydonni klient soxtalashtira olmaydi:
 *
 *   participant yo'q  → serverdan (backend `SendData`) → ishonchli.
 *   participant bor   → klientdan → faqat HOST, faqat `wb` / `poll`.
 *
 * Klientdan kelgan xabarda ism/identity payload'dan EMAS, `participant`dan
 * olinadi — soxta muallif nomi shu yerda uziladi.
 */
export function acceptData(msg, participant) {
  if (!msg) return null

  // Serverdan: `SendData` da yuboruvchi ishtirokchi yo'q.
  if (!participant) return SERVER_KINDS.has(msg.kind) ? msg : null

  // Klientdan: faqat host va faqat unga ruxsat etilgan turlar.
  if (!isHostParticipant(participant)) return null
  if (!HOST_KINDS.has(msg.kind)) return null

  // Muallifni LiveKit tasdiqlagan manbadan qayta yozamiz.
  return {
    ...msg,
    name: participant.name || msg.name,
    identity: participant.identity,
    senderIdentity: participant.identity,
  }
}
