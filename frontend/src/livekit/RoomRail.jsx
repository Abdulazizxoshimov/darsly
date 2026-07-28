import { memo } from 'react'
import { Hand, Volume2 } from 'lucide-react'

// Xona o'ng chekkasidagi axborot ustuni:
//  - "Qo'l ko'targanlar" (faqat HOST ko'radi) — video ham, doska rejimida ham ko'rinadi.
//    Tartib — KO'TARILGAN VAQT bo'yicha (Map insert tartibi): kim birinchi so'rasa,
//    ustoz uni birinchi ko'radi. Navbat tartibi bo'lmasa "kim birinchi edi" savoli
//    har darsda takrorlanardi.
//  - "Gapiryapti" — hozir kim(lar) gapirayotganini hammaga ko'rsatadi.
export const RoomRail = memo(function RoomRail({ participants, isHost, raisedHands, localId }) {
  const speakers = participants.filter((p) => p.speaking)

  // Faqat hali xonada bo'lgan va o'zimiz bo'lmagan qo'llar.
  let raised = []
  if (isHost && raisedHands.size) {
    const present = new Set(participants.map((p) => p.identity))
    raised = [...raisedHands.entries()]
      .filter(([id]) => id !== localId && present.has(id))
      .map(([id, h]) => ({ id, name: h.name }))
  }

  if (!speakers.length && !raised.length) return null

  return (
    <div className="room-rail">
      {raised.length > 0 && (
        <div className="rail-card rail-card--hand">
          <div className="rail-card__title">
            <Hand size={13} /> Qo'l ko'targanlar
          </div>
          {raised.map((r, i) => (
            <div key={r.id} className="rail-row">
              <span className="rail-num">{i + 1}</span>
              <span className="truncate">{r.name}</span>
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
                {p.name}
                {p.isLocal ? ' (siz)' : ''}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
})
