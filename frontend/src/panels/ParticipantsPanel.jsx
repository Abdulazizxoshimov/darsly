import { memo, useMemo, useState } from 'react'
import { Check, Hand, MicOff, Search, UserX, Video, VideoOff, X } from 'lucide-react'
import { useWaiting, useAdmit, useAdmitAll, useReject } from '../store/data'
import { allowSpeak, muteAll, muteParticipant, removeParticipant, revokeSpeak } from '../api/room'
import { errorText } from '../api/api'
import { Avatar } from '../components/Avatar'
import { Button } from '../components/Button'
import { Modal } from '../components/Modal'
import { Toggle } from '../components/Toggle'
import { toast } from '../lib/toast'

// Qidiruv maydoni shu sondan boshlab ko'rinadi: undan kichik ro'yxatni ko'z
// bilan o'qib chiqish qidirishdan tez.
const SEARCH_FROM = 10

// Ishtirokchilar paneli. `participants` — `useRoom` snapshot'i.
//
// Qo'l ko'targanlar ALOHIDA, ENG TEPADA va NAVBAT TARTIBIDA ko'rsatiladi: ustoz
// 40 kishilik ro'yxatni ko'zi bilan qidirib chiqmasligi kerak. Har qatorda ikki
// amal — "so'zga ruxsat" (bir bosishda) va "qo'lni tushirish".
export const ParticipantsPanel = memo(function ParticipantsPanel({
  participants,
  isHost,
  lessonId,
  localId,
  raisedHands,
  allowSelfUnmute = true,
  onPolicyChanged,
  onLowerHand,
  onLowerAllHands,
  onClose,
}) {
  const { data: waitingData } = useWaiting(lessonId, isHost && !!lessonId)
  const waiting = waitingData || [] // backend bo'sh ro'yxatni null qaytaradi → null.length crash bo'lmasin
  const admit = useAdmit()
  const reject = useReject()

  // "Hammani o'chirish" modali (Zoom andozasi: checkbox bilan).
  const [muteAllOpen, setMuteAllOpen] = useState(false)
  // Chiqarish tanlovi: qaysi ishtirokchi uchun dialog ochiq ({identity, name} | null).
  const [removing, setRemoving] = useState(null)
  // «Hammasini kiritish» tasdiq dialogi.
  const [admitAllOpen, setAdmitAllOpen] = useState(false)
  // Ism bo'yicha qidiruv — 300 kishilik darsda aniq odamni ko'z bilan topib
  // bo'lmaydi. Kichik ro'yxatda maydon shunchaki joy egallardi (pastda `SEARCH_FROM`).
  const [query, setQuery] = useState('')

  const present = new Set(participants.map((p) => p.identity))
  const byId = new Map(participants.map((p) => [p.identity, p]))
  const handQueue = [...raisedHands.entries()].filter(([id]) => id !== localId && present.has(id))

  const showSearch = participants.length >= SEARCH_FROM
  const q = query.trim().toLowerCase()
  // Filtr FAQAT asosiy ro'yxatga tegadi: kutayotganlar va qo'l ko'targanlar —
  // amal talab qiladigan navbatlar, ular qidiruv ostida yashirinmasligi kerak.
  const shown = useMemo(
    () => (q ? participants.filter((p) => (p.name || '').toLowerCase().includes(q)) : participants),
    [participants, q],
  )

  async function hostAction(fn, ok) {
    try {
      await fn()
      if (ok) toast.success(ok)
    } catch (e) {
      toast.error(errorText(e))
    }
  }

  return (
    <div className="panel">
      <div className="panel__head">
        <h3 className="h2" style={{ whiteSpace: 'nowrap' }}>
          {/* Qidiruv faol bo'lsa «topilgan/jami» — necha kishi yashiringani ko'rinib tursin. */}
          Ishtirokchilar ({q ? `${shown.length}/${participants.length}` : participants.length})
        </h3>
        <div className="row gap-2">
          {isHost && lessonId && (
            <button
              className="icon-btn"
              // Tor panelda matn ikki qatorga bo'linib sarlavhani siqib qo'yardi.
              style={{ fontSize: 12, fontWeight: 600, width: 'auto', whiteSpace: 'nowrap' }}
              onClick={() => setMuteAllOpen(true)}
            >
              Hammani o'chirish
            </button>
          )}
          <button className="icon-btn" onClick={onClose} aria-label="Yopish">
            <X size={20} />
          </button>
        </div>
      </div>

      <div className="panel__body">
        {isHost && waiting.length > 0 && (
          <div style={{ padding: 12, borderBottom: '1px solid var(--border-subtle)' }}>
            <div className="row gap-2" style={{ marginBottom: 8 }}>
              <span className="row gap-2 grow" style={{ fontSize: 12, fontWeight: 700, color: 'var(--warning)' }}>
                <Hand size={14} /> Kutmoqda ({waiting.length})
              </span>
              {/* Navbat bo'sh bo'lmaganda ko'rinadi: 100+ kishilik darsda har
                  so'rovni alohida bosish real emas. Tasdiq — bir bosishda
                  butun navbat kirib qolmasin. */}
              <button
                className="icon-btn"
                style={{ fontSize: 12, fontWeight: 600, width: 'auto', whiteSpace: 'nowrap' }}
                onClick={() => setAdmitAllOpen(true)}
              >
                Hammasini kiritish
              </button>
            </div>
            <div className="col gap-2">
              {waiting.map((w) => (
                <div key={w.id} className="row gap-2" style={{ background: 'var(--elevated)', borderRadius: 12, padding: 8 }}>
                  <Avatar name={w.requester_name} size={32} />
                  <span className="grow truncate" style={{ fontSize: 14 }}>{w.requester_name}</span>
                  <button
                    className="mini-btn"
                    style={{ background: 'var(--success-soft)', color: 'var(--success)', borderColor: 'transparent' }}
                    onClick={() => admit.mutate(w.id)}
                    title="Qabul qilish"
                  >
                    <Check size={16} />
                  </button>
                  <button
                    className="mini-btn"
                    style={{ background: 'var(--danger-soft)', color: 'var(--danger)', borderColor: 'transparent' }}
                    onClick={() => reject.mutate(w.id)}
                    title="Rad etish"
                  >
                    <X size={16} />
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {isHost && handQueue.length > 0 && (
          <div style={{ padding: 12, borderBottom: '1px solid var(--border-subtle)' }}>
            <div className="row gap-2" style={{ marginBottom: 8 }}>
              <span className="row gap-2 grow" style={{ fontSize: 12, fontWeight: 700, color: 'var(--warning)' }}>
                <Hand size={14} /> Qo'l ko'targanlar ({handQueue.length})
              </span>
              <button
                className="icon-btn"
                style={{ fontSize: 12, fontWeight: 600, width: 'auto' }}
                onClick={onLowerAllHands}
              >
                Hammasini tushirish
              </button>
            </div>
            <div className="col gap-2">
              {handQueue.map(([id, h], i) => {
                const p = byId.get(id)
                const canPublishCamera = p?.canPublishCamera ?? false
                return (
                  <div key={id} className="row gap-2" style={{ background: 'var(--elevated)', borderRadius: 12, padding: 8 }}>
                    <span className="rail-num">{i + 1}</span>
                    <span className="grow truncate" style={{ fontSize: 14 }}>{h.name}</span>
                    {/* Ovoz o'quvchida allaqachon ochiq — ustoz VIDEOga ruxsat beradi. */}
                    {lessonId && !canPublishCamera && (
                      <button
                        className="mini-btn"
                        style={{ background: 'var(--success-soft)', color: 'var(--success)', borderColor: 'transparent' }}
                        title="Videoga ruxsat berish"
                        onClick={() => hostAction(() => allowSpeak(lessonId, id), 'Video yoqildi')}
                      >
                        <Video size={16} />
                      </button>
                    )}
                    <button className="mini-btn" title="Qo'lni tushirish" onClick={() => onLowerHand(id)}>
                      <X size={16} />
                    </button>
                  </div>
                )
              })}
            </div>
          </div>
        )}

        <div style={{ padding: 12 }} className="col gap-1">
          {showSearch && (
            <div className="p-search">
              <Search size={15} />
              <input
                type="search"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Ism bo'yicha qidirish"
                aria-label="Ishtirokchilarni qidirish"
              />
            </div>
          )}

          {shown.length === 0 && (
            <p className="muted" style={{ fontSize: 13, padding: '12px 4px', textAlign: 'center' }}>
              Hech kim topilmadi
            </p>
          )}

          {shown.map((p) => {
            const isSelf = p.identity === localId
            return (
              <div key={p.identity} className="p-row">
                <Avatar name={p.name} size={34} />
                <div className="grow truncate" style={{ fontSize: 14, fontWeight: 500 }}>
                  {p.name}
                  {isSelf && ' (siz)'}
                </div>
                {raisedHands.has(p.identity) && <Hand size={16} color="var(--warning)" />}
                {p.micMuted && <MicOff size={16} color="var(--text-3)" />}
                {isHost && !isSelf && lessonId && (
                  <div className="p-row__actions">
                    <button
                      className="mini-btn"
                      title={p.canPublishCamera ? 'Videoni bekor qilish' : 'Videoga ruxsat'}
                      onClick={() =>
                        hostAction(
                          () => (p.canPublishCamera ? revokeSpeak(lessonId, p.identity) : allowSpeak(lessonId, p.identity)),
                          p.canPublishCamera ? 'Video bekor qilindi' : 'Video yoqildi',
                        )
                      }
                    >
                      {p.canPublishCamera ? (
                        <Video size={16} color="var(--accent-light)" />
                      ) : (
                        <VideoOff size={16} />
                      )}
                    </button>
                    <button className="mini-btn" title="Mute" onClick={() => hostAction(() => muteParticipant(lessonId, p.identity))}>
                      <MicOff size={16} />
                    </button>
                    <button
                      className="mini-btn"
                      title="Chiqarib yuborish"
                      onClick={() => setRemoving({ identity: p.identity, name: p.name })}
                    >
                      <UserX size={16} color="var(--danger)" />
                    </button>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>

      {isHost && lessonId && (
        <MuteAllModal
          open={muteAllOpen}
          allowSelfUnmute={allowSelfUnmute}
          onPolicyChanged={onPolicyChanged}
          lessonId={lessonId}
          onClose={() => setMuteAllOpen(false)}
        />
      )}
      {isHost && lessonId && removing && (
        <RemoveModal lessonId={lessonId} participant={removing} onClose={() => setRemoving(null)} />
      )}
      {isHost && lessonId && admitAllOpen && (
        <AdmitAllModal lessonId={lessonId} count={waiting.length} onClose={() => setAdmitAllOpen(false)} />
      )}
    </div>
  )
})

// «Hammani o'chirish» — Zoom andozasi: barcha mikrofonlar o'chadi, checkbox esa
// o'quvchilar KEYIN o'zlari ocha oladimi-yo'qligini belgilaydi (allow_self_unmute).
function MuteAllModal({ open, lessonId, allowSelfUnmute, onPolicyChanged, onClose }) {
  // Checkbox «o'zi ocholmasin» — joriy siyosatning teskarisi bilan boshlanadi.
  const [dontAllow, setDontAllow] = useState(!allowSelfUnmute)
  const [busy, setBusy] = useState(false)

  async function confirm() {
    setBusy(true)
    try {
      await muteAll(lessonId, !dontAllow)
      onPolicyChanged?.(!dontAllow)
      toast.success('Hammaning mikrofoni o‘chirildi')
      onClose()
    } catch (e) {
      toast.error(errorText(e, 'Mute qilib bo‘lmadi'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal open={open} onClose={() => !busy && onClose()} title="Hammani o'chirish" width={420}>
      <p className="text-2" style={{ fontSize: 14, marginBottom: 16 }}>
        Barcha o'quvchilarning mikrofoni o'chiriladi (sizniki qolmaydi).
      </p>
      <Toggle label="O'quvchilar o'zi ocholmasin" checked={dontAllow} onChange={setDontAllow} />
      <div className="row gap-3" style={{ justifyContent: 'flex-end', marginTop: 20 }}>
        <Button variant="ghost" onClick={onClose} disabled={busy}>
          Bekor qilish
        </Button>
        <Button onClick={confirm} loading={busy}>
          O'chirish
        </Button>
      </div>
    </Modal>
  )
}

// «Hammasini kiritish» — butun kutish navbatini bitta so'rovda kiritadi.
//
// Tasdiq MAJBURIY: bu qaytarib bo'lmaydigan amal (kirgan odamni faqat qo'lda
// chiqarib yuborish mumkin), shuning uchun nechta kishi kiritilishi aniq yoziladi.
// Qisman muvaffaqiyat NORMAL: ro'yxat o'qilgandan keyin alohida hal qilingan
// so'rov o'tkazib yuboriladi — natijada ROSTINI aytamiz, yashirmaymiz.
function AdmitAllModal({ lessonId, count, onClose }) {
  const admitAll = useAdmitAll()

  async function confirm() {
    try {
      const res = await admitAll.mutateAsync(lessonId)
      const admitted = res?.admitted ?? 0
      const failed = res?.failed ?? 0
      if (admitted === 0 && failed === 0) toast.info('Navbat bo‘sh — hamma allaqachon hal qilingan')
      else if (failed > 0) toast.info(`${admitted} o‘quvchi kiritildi · ${failed} tasi kiritilmadi`)
      else toast.success(`${admitted} o‘quvchi kiritildi`)
      onClose()
    } catch (e) {
      toast.error(errorText(e, 'Kiritib bo‘lmadi'))
    }
  }

  return (
    <Modal open onClose={() => !admitAll.isPending && onClose()} title="Hammasini kiritish" width={420}>
      <p className="text-2" style={{ fontSize: 14, marginBottom: 20 }}>
        Kutayotgan <strong>{count}</strong> o‘quvchi darsga kiritiladi. Keyin ularni faqat
        qo‘lda chiqarib yuborish mumkin.
      </p>
      <div className="row gap-3" style={{ justifyContent: 'flex-end' }}>
        <Button variant="ghost" onClick={onClose} disabled={admitAll.isPending}>
          Bekor qilish
        </Button>
        <Button onClick={confirm} loading={admitAll.isPending}>
          Kiritish
        </Button>
      </div>
    </Modal>
  )
}

// Chiqarish tanlovi (№4): bir darslik yoki mentor bo'yicha doimiy blok.
// Doimiy blok QAYTARILADI (Qora ro'yxat sahifasidan) — shuning uchun alohida
// tasdiq bosqichisiz ikkita aniq tugma yetarli.
function RemoveModal({ lessonId, participant, onClose }) {
  const [busy, setBusy] = useState(false)

  async function doRemove(scope) {
    setBusy(true)
    try {
      await removeParticipant(lessonId, participant.identity, scope)
      toast.success(
        scope === 'mentor'
          ? `${participant.name} barcha darslaringizdan bloklandi`
          : `${participant.name} darsdan chiqarildi`,
      )
      onClose()
    } catch (e) {
      toast.error(errorText(e, 'Chiqarib bo‘lmadi'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal open onClose={() => !busy && onClose()} title="Chiqarib yuborish" width={440}>
      <p className="text-2" style={{ fontSize: 14, marginBottom: 20 }}>
        <strong>{participant.name}</strong> qanday chiqarilsin? Doimiy blok «Qora ro'yxat»
        sahifasidan bekor qilinadi.
      </p>
      <div className="col gap-3">
        <Button variant="ghost" onClick={() => doRemove('lesson')} disabled={busy} className="full">
          Shu darsdan
        </Button>
        <Button variant="danger" onClick={() => doRemove('mentor')} disabled={busy} className="full">
          Doimiy (barcha darslarimdan)
        </Button>
      </div>
    </Modal>
  )
}
