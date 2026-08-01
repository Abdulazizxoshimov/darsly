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
import { CAMERA_PUBLISH, SCREEN_CAPTURE, SCREEN_PUBLISH } from './mediaTuning'
import { Modal } from '../components/Modal'
import { Button } from '../components/Button'
import { isTipMuted, muteTip, SHARE_WINDOW_TIP } from '../lib/uiPrefs'

const REACTIONS = ['👍', '👏', '❤️', '😂', '😮', '🎉', '✋']

// Boshqaruv paneli. `local` — `useRoom` snapshot'i (SDK obyektini render paytida
// o'qimaymiz), `room` esa faqat amal bajarish uchun kerak — uning identiteti barqaror,
// shuning uchun `memo` ishlaydi: sahnadagi o'zgarish panelni qayta render qilmaydi.
// getUserMedia xatosini foydalanuvchi TUZATA OLADIGAN o'zbekcha matnga aylantiradi.
// Umumiy "yoqib bo'lmadi" yetarli emas: ruxsat bermaslik, band qurilma va
// qurilma yo'qligi — uchtasi uch xil harakat talab qiladi.
function mediaFailText(what, e) {
  const name = e?.name || ''
  if (name === 'NotAllowedError' || name === 'PermissionDeniedError')
    return `${what}ga ruxsat berilmagan — brauzer manzil satridagi qulf belgisidan ruxsat bering`
  if (name === 'NotFoundError' || name === 'DevicesNotFoundError')
    return `${what} topilmadi — qurilma ulanganini tekshiring`
  if (name === 'NotReadableError' || name === 'TrackStartError')
    return `${what} band — uni ishlatayotgan boshqa dasturni yoping`
  return `${what}ni yoqib bo‘lmadi`
}

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
  // Dars siyosati: allow_self_unmute=false → o'quvchi o'zini OCHOLMAYDI
  // (server enforce qiladi; bu UI'da tugmani o'chirib sababini aytadi).
  selfUnmuteBlocked = false,
}) {
  const [reactOpen, setReactOpen] = useState(false)
  // «Butun ekranni emas, bitta oynani ulashing» eslatmasi (asoschi talabi).
  const [tipOpen, setTipOpen] = useState(false)
  const [tipMute, setTipMute] = useState(false)
  const lp = room.localParticipant
  const canPublish = local?.canPublish ?? isHost
  const micOn = !!local?.micOn
  const camOn = !!local?.camOn
  const screenOn = !!local?.screenOn

  // Insecure origin'da (http + IP, localhost emas) brauzer mediaDevices'ni
  // UMUMAN bermaydi. O'quvchi xonaga KIRADI (ko'rish/eshitish ishlaydi), faqat
  // publish tugmalari o'chiq turadi — sababi tooltip'da.
  const mediaOk =
    typeof navigator !== 'undefined' &&
    !!navigator.mediaDevices &&
    typeof navigator.mediaDevices.getUserMedia === 'function'

  // Mikrofonni OCHISH taqiqlangan holat; o'chirish (mute) har doim mumkin.
  const micLocked = !micOn && selfUnmuteBlocked && !isHost

  const micTitle = !mediaOk
    ? 'HTTPS kerak — mikrofon faqat xavfsiz (https) ulanishda ishlaydi'
    : micLocked
      ? 'Mentor ruxsat bermagan — mikrofonni ustoz ochadi'
      : !canPublish
        ? 'Ustoz gapirishga ruxsat bermagan'
        : 'Mikrofon'
  const camTitle = !mediaOk
    ? 'HTTPS kerak — kamera faqat xavfsiz (https) ulanishda ishlaydi'
    : !canPublish
      ? 'Ustoz kamera uchun ruxsat bermagan'
      : 'Kamera'

  async function toggleMic() {
    if (!canPublish) return toast.info('Ustoz gapirishga ruxsat bermagan')
    try {
      // Optimistik EMAS: tugma holati `local.micOn` (LiveKit muted hodisalari)
      // dan keladi. Server siyosat bo'yicha qayta mute qilsa, tugma o'zi
      // o'chiq holatga qaytadi.
      await lp.setMicrophoneEnabled(!micOn)
    } catch (e) {
      toast.error(mediaFailText('Mikrofon', e))
    }
  }
  async function toggleCam() {
    if (!canPublish) return toast.info('Ustoz kamera uchun ruxsat bermagan')
    try {
      // CAMERA_PUBLISH: kamerada `balanced` degradatsiya (mobil bilan bir xil).
      // Busiz xona darajasidagi `maintain-resolution` kameraga ham tushib,
      // zaif tarmoqda yuzni kichraytirish o'rniga MUZLATIB qo'yardi.
      await lp.setCameraEnabled(!camOn, undefined, CAMERA_PUBLISH)
    } catch (e) {
      toast.error(mediaFailText('Kamera', e))
    }
  }
  async function setScreen(on) {
    try {
      // SCREEN_CAPTURE: ovoz bilan + `contentHint: 'text'` (matn keskinligi uchun).
      // Sifat/qatlam sozlamalari `PUBLISH_DEFAULTS` da, degradatsiya siyosati esa
      // trek turiga bog'liq (`SCREEN_PUBLISH`) — hammasi `mediaTuning.js` da.
      await lp.setScreenShareEnabled(on, on ? SCREEN_CAPTURE : undefined, SCREEN_PUBLISH)
    } catch (e) {
      // Foydalanuvchi tanlash oynasini bekor qilsa — bu xato emas, jim o'tamiz.
      if (e?.name === 'NotAllowedError' || /permission|denied|cancel/i.test(e?.message || '')) return
      toast.error('Ekranni ulashib bo‘lmadi')
    }
  }

  function toggleScreen() {
    if (!canPublish) return toast.info('Ustoz ekran ulashishga ruxsat bermagan')
    if (screenOn) return setScreen(false)
    // Android/iOS brauzerlari getDisplayMedia'ni bermaydi YOKI bersa ham rad etadi (NotAllowedError) —
    // ekran ulashish telefon/planshetda umuman ishlamaydi (platforma cheklovi). Doskadan foydalaning.
    const isMobile = /Android|iPhone|iPad|iPod/i.test(navigator.userAgent)
    if (isMobile || !navigator.mediaDevices?.getDisplayMedia) {
      return toast.error("Ekran ulashish telefon/planshetda ishlamaydi — kompyuterda oching yoki 'Doska'dan foydalaning")
    }
    // ⭐ Brauzer tanlash oynasidan OLDIN eslatma.
    //
    // «Butun ekran» tanlansa suzuvchi chat oynasi ham EFIRGA tushadi: o'quvchilar
    // ustoz o'qiyotgan shaxsiy xabarlarni va qo'l ro'yxatini ko'radi. Buni hech
    // qanday web-kod to'sa olmaydi (OS darajasidagi cheklov) — yagona yechim
    // oldindan tushuntirish. Ustoz «boshqa eslatilmasin» desa, u qaytmaydi.
    if (isTipMuted(SHARE_WINDOW_TIP)) return setScreen(true)
    setTipOpen(true)
  }

  function confirmShare() {
    if (tipMute) muteTip(SHARE_WINDOW_TIP)
    setTipOpen(false)
    // Modal tugmasini bosish — foydalanuvchi harakati, ya'ni getDisplayMedia
    // (va undan keyingi Document PiP) uchun kerakli "ruxsat" shu zanjirda qoladi.
    void setScreen(true)
  }

  function togglePanel(p) {
    setPanel(panel === p ? 'none' : p)
  }

  return (
    <div className="controls">
      <Ctrl label={micTitle} onClick={toggleMic} disabled={!canPublish || !mediaOk || micLocked} danger={!micOn}>
        {micOn ? <Mic size={20} /> : <MicOff size={20} />}
      </Ctrl>
      <Ctrl label={camTitle} onClick={toggleCam} disabled={!canPublish || !mediaOk} danger={!camOn}>
        {camOn ? <Video size={20} /> : <VideoOff size={20} />}
      </Ctrl>
      {/* Ekran ulashish FAQAT ustozda — o'quvchi tokeniga bu manba imzolanmagan
          (backend `studentPublishSources`). Ishlamaydigan tugmani ko'rsatmaymiz. */}
      {isHost && (
        <Ctrl label="Ekran" onClick={toggleScreen} on={screenOn}>
          <MonitorUp size={20} />
        </Ctrl>
      )}

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

      <Modal open={tipOpen} onClose={() => setTipOpen(false)} title="Bitta oynani ulashing" width={460}>
        <p className="text-2" style={{ fontSize: 14, marginBottom: 12 }}>
          Keyingi oynada <b>«Butun ekran»</b> emas, <b>bitta oyna yoki brauzer tabini</b> tanlang.
        </p>
        <p className="text-2" style={{ fontSize: 14, marginBottom: 16 }}>
          Butun ekran ulashilsa, chat va qo‘l signallari turgan suzuvchi oyna ham efirga tushadi —
          o‘quvchilar shaxsiy xabarlarni ko‘rib qoladi. Bitta oyna ulashsangiz, suzuvchi oyna faqat
          sizga ko‘rinadi.
        </p>
        <label className="row gap-2" style={{ fontSize: 13, cursor: 'pointer', marginBottom: 20 }}>
          <input type="checkbox" checked={tipMute} onChange={(e) => setTipMute(e.target.checked)} />
          Boshqa eslatilmasin
        </label>
        <div className="row gap-3" style={{ justifyContent: 'flex-end' }}>
          <Button variant="ghost" onClick={() => setTipOpen(false)}>
            Bekor qilish
          </Button>
          <Button onClick={confirmShare}>Tushunarli, davom etish</Button>
        </div>
      </Modal>
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
