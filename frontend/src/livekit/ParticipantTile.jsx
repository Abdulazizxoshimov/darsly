import { memo, useEffect, useRef } from 'react'
import { Hand, MicOff } from 'lucide-react'
import { Avatar } from '../components/Avatar'

// Bitta ishtirokchi plitkasi.
//
// `useRoom` snapshot'idan keladigan SODDA proplar bilan ishlaydi (SDK obyekti emas) —
// shu sabab `memo` haqiqatan foyda beradi: qo'shni plitkada nimadir o'zgarsa bu plitka
// qayta render bo'lmaydi. Video treki esa havola sifatida uzatiladi, chunki `attach()`
// aynan o'sha obyektni talab qiladi.
export const ParticipantTile = memo(function ParticipantTile({
  name,
  track,
  micMuted,
  speaking,
  isLocal,
  handRaised,
  large,
  screen,
}) {
  const videoRef = useRef(null)

  useEffect(() => {
    const el = videoRef.current
    if (track && el) {
      track.attach(el)
      return () => {
        track.detach(el)
      }
    }
  }, [track])

  return (
    <div className={`tile ${large ? 'tile--large' : ''} ${speaking ? 'tile--speaking' : ''}`}>
      {track ? (
        // Ekran ulashish: `contain` — kadr TO'LIQ ko'rinsin. `cover` bo'lsa portret telefon
        // ekrani landscape oynada qirqiladi (qurilma sinovida o'quvchi ustoz ekranining
        // atigi ~25% ini ko'rgan, qolgani kesilgan + upscale'dan yumshoq bo'lgan).
        // Kamera uchun `cover` to'g'ri — u yerda qirqim tabiiy va bo'sh chet chiqmasin.
        <video
          ref={videoRef}
          className={`tile__video ${screen ? 'tile__video--contain' : ''}`}
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
        {!screen && micMuted && <MicOff size={14} color="var(--danger)" />}
        <span className="truncate">
          {name}
          {isLocal && ' (siz)'}
        </span>
      </div>
    </div>
  )
})
