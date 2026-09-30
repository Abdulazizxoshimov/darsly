// Xona tokenining (LiveKit participant JWT) HAYOT SIKLI — yangilash.
//
// Backend mehmon tokenini 30 daqiqaga beradi (`livekit/token.go`:
// `participantTokenTTLMax`) va o'zi uzaytirmaydi. Token esa xona ichidagi HAR
// BIR server so'roviga (chat tarixi/yuborish/fayl, qo'l, reaksiya, ovoz, holat)
// va LiveKit'ga har to'liq qayta ulanishga kerak. Avval u bir marta olinib
// o'zgarmas edi: 30 daqiqadan keyin hamma so'rov 401 berardi (matni esa
// «Email yoki parol noto'g'ri»), holat so'rovi jimgina yiqilardi, kichik uzilish
// esa cheksiz qayta ulanish sikliga aylanardi.
//
// Endi token IKKI yo'l bilan yangilanadi:
//   · proaktiv — muddati tugashidan `RENEW_LEAD_MS` oldin (`renewDelay`);
//   · talab bo'yicha — xona so'rovi 401 bersa BIR MARTA (`source.run`).
// Yangi token `POST /joinlink/:slug` (mehmon) yoki `POST /lessons/:id/token`
// (ustoz) bilan olinadi; mehmon uchun kirish ma'lumotlari sessiyada saqlanadi.
//
// ⚠️ Backend cheklovlari (klient tomonda hal qilib bo'lmaydi):
//   1. Qayta `join` mehmonga YANGI identity beradi — shuning uchun mehmon
//      LiveKit'ga yangi token bilan qayta ulanadi (identity mos bo'lsin).
//   2. Kutish xonasi YOQILGAN darsda qayta `join` yangi kutish so'rovi
//      yaratadi (`joinlink.Join`) — jimgina yangilab bo'lmaydi; mehmon kutish
//      sahifasiga qaytariladi (`RoomTokenError` kind=`waiting_room`).
// To'liq yechim — identity'ni saqlab tokenni uzaytiradigan backend endpoint'i.

import { ApiError, errorText } from '../api/api'
import { joinLink } from '../api/join'
import { getHostToken } from '../api/lessons'
import { jwtExpiry } from './jwt'

/** Muddat tugashidan qancha oldin yangilanadi. */
export const RENEW_LEAD_MS = 3 * 60_000
/** Proaktiv yangilash tarmoq sababli yiqilsa — qayta urinish oralig'i. */
export const RENEW_RETRY_MS = 60_000

/**
 * @typedef {object} RoomToken
 * @property {string} token      LiveKit access JWT
 * @property {string} ws_url     signaling manzili
 * @property {string} [identity] ishtirokchi identifikatori
 * @property {string} [role]     'host' | 'participant'
 * @property {string} [lesson_id]
 */

/**
 * @typedef {object} GuestJoin  Mehmonning qayta kirish ma'lumotlari.
 * @property {string} slug
 * @property {string} guestName
 * @property {string} [passcode]
 */

/**
 * Proaktiv yangilashgacha qolgan vaqt (ms, ≥0) yoki `null` (`exp` o'qilmasa —
 * rejalashtirmaymiz, talab bo'yicha yo'l baribir ishlaydi).
 */
export function renewDelay(jwt, now = Date.now(), lead = RENEW_LEAD_MS) {
  const exp = jwtExpiry(jwt)
  if (exp === null) return null
  return Math.max(0, exp - lead - now)
}

/** Xona so'rovi tokenni rad etdi (401) — yangilab qayta urinish mumkin. */
export function isRoomAuthError(err) {
  return err instanceof ApiError && err.status === 401
}

/**
 * Yangilash yakuniy yiqildi. `kind`:
 *   'waiting_room' — qayta tasdiq kerak (`requestId` bilan)
 *   'ended'        — dars yakunlangan/o'chirilgan
 *   'not_live'     — ustoz hali xonada emas
 *   'locked'       — ustoz darsni qulflagan
 *   'removed'      — mehmon bloklangan
 *   'forbidden'    — boshqa rad (masalan parol o'zgargan)
 *   'network'      — VAQTINCHALIK: keyinroq qayta urinish mumkin
 */
export class RoomTokenError extends Error {
  constructor(kind, message, extra = {}) {
    super(message)
    this.name = 'RoomTokenError'
    this.kind = kind
    Object.assign(this, extra)
  }
}

const ROOM_TOKEN_TEXT = {
  waiting_room: 'Sessiya muddati tugadi — ustoz tasdig‘i kutilmoqda',
  ended: 'Dars yakunlangan',
  not_live: 'Ustoz hali xonada emas',
  locked: 'Ustoz darsni qulfladi — yangi kirish yo‘q',
  removed: 'Sizni bu darsdan chiqarishgan',
  forbidden: 'Darsga kirish rad etildi',
  network: 'Serverga ulanib bo‘lmadi — qayta urinilmoqda',
}

