// LiveKit data-channel xabar sxemasi (client yuboradigan).
// Backend host-chat'ni entity.ChatMessage shaklida (kind'siz) broadcast qiladi — uni ajratib olamiz.

const enc = new TextEncoder()
const dec = new TextDecoder()

export function encodeData(msg) {
  // new Uint8Array(...) — ArrayBuffer bilan (publishData tipi uchun)
  return new Uint8Array(enc.encode(JSON.stringify(msg)))
}

/**
 * @typedef {object} ChatEntry  Chat paneli tushunadigan yozuv.
 * @property {string} id
 * @property {string} name
 * @property {string} body
 * @property {string} senderIdentity
 * @property {string|null} toIdentity
 * @property {{name:string,size:number,mime:string,url:string}|null} file
 * @property {number} ts
 */

/**
 * Backend `ChatMessage` (REST javobi ham, data-channel ham bir xil shakl) →
 * `ChatEntry`. Bitta joyda: avval bu xaritalash to'rt joyda qo'lda takrorlanardi.
 *
 * `sender_identity` va `to_identity` ham olib o'tiladi: klient shular asosida
 * "bu meniki" va "bu shaxsiy" degan qarorni chiqaradi. Fayl (`file`) presigned
 * havola bilan keladi — u ham o'tadi.
 *
 * @returns {ChatEntry|null} shakl noto'g'ri bo'lsa `null`
 */
export function chatEntryFromServer(m) {
  if (!m || typeof m !== 'object') return null
  if (typeof m.id !== 'string' || typeof m.sender_name !== 'string' || typeof m.body !== 'string') return null
  return {
    id: m.id,
    name: m.sender_name,
    body: m.body,
    senderIdentity: typeof m.sender_identity === 'string' ? m.sender_identity : '',
    toIdentity: typeof m.to_identity === 'string' && m.to_identity ? m.to_identity : null,
    file: m.file && typeof m.file === 'object' ? m.file : null,
    ts: Date.parse(m.created_at) || Date.now(),
  }
}

const isStr = (v) => typeof v === 'string'
const isObj = (v) => !!v && typeof v === 'object' && !Array.isArray(v)

// Ochiq so'rovnoma payload'i: savol va variantlar SHART. Buzuq `poll` xabari
// (masalan eski host klienti yoki qasddan yuborilgan) o'quvchi panelini
// `options.map` da yiqitardi — hamma o'quvchida bir vaqtda.
function validPoll(p) {
  return isObj(p) && isStr(p.id) && isStr(p.question) && Array.isArray(p.options) && p.options.every(isStr)
}

// Har `kind` uchun minimal shakl — UI shu maydonlarga tekshiruvsiz tayanadi.
const SHAPE = {
  chat: (o) => isStr(o.id) && isStr(o.name) && isStr(o.body),
  chat_deleted: (o) => isStr(o.id),
  reaction: (o) => isStr(o.emoji),
  hand: (o) => o.act === 'lower_all' || isStr(o.identity),
  poll: (o) =>
    (o.action === 'open' && validPoll(o.poll)) ||
    (o.action === 'close' && isObj(o.poll) && isStr(o.poll.id)),
  poll_published: (o) => isObj(o.results) && (o.results.poll === undefined || validPoll(o.results.poll)),
  wb: (o) => isStr(o.act),
  policy: () => true,
}

/**
 * Kelgan data'ni normalizatsiya qiladi va SHAKLINI tekshiradi.
 * Qaytadi: {kind:'chat'|'chat_deleted'|'reaction'|'hand'|'poll'|'poll_published'|'wb'|'policy', ...}
 * yoki null (noma'lum tur / buzuq shakl / JSON emas).
 */
export function decodeData(payload) {
  try {
    const obj = JSON.parse(dec.decode(payload))
    if (!isObj(obj)) return null
    const check = SHAPE[obj.kind]
    if (check) return check(obj) ? obj : null
    // Backend ChatMessage (kind yo'q, sender_name + body bor).
    if (obj.kind === undefined) {
      const entry = chatEntryFromServer(obj)
      if (entry) return { kind: 'chat', ...entry }
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
// `policy` (ovoz siyosati) endi SERVERDAN keladi: avval uni faqat WEB ustoz
// klienti yuborardi, ya'ni telefondan o'tilgan darsda o'quvchi hech qanday
// siyosat xabarini olmasdi va mikrofon tugmasi yolg'on ko'rsatardi (yoqadi,
// server esa jimgina qayta mute qiladi). Server `mute-all` dan keyin xonaga
// o'zi tarqatadi — bu ham to'g'ri, ham soxtalashtirib bo'lmaydigan manba.
const SERVER_KINDS = new Set([
  'chat',
  'chat_deleted',
  'reaction',
  'hand',
  'poll',
  'poll_published',
  'policy',
])
const HOST_KINDS = new Set(['wb', 'poll'])

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
