import { memo, useState } from 'react'
import {
  BarChart3,
  Hand,
  MessageSquare,
  Mic,
  MicOff,
  MonitorUp,
  PhoneOff,
  Presentation,
  Radio,
  Smile,
  Users,
  Video,
  VideoOff,
  Zap,
} from 'lucide-react'
import { toast } from '../lib/toast'
import { SCREEN_CAPTURE } from './mediaTuning'

const REACTIONS = ['👍', '👏', '❤️', '😂', '😮', '🎉', '✋']

// Boshqaruv paneli. `local` — `useRoom` snapshot'i (SDK obyektini render paytida
// o'qimaymiz), `room` esa faqat amal bajarish uchun kerak — uning identiteti barqaror,
// shuning uchun `memo` ishlaydi: sahnadagi o'zgarish panelni qayta render qilmaydi.
export const Controls = memo(function Controls({
  room,
  local,
  isHost,
  recording,
  onToggleRecord,
  panel,
  setPanel,
  handRaised,
  onToggleHand,
  onReaction,
  onLeave,
  unreadChat,
  waitingCount,
  raisedHandsCount = 0,
  whiteboardOn,
  onToggleBoard,
  dataSaver,
  onToggleDataSaver,
}) {
  const [reactOpen, setReactOpen] = useState(false)
  const lp = room.localParticipant
  const canPublish = local?.canPublish ?? isHost
  const micOn = !!local?.micOn
  const camOn = !!local?.camOn
  const screenOn = !!local?.screenOn

  async function toggleMic() {
    if (!canPublish) return toast.info('Ustoz gapirishga ruxsat bermagan')
    try {
      await lp.setMicrophoneEnabled(!micOn)
    } catch {
      toast.error('Mikrofonni yoqib bo‘lmadi')
    }
  }
  async function toggleCam() {
    if (!canPublish) return toast.info('Ustoz kamera uchun ruxsat bermagan')
    try {
      await lp.setCameraEnabled(!camOn)
    } catch {
      toast.error('Kamerani yoqib bo‘lmadi')
    }
  }
  async function toggleScreen() {
    if (!canPublish) return toast.info('Ustoz ekran ulashishga ruxsat bermagan')
    // Android/iOS brauzerlari getDisplayMedia'ni bermaydi YOKI bersa ham rad etadi (NotAllowedError) —
    // ekran ulashish telefon/planshetda umuman ishlamaydi (platforma cheklovi). Doskadan foydalaning.
    const isMobile = /Android|iPhone|iPad|iPod/i.test(navigator.userAgent)
    if (!screenOn && (isMobile || !navigator.mediaDevices?.getDisplayMedia)) {
      return toast.error("Ekran ulashish telefon/planshetda ishlamaydi — kompyuterda oching yoki 'Doska'dan foydalaning")
    }
    try {
      // SCREEN_CAPTURE: ovoz bilan + `contentHint: 'text'` (matn keskinligi uchun).
      // Sifat/qatlam sozlamalari `PUBLISH_DEFAULTS` da — ikkalasi `mediaTuning.js` da.
      await lp.setScreenShareEnabled(!screenOn, screenOn ? undefined : SCREEN_CAPTURE)
    } catch (e) {
      // Foydalanuvchi tanlash oynasini bekor qilsa — bu xato emas, jim o'tamiz.
      if (e?.name === 'NotAllowedError' || /permission|denied|cancel/i.test(e?.message || '')) return
      toast.error('Ekranni ulashib bo‘lmadi')
    }
  }

  function togglePanel(p) {
    setPanel(panel === p ? 'none' : p)
  }

  return (
    <div className="controls">
      <Ctrl label="Mikrofon" onClick={toggleMic} disabled={!canPublish} danger={!micOn}>
        {micOn ? <Mic size={20} /> : <MicOff size={20} />}
      </Ctrl>
      <Ctrl label="Kamera" onClick={toggleCam} disabled={!canPublish} danger={!camOn}>
        {camOn ? <Video size={20} /> : <VideoOff size={20} />}
      </Ctrl>
      <Ctrl label="Ekran" onClick={toggleScreen} on={screenOn}>
        <MonitorUp size={20} />
      </Ctrl>

      <div style={{ position: 'relative' }}>
        <Ctrl label="Reaksiya" onClick={() => setReactOpen((v) => !v)} active={reactOpen}>
          <Smile size={20} />
        </Ctrl>
        {reactOpen && (
          <div className="reactions-pop" onMouseLeave={() => setReactOpen(false)}>
            {REACTIONS.map((e) => (
              <button
                key={e}
                onClick={() => {
                  onReaction(e)
                  setReactOpen(false)
                }}
              >
                {e}
              </button>
            ))}
          </div>
        )}
      </div>

      <Ctrl label="Qo'l" onClick={onToggleHand} on={handRaised}>
        <Hand size={20} />
      </Ctrl>

      {/*
        Tejamkor rejim — zaif internet uchun. Kirish KAMERA oqimlari uziladi;
        ekran ulashish va ovoz qoladi, ya'ni darsning mazmuni saqlanadi.
        Ustozga ham kerak: 30 ta o'quvchi kamerasi uning yuklama kanalini yeydi.
      */}
      <Ctrl
        label={dataSaver ? 'Tejamkor rejim yoqilgan' : 'Tejamkor rejim'}
        onClick={onToggleDataSaver}
        on={dataSaver}
      >
        <Zap size={20} />
      </Ctrl>

      <div className="ctrl-sep" />

      <Ctrl
        label="Ishtirokchilar"
        onClick={() => togglePanel('participants')}
        active={panel === 'participants'}
        badge={isHost && waitingCount + raisedHandsCount > 0 ? waitingCount + raisedHandsCount : undefined}
      >
        <Users size={20} />
      </Ctrl>
      <Ctrl label="Chat" onClick={() => togglePanel('chat')} active={panel === 'chat'} badge={unreadChat > 0 ? unreadChat : undefined}>
        <MessageSquare size={20} />
      </Ctrl>
      <Ctrl label="So'rovnoma" onClick={() => togglePanel('polls')} active={panel === 'polls'}>
        <BarChart3 size={20} />
      </Ctrl>

      {isHost && (
        <Ctrl label={whiteboardOn ? 'Doskani yopish' : 'Doska'} onClick={onToggleBoard} on={whiteboardOn}>
          <Presentation size={20} />
        </Ctrl>
      )}

      {/*
        Yozib olish odatda SERVER tomonidan avtomatik boshlanadi (default yoniq,
        `track_published` webhook'i → `recording.EnsureForRoom`). Bu tugma chekka
        holatlar uchun: dars o'rtasida to'xtatish yoki avtomatik boshlash ishlamagan bo'lsa.
      */}
      {isHost && (
        <Ctrl label={recording ? 'Yozilmoqda — to‘xtatish' : 'Yozib olish'} onClick={onToggleRecord} on={recording} danger={recording}>
          <Radio size={20} />
        </Ctrl>
      )}

      <button className="leave-btn" onClick={onLeave}>
        <PhoneOff size={20} /> {isHost ? 'Yakunlash' : 'Chiqish'}
      </button>
    </div>
  )
})

function Ctrl({ children, label, onClick, active, on, danger, disabled, badge }) {
  const cls = ['ctrl', on ? 'ctrl--on' : danger ? 'ctrl--danger' : active ? 'ctrl--active' : '']
    .filter(Boolean)
    .join(' ')
  return (
    <button className={cls} onClick={onClick} disabled={disabled} title={label} aria-label={label}>
      {children}
      {badge !== undefined && <span className="ctrl__badge">{badge}</span>}
    </button>
  )
}
