// LiveKit data-channel xabar sxemasi (client yuboradigan).
// Backend host-chat'ni entity.ChatMessage shaklida (kind'siz) broadcast qiladi — uni ajratib olamiz.

const enc = new TextEncoder()
const dec = new TextDecoder()

export function encodeData(msg) {
  // new Uint8Array(...) — ArrayBuffer bilan (publishData tipi uchun)
  return new Uint8Array(enc.encode(JSON.stringify(msg)))
}

// Kelgan data'ni normalizatsiya qiladi. Qaytadi: {kind:'chat'|'reaction'|'hand'|'poll', ...} yoki null.
export function decodeData(payload) {
  try {
    const obj = JSON.parse(dec.decode(payload))
    if (obj && typeof obj === 'object') {
      if (
        obj.kind === 'chat' ||
        obj.kind === 'reaction' ||
        obj.kind === 'hand' ||
        obj.kind === 'poll' ||
        obj.kind === 'wb'
      ) {
        return obj
      }
      // Backend ChatMessage (kind yo'q, sender_name + body bor)
      if (typeof obj.sender_name === 'string' && typeof obj.body === 'string') {
        return {
          kind: 'chat',
          id: obj.id,
          name: obj.sender_name,
          body: obj.body,
          ts: Date.parse(obj.created_at) || Date.now(),
        }
      }
    }
  } catch {
    /* e'tiborsiz */
  }
  return null
}