/** Yangilash yakunan mumkin emasmi (qayta urinish ma'nosiz). */
export function isFatalRoomTokenError(err) {
  return err instanceof RoomTokenError && err.kind !== 'network'
}

/**
 * Xona ichidagi xatoni o'zbekcha matnga aylantiradi. Umumiy `errorText`
 * 401 ni «Email yoki parol noto'g'ri» deydi — xonada esa bu tokenning
 * muddati, parol emas.
 */
export function roomErrorText(err, fallback = 'Xatolik yuz berdi') {
  if (err instanceof RoomTokenError) return ROOM_TOKEN_TEXT[err.kind] || err.message || fallback
  if (isRoomAuthError(err)) return 'Xona sessiyasi tugadi — qayta ulanmoqda'
  return errorText(err, fallback)
}

/** `POST /joinlink` javobi → token; boshqa har qanday holat `RoomTokenError` tashlaydi. */
export function fromJoinResponse(resp) {
  const step = resp?.next_step
  if (step === 'join' && resp.room?.token) return resp.room
  if (step === 'waiting_room' && resp.request_id) {
    throw new RoomTokenError('waiting_room', 'waiting room approval required', {
      requestId: resp.request_id,
      lesson: resp.lesson || null,
    })
  }
  if (step === 'lesson_ended') throw new RoomTokenError('ended', 'lesson ended')
  if (step === 'waiting_for_host') throw new RoomTokenError('not_live', 'host not in room')
  throw new RoomTokenError('forbidden', 'unexpected join response')
}

/** `POST /joinlink` xatosi → `RoomTokenError`. */
export function fromJoinError(err) {
  if (err instanceof RoomTokenError) return err
  if (err instanceof ApiError) {
    if (err.status === 404) return new RoomTokenError('ended', err.message)
    if (err.status === 403) {
      if (/locked/i.test(err.message)) return new RoomTokenError('locked', err.message)
      if (/removed/i.test(err.message)) return new RoomTokenError('removed', err.message)
      return new RoomTokenError('forbidden', err.message)
    }
    if (err.status === 401) return new RoomTokenError('forbidden', err.message)
    if (err.status >= 500 || err.status === 0 || err.code === 'RATE_LIMITED') {
      return new RoomTokenError('network', err.message)
    }
    return new RoomTokenError('forbidden', err.message)
  }
  return new RoomTokenError('network', err?.message || 'network')
}

/** @param {GuestJoin} [join] kirish ma'lumotlari yo'q bo'lsa (eski sessiya) yangilab bo'lmaydi */
export function guestMinter(join) {
  return async () => {
    if (!join?.slug) throw new RoomTokenError('forbidden', 'no join credentials in session')
    let resp
    try {
      resp = await joinLink(join.slug, { guest_name: join.guestName, passcode: join.passcode || undefined })
    } catch (e) {
      throw fromJoinError(e)
    }
    return fromJoinResponse(resp)
  }
}

export function hostMinter(lessonId) {
  return async () => {
    try {
      return await getHostToken(lessonId)
    } catch (e) {
      throw fromJoinError(e)
    }
  }
}

/**
 * Joriy tokenning yagona manbai.
 *
 *   current()   — joriy `RoomToken`
 *   subscribe() — yangi token kelganda xabardor bo'lish (bekor qilish qaytadi)
 *   renew()     — qayta olish; bir vaqtdagi chaqiruvlar BITTA so'rovga birlashadi
 *   run(fn)     — `fn(token)` ni bajaradi; 401 bo'lsa yangilab BIR MARTA takrorlaydi
 *
 * @param {{ initial: RoomToken, mint: () => Promise<RoomToken>, persist?: (t: RoomToken) => void }} opts
 */
export function createRoomTokenSource({ initial, mint, persist }) {
  let current = initial
  let inflight = null
  const listeners = new Set()

  const source = {
    current: () => current,
    subscribe(fn) {
      listeners.add(fn)
      return () => listeners.delete(fn)
    },
    renew() {
      if (!inflight) {
        inflight = mint()
          .then((fresh) => {
            current = fresh
            if (persist) persist(fresh)
            for (const l of listeners) l(fresh)
            return fresh
          })
          .finally(() => {
            inflight = null
          })
      }
      return inflight
    },
    async run(fn) {
      try {
        return await fn(current)
      } catch (e) {
        if (!isRoomAuthError(e)) throw e
        const fresh = await source.renew()
        return fn(fresh)
      }
    },
  }
  return source
}
