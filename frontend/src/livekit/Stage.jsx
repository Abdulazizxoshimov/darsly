import { memo, useState } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { ParticipantTile } from './ParticipantTile'
import { galleryOrder, galleryPage } from './roomLogic'

// Gallery yoki speaker (ekran ulashilsa avtomatik speaker).
// `participants` — `useRoom` snapshot'i (local birinchi). `memo`: doskadagi chizish
// yoki chat holati o'zgarganda sahna qayta render bo'lmasin.
//
// Galereya SAHIFALANADI (bir sahifada maks 9 — Zoom andozasi): ishtirokchi ko'p
// bo'lsa hamma plitka bir ekranga tiqilmaydi. Faqat joriy sahifa plitkalari
// render qilinadi — ko'rinmagan plitkaning <video>si DOM'da yo'q va LiveKit
// adaptiveStream u kamera trekiga obuna bo'lmaydi (trafik tejaladi).
// Tartib (`galleryOrder`): ustoz → o'zim → gapirayotganlar → qolganlar, ya'ni
// ustoz va gapirayotganlar doim birinchi sahifada.
export const Stage = memo(function Stage({ participants, view, raisedHands }) {
  const [pageReq, setPageReq] = useState(0)
  const screenSharer = participants.find((p) => p.screenTrack)
  const isSpeaker = view === 'speaker' || !!screenSharer

  if (isSpeaker) {
    const feature = screenSharer || participants[0]
    if (!feature) return null
    const showingScreen = !!screenSharer
    return (
      <div className="speaker">
        <div className="speaker__feature">
          <ParticipantTile
            name={feature.name}
            track={showingScreen ? feature.screenTrack : feature.camTrack}
            micMuted={feature.micMuted}
            speaking={feature.speaking}
            isLocal={feature.isLocal}
            screen={showingScreen}
            large
          />
        </div>
        {participants.length > 1 && (
          <div className="filmstrip">
            {participants
              .filter((p) => p.identity !== feature.identity)
              .map((p) => (
                <div className="filmstrip__tile" key={p.identity}>
                  <ParticipantTile
                    name={p.name}
                    track={p.camTrack}
                    micMuted={p.micMuted}
                    speaking={p.speaking}
                    isLocal={p.isLocal}
                    handRaised={raisedHands.has(p.identity)}
                  />
                </div>
              ))}
          </div>
        )}
      </div>
    )
  }

  // `galleryPage` so'ralgan sahifani chegaraga qisadi — ishtirokchilar soni
  // kamayganda state'ni alohida "tuzatish" shart emas.
  const { items, page, total } = galleryPage(galleryOrder(participants), pageReq)

  const cols = items.length <= 1 ? 1 : items.length <= 4 ? 2 : 3
  return (
    <div className="gallery-wrap">
      <div className="gallery" style={{ gridTemplateColumns: `repeat(${cols}, 1fr)` }}>
        {items.map((p) => (
          <ParticipantTile
            key={p.identity}
            name={p.name}
            track={p.camTrack}
            micMuted={p.micMuted}
            speaking={p.speaking}
            isLocal={p.isLocal}
            handRaised={raisedHands.has(p.identity)}
          />
        ))}
      </div>
      {total > 1 && (
        <div className="gallery-pager">
          <button
            className="icon-btn"
            onClick={() => setPageReq(page - 1)}
            disabled={page === 0}
            aria-label="Oldingi sahifa"
          >
            <ChevronLeft size={18} />
          </button>
          <span className="gallery-pager__label">
            {page + 1} / {total} sahifa
          </span>
          <button
            className="icon-btn"
            onClick={() => setPageReq(page + 1)}
            disabled={page >= total - 1}
            aria-label="Keyingi sahifa"
          >
            <ChevronRight size={18} />
          </button>
        </div>
      )}
    </div>
  )
})
