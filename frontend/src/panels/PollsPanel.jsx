import { useState } from 'react'
import { AlertTriangle, BarChart3, Eye, EyeOff, Lock, Megaphone, Plus, Trash2, X } from 'lucide-react'
import { usePolls, useCreatePoll, useClosePoll, usePollResults, usePublishPoll } from '../store/data'
import { votePoll, voteErrorText, RESULTS_VISIBILITY } from '../api/polls'
import { errorText } from '../api/api'
import { Button } from '../components/Button'
import { Spinner } from '../components/Spinner'
import { toast } from '../lib/toast'

// So'rovnoma paneli.
//
// Natija KO'RINUVCHANLIGI ikki rejimda (mahsulot qoidasi №7):
//  · mentor_only (default) — natija faqat ustozda. Yopiq tomon: eski klient
//    yoki e'tiborsiz ustoz natijani TASODIFAN ochib yubormaydi.
//  · public — o'quvchi ham ko'radi, LEKIN faqat ustoz «E'lon qilish» bosgach.
//
// Yopish ≠ e'lon qilish: yopish faqat ovoz berishni to'xtatadi. Bu ikkisi
// atayin ajratilgan — ustoz odatda ovozni yopib, natijani muhokamadan keyin
// ko'rsatadi.
// `roomToken` — JORIY xona tokeni (natija so'rovlari uchun; yangilanganda
// keyingi so'rov yangi tokenni oladi). `withRoomToken(fn)` — `fn(token)` ni
// bajaradi va 401 da tokenni yangilab BIR MARTA takrorlaydi (`lib/roomToken`):
// ovoz berish 30 daqiqadan keyin ham jimgina ishlayveradi.
export function PollsPanel({
  isHost,
  lessonId,
  roomToken,
  withRoomToken,
  guestActivePoll,
  publishedResults,
  votedPollId,
  onVoted,
  onBroadcastPoll,
  onClose,
}) {
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
          <HostPolls lessonId={lessonId} roomToken={roomToken} onBroadcastPoll={onBroadcastPoll} />
        ) : (
          <GuestPoll
            guestActivePoll={guestActivePoll}
            roomToken={roomToken}
            withRoomToken={withRoomToken}
            publishedResults={publishedResults}
            votedPollId={votedPollId}
            onVoted={onVoted}
          />
        )}
      </div>
    </div>
  )
}

