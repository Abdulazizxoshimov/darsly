import { memo, useEffect, useRef, useState } from 'react'
import {
  Pen,
  Eraser,
  Trash2,
  Download,
  FileText,
  ChevronLeft,
  ChevronRight,
  X,
  Hand,
  ArrowUpToLine,
} from 'lucide-react'
import * as pdfjsLib from 'pdfjs-dist'
import { toast } from '../lib/toast'

// pdfjs worker'ni Vite asset sifatida bundle qilamiz (CDN'siz, offline ishlaydi).
pdfjsLib.GlobalWorkerOptions.workerSrc = new URL(
  'pdfjs-dist/build/pdf.worker.min.mjs',
  import.meta.url,
).toString()

// PDF sahifasi qanchalik katta rasterlansa (px eni). Kattaroq = tiniqroq, lekin og'irroq chunk.
const PDF_RENDER_WIDTH = 1200
// JPEG sifati — ~80-200KB oralig'ida ushlab turadi (data-channel chunk'lar uchun).
const PDF_JPEG_QUALITY = 0.7

// Qalam ranglari (dizayn-tizim tokenlari mos kelmasa qalam uchun hardcode ruxsat).
const COLORS = [
  { k: 'qora', v: '#111827' },
  { k: "ko'k", v: '#2563eb' },
  { k: 'qizil', v: '#dc2626' },
  { k: 'yashil', v: '#16a34a' },
  { k: 'sariq', v: '#f59e0b' },
]
const WIDTHS = [
  { k: 'Nozik', v: 2 },
  { k: "O'rta", v: 4 },
  { k: "Yo'g'on", v: 8 },
]
const THROTTLE_MS = 55
// Follow-presenter: host view (scrollY) o'zgarishini ~100ms'da bir tarqatamiz.
const VIEW_BCAST_MS = 100

// World (jahon) qat'iy en — barcha qurilma shu enni to'liq sig'diradi.
// x ∈ [0, WORLD_W]; y ∈ [0, ∞) (cheksiz past). Gorizontal pan/zoom YO'Q.
const WORLD_W = 1000
// Chizishda pastki chekkaga yaqinlashganda avto-scroll qiladigan chegara (css px) va tezlik.
const AUTOSCROLL_MARGIN = 56
const AUTOSCROLL_RATE = 0.18

const clamp = (v, lo, hi) => Math.min(hi, Math.max(lo, v))

