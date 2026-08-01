import { useEffect, useRef, useState } from 'react'
import { Check, Copy, ExternalLink, Link2Off, Send, Users } from 'lucide-react'
import { useTelegramStatus, useTelegramLink, useTelegramUnlink } from '../store/data'
import { errorText } from '../api/api'
import { Button } from '../components/Button'
import { Spinner } from '../components/Spinner'
import { toast } from '../lib/toast'

// Telegram bog'lanishi (PRODUCT.md «Dars arxivi va Telegram saqlash»).
//
// Nima uchun kerak: dars videosi mentorning Telegram guruhiga saqlanadi
// (ikkinchi nusxa + o'quvchilarga tarqatish). Buning uchun bot mentorning
// KIM ekanini bilishi shart, Telegram esa buni aytmaydi — shuning uchun
// bir martalik kod: ilova kodni beradi, mentor botga `/start <kod>` yuboradi.
//
// QOIDA: `enabled:false` bo'lsa (serverda bot tokeni yo'q) bo'lim UMUMAN
// ko'rsatilmaydi. Ishlamaydigan tugma faqat savol tug'diradi.

export function TelegramSection() {
  const [code, setCode] = useState(null)
  // Kod ekranda turganda holat tez-tez so'raladi: mentor botga `/start`
  // yuborgan zahoti kartochka «Bog'langan» ga o'zi almashadi va u sahifani
  // qayta yuklashni o'ylab ham o'tirmaydi.
  //
  // Interval FUNKSIYA — sabab muhim: `code ? 5000 : undefined` bilan poll
  // bog'lanish MUVAFFAQIYATLI tugagandan keyin ham davom etardi (kod state'i
  // tozalanmaydi), ya'ni profil sahifasi ochiq turgan vaqtning hammasida
  // har 5 soniyada bekorga so'rov ketardi. Funksiya esa natijaning O'ZINI
  // ko'radi va `linked` bo'lishi bilan to'xtaydi.
  const { data, isLoading, isError } = useTelegramStatus({
    refetchInterval: (q) => (code && !q.state.data?.linked ? 5000 : false),
  })
  const link = useTelegramLink()
  const unlink = useTelegramUnlink()
  const [confirmUnlink, setConfirmUnlink] = useState(false)
  const [copied, setCopied] = useState(false)
  const copyTimer = useRef(null)
  useEffect(() => () => clearTimeout(copyTimer.current), [])

  // Kod muddati (server aytadi, odatda 15 daqiqa). Muddat tugagach kod
  // ekrandan olinadi — aks holda o'lik kod va abadiy «kutilmoqda» spinneri
  // qolar, poll esa bekorga aylanaverardi.
  useEffect(() => {
    if (!code?.expires_in_s) return undefined
    const t = setTimeout(() => setCode(null), code.expires_in_s * 1000)
    return () => clearTimeout(t)
  }, [code])

  // Yuklanayotganda ham, xatoda ham hech nima ko'rsatilmaydi: bu bo'lim
  // IXTIYORIY qo'shimcha — uning xatosi profil sahifasini buzmasligi kerak.
  // Hook'lardan KEYIN: erta `return` hook tartibini buzmasligi shart.
  if (isLoading || isError || !data?.enabled) return null

  async function startLink() {
    try {
      setCode(await link.mutateAsync())
    } catch (e) {
      toast.error(errorText(e, 'Bog‘lash kodini olib bo‘lmadi'))
    }
  }

  async function doUnlink() {
    try {
      await unlink.mutateAsync()
      setConfirmUnlink(false)
      setCode(null)
      toast.info('Telegram uzildi')
    } catch (e) {
      toast.error(errorText(e, 'Telegramni uzib bo‘lmadi'))
    }
  }

  async function copyCode() {
    try {
      // Ruxsat berilmagan/HTTP kontekstda `writeText` rad etadi — ushlanmasa
      // konsolda ushlanmagan promise xatosi qoladi va foydalanuvchi «nusxalandi»
      // degan yolg'on belgini ko'rardi.
      await navigator.clipboard?.writeText(code.code)
      setCopied(true)
      copyTimer.current = setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error('Nusxalab bo‘lmadi — kodni qo‘lda ko‘chiring')
    }
  }

  return (
    <div className="card card--pad" style={{ marginTop: 20 }}>
      <div className="row gap-2" style={{ marginBottom: 6 }}>
        <Send size={16} color="var(--accent-light)" />
        <h2 className="h2">Telegram</h2>
      </div>
      <p className="text-2" style={{ fontSize: 13.5, marginTop: 0, marginBottom: 16 }}>
        Dars videolari Telegram guruhingizga saqlanadi — bu ikkinchi nusxa va o‘quvchilarga
        tarqatishning eng oson yo‘li.
      </p>

      {data.linked ? (
        <div className="tg-linked">
          <div className="tg-linked__row">
            <span className="tg-badge">
              <Check size={13} /> Bog‘langan
            </span>
            <span className="text-2 truncate" style={{ fontSize: 14 }}>
              {data.username ? `@${data.username}` : 'Telegram akkaunt'}
            </span>
          </div>

          {data.chats.length > 0 && (
            <div className="tg-chats">
              <span className="muted row gap-2" style={{ fontSize: 12.5 }}>
                <Users size={13} /> Bot qo‘shilgan guruhlar
              </span>
              <ul className="tg-chats__list">
                {data.chats.map((c) => (
                  <li key={c.chat_id} className="truncate">{c.title}</li>
                ))}
              </ul>
            </div>
          )}

          {confirmUnlink ? (
            <div className="confirm-inline">
              <p style={{ fontSize: 14, fontWeight: 600, margin: '0 0 4px' }}>Telegramni uzasizmi?</p>
              <p className="text-2" style={{ fontSize: 13, margin: '0 0 12px' }}>
                Yangi darslar videosi Telegramga yuborilmaydi. Avval yuborilganlari guruhda qoladi.
              </p>
              <div className="row gap-2" style={{ justifyContent: 'flex-end' }}>
                <Button variant="ghost" size="sm" onClick={() => setConfirmUnlink(false)} disabled={unlink.isPending}>
                  Bekor qilish
                </Button>
                <Button variant="danger" size="sm" onClick={doUnlink} loading={unlink.isPending}>
                  Ha, uzish
                </Button>
              </div>
            </div>
          ) : (
            <Button variant="secondary" size="sm" onClick={() => setConfirmUnlink(true)}>
              <Link2Off size={15} /> Uzish
            </Button>
          )}
        </div>
      ) : /* Bog'lanish tugagach kod avtomatik yo'qoladi — yuqoridagi
             `data.linked` shoxiga o'tamiz, kodni tozalash shart emas. */
      code ? (
        <div className="tg-code">
          <p style={{ fontSize: 14, margin: '0 0 10px' }}>
            Botga quyidagi buyruqni yuboring:
          </p>
          <div className="tg-code__box">
            <code>/start {code.code}</code>
            <button type="button" className="tbl-btn" onClick={copyCode} title="Nusxalash" aria-label="Kodni nusxalash">
              {copied ? <Check size={15} /> : <Copy size={15} />}
            </button>
          </div>
          {code.deep_link && (
            <a className="btn btn--secondary btn--sm" href={code.deep_link} target="_blank" rel="noopener noreferrer">
              <ExternalLink size={15} /> Telegramda ochish
            </a>
          )}
          <div className="muted row gap-2" style={{ fontSize: 12.5, marginTop: 12 }}>
            <Spinner size={12} />
            {/* Muddat SERVERdan — «15 daqiqa» deb qotirib yozilsa server
                sozlamasi o'zgargan kuni matn jimgina yolg'onga aylanardi. */}
            <span>
              Bog‘lanish kutilmoqda… Kod {Math.max(1, Math.round((code.expires_in_s || 900) / 60))} daqiqa
              amal qiladi va bir marta ishlaydi.
            </span>
          </div>
        </div>
      ) : (
        <Button onClick={startLink} loading={link.isPending}>
          <Send size={16} /> Telegram bilan bog‘lash
        </Button>
      )}
    </div>
  )
}
