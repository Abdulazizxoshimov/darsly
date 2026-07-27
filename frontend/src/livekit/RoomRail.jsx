import { Hand, Volume2 } from 'lucide-react'
import { roomParticipants } from './useRoom'

// Xona o'ng chekkasidagi axborot ustuni:
//  - "Qo'l ko'targanlar" (faqat HOST ko'radi) — video ham, doska rejimida ham ko'rinadi (#3).
//  - "Gapiryapti" — hozir kim(lar) gapirayotganini hammaga ko'rsatadi (#4).
// room.activeSpeakers / participant.isSpeaking'dan foydalanadi; re-render useRoom bump orqali
// (ActiveSpeakersChanged) keladi. Hech kim gapirmasa/qo'l ko'tarmasa — hech narsa ko'rsatilmaydi.
export function RoomRail({ room, isHost, raisedHands }) {
  const localId = room.localParticipant.identity
  const speakers = (room.activeSpeakers || []).filter((p) => p.isSpeaking)

  // Host uchun qo'l ko'targanlar (o'zidan tashqari, faqat hali xonada bo'lganlar).
  let raisedNames = []
  if (isHost && raisedHands && raisedHands.size) {
    const byId = new Map(roomParticipants(room).map((p) => [p.identity, p]))
    raisedNames = [...raisedHands]
      .filter((id) => id !== localId && byId.has(id))
      .map((id) => byId.get(id).name || byId.get(id).identity)
  }

  if (!speakers.length && !raisedNames.length) return null

  return (
    <div className="room-rail">
      {raisedNames.length > 0 && (
        <div className="rail-card rail-card--hand">
          <div className="rail-card__title">
            <Hand size={13} /> Qo'l ko'targanlar
          </div>
          {raisedNames.map((name, i) => (
            <div key={i} className="rail-row">
              <Hand size={12} />
              <span className="truncate">{name}</span>
            </div>
          ))}
        </div>
      )}

      {speakers.length > 0 && (
        <div className="rail-card rail-card--speak">
          <div className="rail-card__title">
            <Volume2 size={13} /> Gapiryapti
          </div>
          {speakers.map((p) => (
            <div key={p.identity} className="rail-row">
              <span className="speak-eq" aria-hidden="true">
                <i />
                <i />
                <i />
              </span>
              <span className="truncate">
                {p.name || p.identity}
                {p.identity === localId ? ' (siz)' : ''}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
