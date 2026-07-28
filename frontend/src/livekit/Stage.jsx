import { memo } from 'react'
import { ParticipantTile } from './ParticipantTile'

// Gallery yoki speaker (ekran ulashilsa avtomatik speaker).
// `participants` — `useRoom` snapshot'i (local birinchi). `memo`: doskadagi chizish
// yoki chat holati o'zgarganda sahna qayta render bo'lmasin.
export const Stage = memo(function Stage({ participants, view, raisedHands }) {
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

  const cols = participants.length <= 1 ? 1 : participants.length <= 4 ? 2 : participants.length <= 9 ? 3 : 4
  return (
    <div className="gallery" style={{ gridTemplateColumns: `repeat(${cols}, 1fr)` }}>
      {participants.map((p) => (
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
  )
})