function ResultBars({ results, options, error }) {
  // XATO ≠ NOL OVOZ. Tarmoq xatosida bo'sh diagramma chizish ustozga
  // "hech kim ovoz bermadi" deb yolg'on aytardi va u shunga qarab qaror
  // qabul qilardi (masalan mavzuni qaytadan tushuntirardi).
  if (error) {
    return (
      <p className="poll-load-error">
        <AlertTriangle size={13} /> Natijani yuklab bo‘lmadi
      </p>
    )
  }
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

// `roomToken` SHART: natijalar endpointi room-token talab qiladi. U prop
// sifatida uzatilmasa `HostPollCard` ga `undefined` ketardi — aslida esa
// `roomToken` `HostPolls` qamrovida umuman e'lon qilinmagan edi va birinchi
// so'rovnoma paydo bo'lishi bilan render `ReferenceError` bilan yiqilardi
// (ustozning so'rovnomalar paneli ishlamas edi). ESLint `no-undef` bilan topdi.
function HostPolls({ lessonId, roomToken, onBroadcastPoll }) {
  const { data: polls = [], isLoading, isError, refetch } = usePolls(lessonId)
  const create = useCreatePoll()
  const close = useClosePoll()
  const publish = usePublishPoll()
  const [creating, setCreating] = useState(false)
  const [question, setQuestion] = useState('')
  const [options, setOptions] = useState(['', ''])
  const [visibility, setVisibility] = useState(RESULTS_VISIBILITY.MENTOR_ONLY)

  async function submit() {
    const opts = options.map((o) => o.trim()).filter(Boolean)
    try {
      const poll = await create.mutateAsync({
        lessonId,
        question: question.trim(),
        options: opts,
        resultsVisibility: visibility,
      })
      // O'quvchiga so'rovnomani host e'lon qiladi. `results_visibility` ham
      // uzatiladi: o'quvchi ovoz bergach «natija e'lon qilinmagan» deymizmi
      // yoki «faqat ustoz ko'radi» deymizmi — javob shunga bog'liq.
      onBroadcastPoll('open', {
        id: poll.id,
        question: poll.question,
        options: poll.options,
        results_visibility: poll.results_visibility || visibility,
      })
      toast.success("So'rovnoma boshlandi")
      setCreating(false)
      setQuestion('')
      setOptions(['', ''])
      setVisibility(RESULTS_VISIBILITY.MENTOR_ONLY)
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

  // Natijani serverning O'ZI xonaga tarqatadi (`poll_published` data-xabari),
  // shuning uchun klient qo'shimcha broadcast qilmaydi.
  async function doPublish(poll) {
    try {
      await publish.mutateAsync({ lessonId, pollId: poll.id })
      toast.success("Natija e'lon qilindi")
    } catch (e) {
      toast.error(errorText(e, "Natijani e'lon qilib bo'lmadi"))
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

        <div className="field" style={{ marginTop: 4 }}>
          <label className="field__label" htmlFor="poll-visibility">
            Natija kimga ko'rinadi
          </label>
          <select
            id="poll-visibility"
            className="input"
            value={visibility}
            onChange={(e) => setVisibility(e.target.value)}
          >
            <option value={RESULTS_VISIBILITY.MENTOR_ONLY}>Faqat menga</option>
            <option value={RESULTS_VISIBILITY.PUBLIC}>Hammaga — men e'lon qilganimdan keyin</option>
          </select>
          <p className="muted" style={{ fontSize: 12, marginTop: 6 }}>
            {visibility === RESULTS_VISIBILITY.PUBLIC
              ? "O'quvchilar natijani siz «E'lon qilish» bosgandan keyin ko'radi."
              : "Natijani faqat siz ko'rasiz — keyinchalik e'lon qilib bo'lmaydi."}
          </p>
        </div>

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

      {/* Uch holat ATAYLAB ajratilgan: yuklanmoqda ≠ bo'sh ≠ xato. Avval
          uchalasi ham "Hali so'rovnoma yo'q" deb ko'rinardi va so'rov yiqilganda
          ustoz o'zi yaratgan so'rovnomani yo'qolgan deb o'ylardi. */}
      {isLoading ? (
        <div className="row center" style={{ height: 100, justifyContent: 'center' }}>
          <Spinner size={24} />
        </div>
      ) : isError ? (
        <div className="poll-hidden">
          <AlertTriangle size={20} />
          <p className="text-2" style={{ fontSize: 13, margin: '8px 0 10px' }}>
            So‘rovnomalarni yuklab bo‘lmadi.
          </p>
          <Button variant="ghost" size="sm" onClick={() => refetch()}>
            Qayta urinish
          </Button>
        </div>
      ) : polls.length === 0 ? (
        <p className="muted" style={{ textAlign: 'center', fontSize: 14, marginTop: 24 }}>Hali so'rovnoma yo'q</p>
      ) : (
        polls.map((p) => (
          <HostPollCard
            key={p.id}
            poll={p}
            roomToken={roomToken}
            onClose={() => doClose(p)}
            // Pending FAQAT o'sha kartochkada: avval bitta tugma bosilganda
            // hamma kartochkadagi tugmalar spinnerga aylanardi.
            closing={close.isPending && close.variables === p.id}
            onPublish={() => doPublish(p)}
            publishing={publish.isPending && publish.variables?.pollId === p.id}
          />
        ))
      )}
    </div>
  )
}

function HostPollCard({ poll, roomToken, onClose, closing, onPublish, publishing }) {
  const { data, isError } = usePollResults(poll.id, roomToken?.token, {
    refetchInterval: poll.is_active ? 3000 : false,
  })
  const isPublic = poll.results_visibility === RESULTS_VISIBILITY.PUBLIC
  const published = !!poll.results_published_at

  return (
    <div style={{ background: 'var(--elevated)', border: '1px solid var(--border-strong)', borderRadius: 12, padding: 14 }}>
      <div className="row between" style={{ marginBottom: 8 }}>
        <p style={{ fontSize: 14, fontWeight: 600 }}>{poll.question}</p>
        <span style={{ fontSize: 11, fontWeight: 700, color: poll.is_active ? 'var(--success)' : 'var(--text-3)' }}>
          {poll.is_active ? 'Faol' : 'Yopiq'}
        </span>
      </div>

      {/* Ustoz natija KIMGA ko'rinishini bir qarashda bilishi kerak: aks holda
          "o'quvchilar ko'rdimi?" degan savol har safar taxminga aylanadi. */}
      <span className={`poll-vis ${isPublic ? 'poll-vis--public' : ''}`}>
        {isPublic ? <Eye size={12} /> : <Lock size={12} />}
        {isPublic ? (published ? "Natija e'lon qilingan" : "Natija hali e'lon qilinmagan") : "Natija faqat sizda"}
      </span>

      <ResultBars results={data} options={poll.options} error={isError && !data} />

      {isPublic && !published && (
        <Button variant="secondary" size="sm" className="full" style={{ marginTop: 8 }} onClick={onPublish} loading={publishing}>
          <Megaphone size={15} /> Natijani e'lon qilish
        </Button>
      )}
      {poll.is_active && (
        <Button variant="ghost" size="sm" className="full" style={{ marginTop: 8 }} onClick={onClose} loading={closing}>
          Yopish va natija
        </Button>
      )}
    </div>
  )
}

// `voted` holati ATAYLAB tashqarida (`LiveRoom`): panel yopilib qayta
// ochilganda bu komponent unmount bo'ladi va lokal holat yo'qolardi —
// o'quvchi variantlarni yana ko'rib, qayta ovoz berishga urinardi.
function GuestPoll({ guestActivePoll, roomToken, withRoomToken, publishedResults, votedPollId, onVoted }) {
  const [voting, setVoting] = useState(false)
  const [voteError, setVoteError] = useState(null)
  const isPublic = guestActivePoll?.results_visibility === RESULTS_VISIBILITY.PUBLIC

  // Data-channel'dan kelgan natija — AYNI shu so'rovnomaniki bo'lsagina.
  const pushed = publishedResults?.poll?.id === guestActivePoll?.id ? publishedResults : null

  // Bir martalik so'rov: natija ovoz berishimizdan OLDIN e'lon qilingan bo'lishi
  // mumkin (kech kirgan o'quvchi hodisani ko'rmagan). E'lon qilinmagan bo'lsa
  // server 403 beradi — bu KUTILGAN javob, shuning uchun qayta urinmaymiz va
  // xato ham ko'rsatmaymiz.
  const { data: fetched } = usePollResults(guestActivePoll?.id, roomToken?.token, {
    enabled: !!guestActivePoll && votedPollId === guestActivePoll?.id && isPublic && !pushed,
    retry: false,
  })

  const results = pushed || fetched || null

  async function vote(idx) {
    setVoting(true)
    setVoteError(null)
    try {
      await withRoomToken((t) => votePoll(guestActivePoll.id, t.token, idx))
      onVoted(guestActivePoll.id)
      toast.success('Ovoz berildi')
    } catch (e) {
      // Umumiy `errorText` bu yerda «So'rovda xatolik» derdi — o'quvchi
      // ovozi NEGA o'tmaganini bilmay, tugmani qayta-qayta bosardi.
      const text = voteErrorText(e)
      setVoteError(text)
      toast.error(text)
      // Allaqachon ovoz bergan bo'lsa — variantlarni qayta ko'rsatishning
      // ma'nosi yo'q, natija/kutish holatiga o'tkazamiz.
      if (e?.code === 'CONFLICT') onVoted(guestActivePoll.id)
    } finally {
      setVoting(false)
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

  const hasVoted = votedPollId === guestActivePoll.id
  const closed = guestActivePoll.is_active === false

  return (
    <div>
      <p style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>{guestActivePoll.question}</p>
      {/* Natija ko'rsatilayotganda «yopilgan» belgisi kerak (ovoz berish
          tugagani ayon bo'lsin); natija yo'q holatda buni pastdagi blok
          aytadi — ikki marta takrorlamaymiz. */}
      {closed && results && (
        <p className="poll-vis" style={{ marginBottom: 10 }}>
          <Lock size={12} /> So‘rovnoma yopilgan
        </p>
      )}

      {hasVoted || results || closed ? (
        results ? (
          <ResultBars results={results} options={guestActivePoll.options} />
        ) : (
          // Natija YO'Q va bo'sh diagramma ham chizilmaydi: «0 ovoz» bilan
          // «ko'rsatilmaydi» ni farqlab bo'lmasa o'quvchi noto'g'ri xulosa chiqaradi.
          <div className="poll-hidden">
            <EyeOff size={20} />
            <p style={{ fontSize: 14, fontWeight: 600, margin: '8px 0 4px' }}>
              {hasVoted ? 'Ovozingiz qabul qilindi' : 'So‘rovnoma yopilgan'}
            </p>
            <p className="text-2" style={{ fontSize: 13, margin: 0 }}>
              {isPublic
                ? "Natijani ustoz hali e'lon qilmagan."
                : "Natijani faqat ustoz ko'radi."}
            </p>
          </div>
        )
      ) : (
        <div className="col gap-2">
          {guestActivePoll.options.map((o, i) => (
            <button key={i} className="poll-option" onClick={() => vote(i)} disabled={voting}>
              {o}
            </button>
          ))}
          {voteError && (
            <p className="poll-load-error" role="alert">
              <AlertTriangle size={13} /> {voteError}
            </p>
          )}
        </div>
      )}
    </div>
  )
}