// Real-time oq doska — EN-FIT + VERTIKAL SCROLL modeli.
// World eni qat'iy WORLD_W; scale = canvasCssW / WORLD_W → EN DOIM to'liq sig'adi (telefon, planshet — bir xil enli kontent).
// View = faqat scrollY (canvas tepasidagi world-Y). Gorizontal siljish/zoom yo'q.
// World→screen(css): sx = x*scale ; sy = (y - scrollY)*scale.  screen→world: x = sx/scale ; y = sy/scale + scrollY.
// Stroke nuqtalari absolyut world piksel: [wx, wy, p] (wx ∈ 0..WORLD_W, wy cheksiz past).
// strokesRef — manba (LiveRoom boshqaradi). viewRef = { scrollY } (LiveRoom bilan bo'lishilgan).
// seq: yangi segmentlar inkremental chiziladi. clearSeq: to'liq qayta chizish.
// viewSeq: remote (yoki snapshot) scrollY keldi → to'liq qayta chizish (guest ergashadi).
// canDraw — faqat host chizadi/suradi/PDF boshqaradi; onViewChange({scrollY}) host scrollY'ini tarqatadi.
//
// FOLLOW-PRESENTER: en doim fit bo'lgani uchun faqat scrollY sinxronlanadi. Host {act:'view', scrollY} tarqatadi.
// Guest scrollY'ni oladi, o'z en-fit scale'i bilan render → ustoz ko'rgan aynan shu enli kontentni, shu
// vertikal pozitsiyada ko'radi. Hech narsa kichraymaydi (en to'liq). followRef — guest uchun oxirgi scrollY.
// `memo`: doska og'ir komponent (canvas + PDF). Xonadagi ishtirokchi holati,
// chat yoki reaksiya o'zgarganda uni qayta render qilishning ma'nosi yo'q —
// proplari faqat ref'lar, sonlar va (LiveRoom'da `useCallback` bilan) barqaror
// funksiyalar, ya'ni memo haqiqatan kesadi.
export const Whiteboard = memo(function Whiteboard({
  strokesRef,
  bgRef,
  viewRef,
  followRef,
  seq,
  clearSeq,
  viewSeq,
  canDraw,
  onHostDraw,
  onClear,
  onViewChange,
  onSetBackground,
  onClearBackground,
}) {
  const wrapRef = useRef(null)
  const canvasRef = useRef(null)
  const ctxRef = useRef(null)
  const sizeRef = useRef({ w: 0, h: 0 })
  const dprRef = useRef(1)
  const scaleRef = useRef(1) // joriy en-fit scale = cssW / WORLD_W
  const renderedRef = useRef(new Map()) // id -> chizilgan nuqtalar soni
  const lastBgImgRef = useRef(null) // oxirgi ko'rilgan fon rasm (host: yangi sahifada tepaga qaytarish uchun)

  // rAF bilan qayta chizishni birlashtiramiz (scroll/chizish paytida silliq).
  const rafRef = useRef(0)
  // View broadcast throttle holati.
  const lastBcastRef = useRef(0)
  const bcastTimerRef = useRef(0)
  const onViewRef = useRef(onViewChange)
  onViewRef.current = onViewChange

  // --- PDF fon (faqat host yuklaydi/sahifa almashtiradi) ---
  const pdfDocRef = useRef(null) // yuklangan pdfjs hujjati (faqat host xotirasida)
  const fileRef = useRef(null) // yashirin <input type=file>
  const [pdf, setPdf] = useState(null) // { page, total } yoki null
  const [pdfBusy, setPdfBusy] = useState(false)

  // Host chizish hodisalarini ref'da saqlaymiz — pointer effekti mid-stroke re-subscribe bo'lmasin.
  const onDrawRef = useRef(onHostDraw)
  onDrawRef.current = onHostDraw

  const [color, setColor] = useState(COLORS[0].v)
  const [width, setWidth] = useState(WIDTHS[1].v)
  const [eraser, setEraser] = useState(false)
  const [hand, setHand] = useState(false) // "Qo'l" asbobi (drag = vertikal scroll)
  const toolRef = useRef({ c: color, w: width, e: eraser })
  const handRef = useRef(hand)
  useEffect(() => {
    toolRef.current = { c: color, w: width, e: eraser }
  }, [color, width, eraser])
  useEffect(() => {
    handRef.current = hand
  }, [hand])

  // Joriy view'ni canvas device-transform sifatida o'rnatadi (DPR * scale; faqat vertikal scrollY translate).
  function setWorldTransform(ctx) {
    const dpr = dprRef.current
    const s = dpr * scaleRef.current
    const scrollY = viewRef.current.scrollY || 0
    ctx.setTransform(s, 0, 0, s, 0, -s * scrollY)
  }

  // Bitta stroke'ni `from` indeksidan boshlab chizadi (world koordinata; transform view'ni qo'llaydi).
  function drawStroke(ctx, s, from) {
    const pts = s.pts
    if (!pts || !pts.length) return
    const col = s.e ? '#fff' : s.c
    ctx.strokeStyle = col
    ctx.fillStyle = col
    ctx.lineCap = 'round'
    ctx.lineJoin = 'round'
    // Bir nuqtalik stroke — nuqta chiz.
    if (pts.length === 1 && from === 0) {
      const [wx, wy, p] = pts[0]
      const r = Math.max((s.w * (0.5 + (p ?? 0.5))) / 2, 0.6)
      ctx.beginPath()
      ctx.arc(wx, wy, r, 0, Math.PI * 2)
      ctx.fill()
      return
    }
    const start = Math.max(1, from)
    for (let i = start; i < pts.length; i++) {
      const a = pts[i - 1]
      const b = pts[i]
      const p = b[2] ?? 0.5
      ctx.lineWidth = s.w * (0.5 + p)
      ctx.beginPath()
      ctx.moveTo(a[0], a[1])
      ctx.lineTo(b[0], b[1])
      ctx.stroke()
    }
  }

  // PDF fon world (0,0)'da, EN = WORLD_W (to'liq en), balandligi aspect bo'yicha. Transform view'ni qo'llaydi.
  function drawBackground(ctx) {
    const img = bgRef?.current?.img
    if (!img || !img.width || !img.height) return
    const h = (WORLD_W * img.height) / img.width
    ctx.drawImage(img, 0, 0, WORLD_W, h)
  }

  // Butun doskani tozalab (oq fon → PDF fon → strokes) hamma stroke'ni qaytadan chizadi.
  function fullRedraw() {
    const ctx = ctxRef.current
    if (!ctx) return
    const dpr = dprRef.current
    const { w, h } = sizeRef.current
    // Oq fon — device (css) space'da butun canvasni qoplaymiz.
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    ctx.fillStyle = '#fff'
    ctx.fillRect(0, 0, w, h)
    // World transform — fon + strokes.
    setWorldTransform(ctx)
    drawBackground(ctx)
    renderedRef.current.clear()
    for (const s of strokesRef.current) {
      drawStroke(ctx, s, 0)
      renderedRef.current.set(s.id, s.pts.length)
    }
  }

  // rAF orqali bitta frame'da bitta qayta chizish.
  function scheduleRedraw() {
    if (rafRef.current) return
    rafRef.current = requestAnimationFrame(() => {
      rafRef.current = 0
      fullRedraw()
    })
  }

  // Host scrollY o'zgarishini throttle bilan tarqatadi (leading + trailing).
  function broadcastView() {
    if (!canDraw || !onViewRef.current) return
    const now = performance.now()
    const remaining = VIEW_BCAST_MS - (now - lastBcastRef.current)
    clearTimeout(bcastTimerRef.current)
    const send = () => {
      lastBcastRef.current = performance.now()
      onViewRef.current?.({ scrollY: viewRef.current.scrollY || 0 })
    }
    if (remaining <= 0) send()
    else bcastTimerRef.current = setTimeout(send, remaining)
  }

  // Guest: host'ning oxirgi scrollY'ini o'z view'iga qo'llaydi (scale o'z enidan avtomatik).
  function applyFollow() {
    const f = followRef?.current
    if (!f) return
    viewRef.current = { scrollY: Math.max(0, f.scrollY || 0) }
  }

  // scrollY'ni o'rnatadi (>= 0), qayta chizadi va (host bo'lsa) tarqatadi.
  function applyScroll(scrollY) {
    viewRef.current = { scrollY: Math.max(0, scrollY) }
    scheduleRedraw()
    broadcastView()
  }

  // Doskani eng tepaga qaytaradi (scrollY = 0).
  function scrollTop() {
    applyScroll(0)
  }

  // Canvas o'lchami + DPR + en-fit scale sozlash va resize kuzatuvchi.
  useEffect(() => {
    const cv = canvasRef.current
    const wrap = wrapRef.current
    if (!cv || !wrap) return
    const ctx = cv.getContext('2d')
    ctxRef.current = ctx

    const setup = () => {
      const w = wrap.clientWidth
      const h = wrap.clientHeight
      if (!w || !h) return
      const dpr = window.devicePixelRatio || 1
      dprRef.current = dpr
      cv.width = Math.round(w * dpr)
      cv.height = Math.round(h * dpr)
      cv.style.width = w + 'px'
      cv.style.height = h + 'px'
      sizeRef.current = { w, h }
      scaleRef.current = w / WORLD_W // EN DOIM to'liq sig'adi
      if (canDraw) {
        // Host: doska ochilganda va o'z ekrani o'lchami o'zgarganda joriy scrollY'ni tarqatadi.
        broadcastView()
      } else {
        // Guest: ekran o'lchami o'zgardi (resize/rotate) → host'ning scrollY'ini qayta qo'lla.
        applyFollow()
      }
      fullRedraw()
    }

    const ro = new ResizeObserver(setup)
    ro.observe(wrap)
    setup()
    return () => {
      ro.disconnect()
      if (rafRef.current) cancelAnimationFrame(rafRef.current)
      clearTimeout(bcastTimerRef.current)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // To'liq qayta chizish (clear / snapshot / yangi PDF fon). Host: yangi sahifa yuklansa tepaga qaytaradi.
  useEffect(() => {
    fullRedraw()
    const img = bgRef?.current?.img || null
    if (!img) {
      lastBgImgRef.current = null
    } else if (canDraw && img !== lastBgImgRef.current) {
      lastBgImgRef.current = img
      applyScroll(0) // yangi sahifa tepada — scrollY=0 ga qaytar + tarqat
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clearSeq])

  // Remote (yoki snapshot) scrollY keldi — guest ergashadi.
  useEffect(() => {
    if (!canDraw) applyFollow()
    fullRedraw()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [viewSeq])

  // Inkremental render — faqat yangi nuqtalarni chizadi.
  useEffect(() => {
    const ctx = ctxRef.current
    if (!ctx) return
    if (rafRef.current) return // to'liq qayta chizish kutilmoqda — u hammasini chizadi
    setWorldTransform(ctx)
    for (const s of strokesRef.current) {
      const done = renderedRef.current.get(s.id) || 0
      if (s.pts.length > done) {
        drawStroke(ctx, s, done)
        renderedRef.current.set(s.id, s.pts.length)
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [seq])

  // Host pointer: chizish (1 barmoq/S Pen) + vertikal scroll (2 barmoq / hand-tool / wheel).
  // Deps faqat [canDraw] — guest hech qanday pointer bilan ta'sir qilolmaydi (faqat ergashadi).
  useEffect(() => {
    if (!canDraw) return
    const cv = canvasRef.current
    if (!cv) return

    const pointers = new Map() // pointerId -> {x, y} (canvasga nisbatan css px)
    let scroll = null // {midY0, scrollY0} — 2-barmoq vertikal scroll
    let suppressDraw = false // 2+ pointer bo'lgach barcha ko'tarilguncha chizmaymiz
    let panning = null // hand-tool 1-barmoq scroll: {y, scrollY0}
    let drawing = false
    let curId = null
    let buf = []
    let lastFlush = 0

    const rel = (e) => {
      const r = cv.getBoundingClientRect()
      return { x: e.clientX - r.left, y: e.clientY - r.top }
    }
    // css nuqta → world nuqta.
    const toWorld = (sx, sy) => {
      const s = scaleRef.current
      const scrollY = viewRef.current.scrollY || 0
      return [sx / s, sy / s + scrollY]
    }

    const flush = () => {
      if (!buf.length) return
      const t = toolRef.current
      const batch = buf
      buf = []
      onDrawRef.current?.({ id: curId, c: t.c, w: t.w, e: t.e, pts: batch })
    }

    const startDraw = (e) => {
      drawing = true
      curId = crypto.randomUUID()
      const { x, y } = rel(e)
      const p = e.pressure > 0 ? e.pressure : 0.5
      const [wx, wy] = toWorld(x, y)
      buf = [[wx, wy, p]]
      lastFlush = performance.now()
      flush() // birinchi nuqta darhol (yangi stroke)
    }

    const midOf = () => {
      const pts = [...pointers.values()]
      const [a, b] = pts
      return (a.y + b.y) / 2
    }

    const startScroll = () => {
      if (pointers.size < 2) return
      scroll = { midY0: midOf(), scrollY0: viewRef.current.scrollY || 0 }
    }

    const updateScroll = () => {
      if (pointers.size < 2 || !scroll) return
      const dy = midOf() - scroll.midY0
      applyScroll(scroll.scrollY0 - dy / scaleRef.current)
    }

    // Chizishda pastki chekkaga yetganda avto-scroll (kontent pen tagida ochiladi).
    const maybeAutoScroll = (y) => {
      const { h } = sizeRef.current
      const over = y - (h - AUTOSCROLL_MARGIN)
      if (over > 0) applyScroll((viewRef.current.scrollY || 0) + (over * AUTOSCROLL_RATE) / scaleRef.current)
    }

    const down = (e) => {
      if (e.pointerType === 'mouse' && e.button !== 0) return
      const { x, y } = rel(e)
      pointers.set(e.pointerId, { x, y })
      try {
        cv.setPointerCapture(e.pointerId)
      } catch {
        /* ignore */
      }

      if (pointers.size >= 2) {
        // 2+ barmoq → vertikal scroll. Boshlangan chiziqni yakunlaymiz.
        suppressDraw = true
        if (drawing) {
          flush()
          drawing = false
          curId = null
        }
        panning = null
        startScroll()
        e.preventDefault()
        return
      }

      if (suppressDraw) {
        e.preventDefault()
        return
      }

      if (handRef.current) {
        panning = { y, scrollY0: viewRef.current.scrollY || 0 } // hand-tool 1-barmoq scroll
      } else {
        startDraw(e)
      }
      e.preventDefault()
    }

    const move = (e) => {
      const p = pointers.get(e.pointerId)
      if (!p) return
      const { x, y } = rel(e)
      p.x = x
      p.y = y

      if (scroll) {
        updateScroll()
        e.preventDefault()
        return
      }

      if (panning) {
        applyScroll(panning.scrollY0 - (y - panning.y) / scaleRef.current)
        e.preventDefault()
        return
      }

      if (!drawing) return
      const evs = e.getCoalescedEvents ? e.getCoalescedEvents() : null
      if (evs && evs.length) {
        for (const ev of evs) {
          const r = rel(ev)
          const pr = ev.pressure > 0 ? ev.pressure : 0.5
          const [wx, wy] = toWorld(r.x, r.y)
          buf.push([wx, wy, pr])
        }
      } else {
        const pr = e.pressure > 0 ? e.pressure : 0.5
        const [wx, wy] = toWorld(x, y)
        buf.push([wx, wy, pr])
      }
      const now = performance.now()
      if (now - lastFlush >= THROTTLE_MS) {
        lastFlush = now
        flush()
      }
      maybeAutoScroll(y)
      e.preventDefault()
    }

    const finishPointer = (e) => {
      const had = pointers.delete(e.pointerId)
      try {
        cv.releasePointerCapture(e.pointerId)
      } catch {
        /* ignore */
      }
      if (!had) return

      if (scroll) {
        if (pointers.size < 2) scroll = null
        e.preventDefault?.()
      } else if (panning) {
        panning = null
      } else if (drawing) {
        const { x, y } = rel(e)
        const pr = e.pressure > 0 ? e.pressure : 0.5
        const [wx, wy] = toWorld(x, y)
        buf.push([wx, wy, pr])
        flush()
        drawing = false
        curId = null
      }
      // Barcha barmoq ko'tarilgach chizishni yana ochamiz.
      if (pointers.size === 0) {
        suppressDraw = false
        scroll = null
      }
    }

    const cancel = () => {
      pointers.clear()
      scroll = null
      panning = null
      suppressDraw = false
      if (drawing) {
        flush()
        drawing = false
        curId = null
      }
    }

    // Desktop: wheel = vertikal scroll (faqat scrollY).
    const wheel = (e) => {
      e.preventDefault()
      applyScroll((viewRef.current.scrollY || 0) + e.deltaY / scaleRef.current)
    }

    cv.addEventListener('pointerdown', down)
    cv.addEventListener('pointermove', move)
    cv.addEventListener('pointerup', finishPointer)
    cv.addEventListener('pointercancel', cancel)
    cv.addEventListener('pointerleave', finishPointer)
    cv.addEventListener('wheel', wheel, { passive: false })
    return () => {
      cv.removeEventListener('pointerdown', down)
      cv.removeEventListener('pointermove', move)
      cv.removeEventListener('pointerup', finishPointer)
      cv.removeEventListener('pointercancel', cancel)
      cv.removeEventListener('pointerleave', finishPointer)
      cv.removeEventListener('wheel', wheel)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [canDraw])

  // Joriy ko'rinishni PNG rasm sifatida saqlaydi (host ham, o'quvchi ham).
  function saveImage() {
    const cv = canvasRef.current
    if (!cv) return
    try {
      const url = cv.toDataURL('image/png')
      const stamp = new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')
      const a = document.createElement('a')
      a.href = url
      a.download = `doska-${stamp}.png`
      document.body.appendChild(a)
      a.click()
      a.remove()
      toast.success('Doska rasm sifatida saqlandi')
    } catch {
      toast.error('Rasmni saqlab bo‘lmadi')
    }
  }

  // Hujjatni yopilganda pdfjs resurslarini bo'shatadi.
  useEffect(() => {
    return () => {
      try {
        pdfDocRef.current?.destroy?.()
      } catch {
        /* ignore */
      }
      pdfDocRef.current = null
    }
  }, [])

  // Berilgan sahifani rasterlab (~1200px) JPEG'ga aylantiradi va yuqoriga (LiveRoom) uzatadi.
  async function renderPdfPage(n, doc = pdfDocRef.current) {
    if (!doc) return
    setPdfBusy(true)
    try {
      const page = await doc.getPage(n)
      const base = page.getViewport({ scale: 1 })
      const scale = PDF_RENDER_WIDTH / base.width
      const vp = page.getViewport({ scale })
      const off = document.createElement('canvas')
      off.width = Math.round(vp.width)
      off.height = Math.round(vp.height)
      const octx = off.getContext('2d')
      octx.fillStyle = '#fff'
      octx.fillRect(0, 0, off.width, off.height)
      await page.render({ canvasContext: octx, viewport: vp }).promise
      const dataURL = off.toDataURL('image/jpeg', PDF_JPEG_QUALITY)
      setPdf({ page: n, total: doc.numPages })
      onSetBackground?.(dataURL, n, doc.numPages)
    } catch {
      toast.error('Sahifani ochib bo‘lmadi')
    } finally {
      setPdfBusy(false)
    }
  }

  // Host PDF fayl tanlaganda: hujjatni och → 1-sahifani fon qil.
  async function onPickPdf(e) {
    const file = e.target.files?.[0]
    e.target.value = '' // bir xil faylni qayta tanlash mumkin bo'lsin
    if (!file) return
    setPdfBusy(true)
    try {
      const buf = await file.arrayBuffer()
      const doc = await pdfjsLib.getDocument({ data: buf }).promise
      try {
        pdfDocRef.current?.destroy?.()
      } catch {
        /* ignore */
      }
      pdfDocRef.current = doc
      await renderPdfPage(1, doc)
      toast.success('PDF doskaga qo‘yildi')
    } catch {
      toast.error('PDF ochib bo‘lmadi')
      setPdfBusy(false)
    }
  }

  function gotoPage(n) {
    if (!pdf || pdfBusy) return
    if (n < 1 || n > pdf.total) return
    renderPdfPage(n)
  }

  function clearPdf() {
    try {
      pdfDocRef.current?.destroy?.()
    } catch {
      /* ignore */
    }
    pdfDocRef.current = null
    setPdf(null)
    onClearBackground?.()
  }

  const cursor = canDraw ? (hand ? 'grab' : 'crosshair') : 'default'

  return (
    <div className="whiteboard" ref={wrapRef}>
      <canvas ref={canvasRef} className="whiteboard__canvas" style={{ cursor }} />
      {canDraw && (
        <button type="button" title="Joriy ko'rinishni rasm sifatida saqlash" className="wb-save" onClick={saveImage}>
          <Download size={18} />
        </button>
      )}
      {canDraw && (
        <div className="wb-toolbar">
          <div className="wb-group">
            <div className="wb-colors">
              {COLORS.map((c) => (
                <button
                  key={c.v}
                  type="button"
                  title={c.k}
                  className={'wb-swatch' + (!eraser && color === c.v ? ' active' : '')}
                  style={{ background: c.v }}
                  onClick={() => {
                    setColor(c.v)
                    setEraser(false)
                    setHand(false)
                  }}
                />
              ))}
            </div>
            <div className="wb-widths">
              {WIDTHS.map((w) => (
                <button
                  key={w.v}
                  type="button"
                  title={w.k}
                  className={'wb-width' + (width === w.v ? ' active' : '')}
                  onClick={() => setWidth(w.v)}
                >
                  <span style={{ width: w.v + 4, height: w.v + 4 }} />
                </button>
              ))}
            </div>
          </div>

          <div className="wb-sep" />

          <div className="wb-group">
            <button
              type="button"
              title="Qalam"
              className={'wb-tool' + (!eraser && !hand ? ' active' : '')}
              onClick={() => {
                setEraser(false)
                setHand(false)
              }}
            >
              <Pen size={18} />
            </button>
            <button
              type="button"
              title="O'chirg'ich"
              className={'wb-tool' + (eraser && !hand ? ' active' : '')}
              onClick={() => {
                setEraser(true)
                setHand(false)
              }}
            >
              <Eraser size={18} />
            </button>
            <button type="button" title="Tozalash" className="wb-tool wb-tool--danger" onClick={onClear}>
              <Trash2 size={18} />
            </button>
          </div>

          <div className="wb-sep" />

          <div className="wb-group">
            <button
              type="button"
              title="Qo'l (vertikal surish)"
              className={'wb-tool' + (hand ? ' active' : '')}
              onClick={() => setHand((h) => !h)}
            >
              <Hand size={18} />
            </button>
            <button type="button" title="Tepaga" className="wb-tool" onClick={scrollTop}>
              <ArrowUpToLine size={18} />
            </button>
          </div>

          <div className="wb-sep" />

          <div className="wb-group">
            <input
              ref={fileRef}
              type="file"
              accept=".pdf,application/pdf"
              style={{ display: 'none' }}
              onChange={onPickPdf}
            />
            <button
              type="button"
              title="PDF yuklash"
              className={'wb-tool wb-tool--labeled' + (pdf ? ' active' : '')}
              onClick={() => fileRef.current?.click()}
              disabled={pdfBusy}
            >
              <FileText size={18} />
              <span>PDF</span>
            </button>
            {pdf && (
              <div className="wb-page">
                <button
                  type="button"
                  title="Oldingi sahifa"
                  className="wb-tool"
                  onClick={() => gotoPage(pdf.page - 1)}
                  disabled={pdfBusy || pdf.page <= 1}
                >
                  <ChevronLeft size={18} />
                </button>
                <span className="wb-page__label">
                  {pdf.page}/{pdf.total}
                </span>
                <button
                  type="button"
                  title="Keyingi sahifa"
                  className="wb-tool"
                  onClick={() => gotoPage(pdf.page + 1)}
                  disabled={pdfBusy || pdf.page >= pdf.total}
                >
                  <ChevronRight size={18} />
                </button>
                <button
                  type="button"
                  title="PDF'ni olib tashlash"
                  className="wb-tool wb-tool--danger"
                  onClick={clearPdf}
                  disabled={pdfBusy}
                >
                  <X size={18} />
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
})
