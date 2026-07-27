import { Track } from 'livekit-client'
import { roomParticipants } from './useRoom'
import { ParticipantTile } from './ParticipantTile'

// Gallery yoki speaker (ekran ulashilsa avtomatik speaker).
export function Stage({ room, view, raisedHands }) {
  const participants = roomParticipants(room)
  const localId = room.localParticipant.identity

  // Ekran ulashuvchi (agar bor bo'lsa)
  const screenSharer = participants.find((p) => {
    const pub = p.getTrackPublication(Track.Source.ScreenShare)
    return pub && pub.videoTrack && !pub.isMuted
  })

  const isSpeaker = view === 'speaker' || !!screenSharer

  if (isSpeaker) {
    const feature = screenSharer || participants[0]
    const featureSource = screenSharer ? Track.Source.ScreenShare : Track.Source.Camera
    return (
      <div className="speaker">
        <div className="speaker__feature">
          <ParticipantTile participant={feature} source={featureSource} isLocal={feature.identity === localId} large />
        </div>
        {participants.length > 1 && (
          <div className="filmstrip">
            {participants
              .filter((p) => p.identity !== feature.identity)
              .map((p) => (
                <div className="filmstrip__tile" key={p.identity}>
                  <ParticipantTile
                    participant={p}
                    handRaised={raisedHands.has(p.identity)}
                    isLocal={p.identity === localId}
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
          participant={p}
          handRaised={raisedHands.has(p.identity)}
          isLocal={p.identity === localId}
        />
      ))}
    </div>
  )
}
