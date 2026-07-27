import { useState } from 'react'
import { BarChart3, Plus, Trash2, X } from 'lucide-react'
import { usePolls, useCreatePoll, useClosePoll, usePollResults } from '../store/data'
import { votePoll } from '../api/polls'
import { errorText } from '../api/api'
import { Button } from '../components/Button'
import { toast } from '../lib/toast'

export function PollsPanel({ isHost, lessonId, roomToken, guestActivePoll, onBroadcastPoll, onClose }) {
  return (
    <div className="panel">
      <div className="panel__head">
        <h3 className="h2">So'rovnomalar</h3>
        <button className="icon-btn" onClick={onClose} aria-label="Yopish">
          <X size={20} />
        </button>
      </div>
      <div className="panel__body" style={{ padding: 16 }}>
        {isHost ? (
          <HostPolls lessonId={lessonId} onBroadcastPoll={onBroadcastPoll} />
        ) : (
          <GuestPoll guestActivePoll={guestActivePoll} roomToken={roomToken} />
        )}
      </div>
    </div>
  )
}

function ResultBars({ results, options }) {
  const counts = results?.counts ?? options.map(() => 0)
  const total = results?.total ?? 0
  return (
    <div className="col gap-2" style={{ marginTop: 4 }}>
      {options.map((o, i) => {
        const pct = total > 0 ? Math.round((counts[i] / total) * 100) : 0
        return (
          <div key={i}>
            <div className="row between" style={{ fontSize: 13, marginBottom: 2 }}>
              <span style={{ color: 'var(--text-bright)' }}>{o}</span>
              <span className="muted">{pct}%</span>
            </div>
            <div className="poll-bar-track">
              <div className="poll-bar-fill" style={{ width: `${pct}%` }} />
            </div>
          </div>
        )
      })}
      <p className="muted" style={{ fontSize: 12, marginTop: 4 }}>{total} ovoz</p>
    </div>
  )
}

function HostPolls({ lessonId, onBroadcastPoll }) {
  const { data: polls = [] } = usePolls(lessonId)
  const create = useCreatePoll()
  const close = useClosePoll()
  const [creating, setCreating] = useState(false)
  const [question, setQuestion] = useState('')
  const [options, setOptions] = useState(['', ''])

  async function submit() {
    const opts = options.map((o) => o.trim()).filter(Boolean)
    try {
      const poll = await create.mutateAsync({ lessonId, question: question.trim(), options: opts })
      onBroadcastPoll('open', { id: poll.id, question: poll.question, options: poll.options })
      toast.success("So'rovnoma boshlandi")
      setCreating(false)
      setQuestion('')
      setOptions(['', ''])
    } catch (e) {
      toast.error(errorText(e))
    }
  }

  async function doClose(poll) {
    try {
      const res = await close.mutateAsync(poll.id)
      onBroadcastPoll('close', { id: res.poll.id, question: res.poll.question, options: res.poll.options })
      toast.info("So'rovnoma yopildi")
    } catch (e) {
      toast.error(errorText(e))
    }
  }

  if (creating) {
    return (
      <div className="col gap-3">
        <input className="input" value={question} onChange={(e) => setQuestion(e.target.value)} placeholder="Savol" />
        {options.map((o, i) => (
          <div key={i} className="row gap-2">
            <input
              className="input"
              value={o}
              onChange={(e) => setOptions((p) => p.map((x, j) => (j === i ? e.target.value : x)))}
              placeholder={`Variant ${i + 1}`}
            />
            {options.length > 2 && (
              <button className="mini-btn" style={{ width: 36, height: 42 }} onClick={() => setOptions((p) => p.filter((_, j) => j !== i))}>
                <Trash2 size={16} />
              </button>
            )}
          </div>
        ))}
        {options.length < 10 && (
          <button
            className="row gap-1"
            style={{ background: 'none', border: 'none', color: 'var(--accent-light)', fontSize: 13, fontWeight: 600, cursor: 'pointer' }}
            onClick={() => setOptions((p) => [...p, ''])}
          >
            <Plus size={16} /> Variant qo'shish
          </button>
        )}
        <div className="row gap-2" style={{ marginTop: 4 }}>
          <Button variant="ghost" size="sm" className="grow" onClick={() => setCreating(false)}>
            Bekor
          </Button>
          <Button
            size="sm"
            className="grow"
            loading={create.isPending}
            disabled={!question.trim() || options.filter((o) => o.trim()).length < 2}
            onClick={submit}
          >
            Boshlash
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="col gap-3">
      <Button size="sm" onClick={() => setCreating(true)}>
        <Plus size={16} /> Yangi so'rovnoma
      </Button>
      {polls.length === 0 && (
        <p className="muted" style={{ textAlign: 'center', fontSize: 14, marginTop: 24 }}>Hali so'rovnoma yo'q</p>
      )}
      {polls.map((p) => (
        <HostPollCard key={p.id} poll={p} onClose={() => doClose(p)} closing={close.isPending} />
      ))}
    </div>
  )
}

function HostPollCard({ poll, onClose, closing }) {
  const { data } = usePollResults(poll.id, { refetchInterval: poll.is_active ? 3000 : false })
  return (
    <div style={{ background: 'var(--elevated)', border: '1px solid var(--border-strong)', borderRadius: 12, padding: 14 }}>
      <div className="row between" style={{ marginBottom: 8 }}>
        <p style={{ fontSize: 14, fontWeight: 600 }}>{poll.question}</p>
        <span style={{ fontSize: 11, fontWeight: 700, color: poll.is_active ? 'var(--success)' : 'var(--text-3)' }}>
          {poll.is_active ? 'Faol' : 'Yopiq'}
        </span>
      </div>
      <ResultBars results={data} options={poll.options} />
      {poll.is_active && (
        <Button variant="ghost" size="sm" className="full" style={{ marginTop: 8 }} onClick={onClose} loading={closing}>
          Yopish va natija
        </Button>
      )}
    </div>
  )
}

function GuestPoll({ guestActivePoll, roomToken }) {
  const [voted, setVoted] = useState(null)
  const { data: results } = usePollResults(guestActivePoll?.id, {
    enabled: !!guestActivePoll && voted === guestActivePoll?.id,
    refetchInterval: 3000,
  })

  async function vote(idx) {
    try {
      await votePoll(guestActivePoll.id, roomToken.token, idx)
      setVoted(guestActivePoll.id)
      toast.success('Ovoz berildi')
    } catch (e) {
      toast.error(errorText(e))
    }
  }

  if (!guestActivePoll) {
    return (
      <div className="col center" style={{ marginTop: 40, color: 'var(--text-3)', textAlign: 'center' }}>
        <BarChart3 size={32} style={{ marginBottom: 8 }} />
        <p style={{ fontSize: 14 }}>Hozircha faol so'rovnoma yo'q</p>
      </div>
    )
  }

  return (
    <div>
      <p style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>{guestActivePoll.question}</p>
      {voted === guestActivePoll.id ? (
        <ResultBars results={results} options={guestActivePoll.options} />
      ) : (
        <div className="col gap-2">
          {guestActivePoll.options.map((o, i) => (
            <button
              key={i}
              onClick={() => vote(i)}
              style={{
                textAlign: 'left',
                background: 'var(--elevated)',
                border: '1px solid var(--border-strong)',
                borderRadius: 12,
                padding: '10px 14px',
                fontSize: 14,
                color: 'var(--text)',
                cursor: 'pointer',
              }}
            >
              {o}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
