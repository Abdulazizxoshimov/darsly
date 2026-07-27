import { useEffect, useRef } from 'react'
import { Track } from 'livekit-client'
import { Hand, MicOff } from 'lucide-react'
import { Avatar } from '../components/Avatar'

// Ishtirokchining videosini (kamera yoki ekran) <video>ga bog'laydi, yoki avatar ko'rsatadi.
export function ParticipantTile({ participant, source = Track.Source.Camera, handRaised, isLocal, large }) {
  const videoRef = useRef(null)
  const pub = participant.getTrackPublication(source)
  const videoTrack = pub && pub.videoTrack && !pub.isMuted ? pub.videoTrack : null

  useEffect(() => {
    const el = videoRef.current
    if (videoTrack && el) {
      videoTrack.attach(el)
      return () => {
        videoTrack.detach(el)
      }
    }
  }, [videoTrack])

  const micPub = participant.getTrackPublication(Track.Source.Microphone)
  const muted = !micPub || micPub.isMuted
  const name = participant.name || participant.identity

  return (
    <div className={`tile ${large ? 'tile--large' : ''} ${participant.isSpeaking ? 'tile--speaking' : ''}`}>
      {videoTrack ? (
        // Ekran ulashish: `contain` — kadr TO'LIQ ko'rinsin. `cover` bo'lsa portret telefon
        // ekrani landscape oynada qirqiladi (qurilma sinovida o'quvchi ustoz ekranining
        // atigi ~25% ini ko'rgan, qolgani kesilgan + upscale'dan yumshoq bo'lgan).
        // Kamera uchun `cover` to'g'ri — u yerda qirqim tabiiy va bo'sh chet chiqmasin.
        <video
          ref={videoRef}
          className={`tile__video ${source === Track.Source.ScreenShare ? 'tile__video--contain' : ''}`}
          autoPlay
          playsInline
          muted={isLocal}
        />
      ) : (
        <div className="tile__placeholder">
          <Avatar name={name} size={large ? 96 : 72} />
        </div>
      )}
      {handRaised && (
        <div className="tile__hand">
          <Hand size={16} />
        </div>
      )}
      <div className="tile__name">
        {source === Track.Source.Camera && muted && <MicOff size={14} color="var(--danger)" />}
        <span className="truncate">
          {name}
          {isLocal && ' (siz)'}
        </span>
      </div>
    </div>
  )
}
