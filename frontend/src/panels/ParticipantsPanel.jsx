import { Check, Hand, Mic, MicOff, UserX, X } from 'lucide-react'
import { Track } from 'livekit-client'
import { roomParticipants } from '../livekit/useRoom'
import { useWaiting, useAdmit, useReject } from '../store/data'
import { allowSpeak, muteAll, muteParticipant, removeParticipant, revokeSpeak } from '../api/room'
import { errorText } from '../api/api'
import { Avatar } from '../components/Avatar'
import { toast } from '../lib/toast'

export function ParticipantsPanel({ room, isHost, lessonId, raisedHands, onClose }) {
  const participants = roomParticipants(room)
  const localId = room.localParticipant.identity
  const { data: waitingData } = useWaiting(lessonId, isHost && !!lessonId)
  const waiting = waitingData || [] // backend null qaytarishi mumkin → guard
  const admit = useAdmit()
  const reject = useReject()

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
        <h3 className="h2">Ishtirokchilar ({participants.length})</h3>
        <div className="row gap-2">
          {isHost && lessonId && (
            <button
              className="icon-btn"
              style={{ fontSize: 12, fontWeight: 600, width: 'auto' }}
              onClick={() => hostAction(() => muteAll(lessonId), 'Hamma mute qilindi')}
            >
              Hammani mute
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
            <div className="row gap-2" style={{ fontSize: 12, fontWeight: 700, color: 'var(--warning)', marginBottom: 8 }}>
              <Hand size={14} /> Kutmoqda ({waiting.length})
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

        <div style={{ padding: 12 }} className="col gap-1">
          {participants.map((p) => {
            const canPublish = p.permissions?.canPublish ?? false
            const isSelf = p.identity === localId
            const micPub = p.getTrackPublication(Track.Source.Microphone)
            const muted = !micPub || micPub.isMuted
            return (
              <div key={p.identity} className="p-row">
                <Avatar name={p.name || p.identity} size={34} />
                <div className="grow truncate" style={{ fontSize: 14, fontWeight: 500 }}>
                  {p.name || p.identity}
                  {isSelf && ' (siz)'}
                </div>
                {raisedHands.has(p.identity) && <Hand size={16} color="var(--warning)" />}
                {muted && <MicOff size={16} color="var(--text-3)" />}
                {isHost && !isSelf && lessonId && (
                  <div className="p-row__actions">
                    <button
                      className="mini-btn"
                      title={canPublish ? "So'zlashni bekor qilish" : "So'zlashga ruxsat"}
                      onClick={() =>
                        hostAction(
                          () => (canPublish ? revokeSpeak(lessonId, p.identity) : allowSpeak(lessonId, p.identity)),
                          canPublish ? 'Ruxsat bekor qilindi' : 'Ruxsat berildi',
                        )
                      }
                    >
                      <Mic size={16} color={canPublish ? 'var(--accent-light)' : undefined} />
                    </button>
                    <button className="mini-btn" title="Mute" onClick={() => hostAction(() => muteParticipant(lessonId, p.identity))}>
                      <MicOff size={16} />
                    </button>
                    <button
                      className="mini-btn"
                      title="Chiqarib yuborish"
                      onClick={() => hostAction(() => removeParticipant(lessonId, p.identity), 'Chiqarildi')}
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
    </div>
  )
}
