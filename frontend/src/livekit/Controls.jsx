import { useState } from 'react'
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
} from 'lucide-react'
import { toast } from '../lib/toast'

const REACTIONS = ['👍', '👏', '❤️', '😂', '😮', '🎉', '✋']

export function Controls({
  room,
  isHost,
  panel,
  setPanel,
  handRaised,
  onToggleHand,
  onReaction,
  recording,
  onToggleRecord,
  onLeave,
  unreadChat,
  waitingCount,
  raisedHandsCount = 0,
  whiteboardOn,
  onToggleBoard,
}) {
  const [reactOpen, setReactOpen] = useState(false)
  const lp = room.localParticipant
  const canPublish = lp.permissions?.canPublish ?? isHost
  const micOn = lp.isMicrophoneEnabled
  const camOn = lp.isCameraEnabled
  const screenOn = lp.isScreenShareEnabled

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
      await lp.setScreenShareEnabled(!screenOn)
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

      {isHost && (
        <Ctrl label={recording ? 'Yozilmoqda' : 'Yozib olish'} onClick={onToggleRecord} on={recording} danger={recording}>
          <Radio size={20} />
        </Ctrl>
      )}

      <button className="leave-btn" onClick={onLeave}>
        <PhoneOff size={20} /> {isHost ? 'Yakunlash' : 'Chiqish'}
      </button>
    </div>
  )
}

function Ctrl({ children, label, onClick, active, on, danger, disabled, badge }) {
  const cls = ['ctrl', on ? 'ctrl--on' : danger ? 'ctrl--danger' : active ? 'ctrl--active' : '']
    .filter(Boolean)
    .join(' ')
  return (
    <button className={cls} onClick={onClick} disabled={disabled} title={label}>
      {children}
      {badge !== undefined && <span className="ctrl__badge">{badge}</span>}
    </button>
  )
}
