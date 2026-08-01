// O'lchov manzarasi (scene) — nashr qiluvchi VA obunachi TOMONIDA bir xil chiziladi.
//
// Nega deterministik: obunachi dekodlangan kadrni ASL kadr bilan solishtiradi
// (PSNR / keskinlik). Buning uchun u asl kadrni QAYTA chiza olishi kerak — demak
// manzara faqat `frameIndex` ga bog'liq bo'lishi shart, tasodif yo'q.
//
// Kadr ichiga MARKER yoziladi (yuqori chapdagi oq/qora kvadratchalar):
//   · kadr raqami   (12 bit)  → obunachi qaysi asl kadrni qayta chizishni biladi
//   · Date.now()%60000 (16 bit) → "shishadan-shishagacha" (glass-to-glass) kechikish
// Marker piksel-darajada emas, BLOK darajasida o'qiladi (32×32) — shuning uchun u
// 640×360 gacha kichraytirilgan va qattiq siqilgan oqimda ham omon qoladi.

export const MARKER_BLOCK = 32
export const MARKER_BLOCKS = 30 // 2 kalibrovka + 12 kadr raqami + 16 vaqt
export const MARKER_H = MARKER_BLOCK
/** PSNR faqat shu chiziqdan pastda hisoblanadi — marker sifatga aralashmasin. */
export const ROI_TOP = MARKER_BLOCK + 8

const CODE = [
  'func (u *UseCase) AdmitGuest(ctx context.Context, id uuid.UUID) error {',
  '    req, err := u.repo.TransitionFromPending(ctx, id, StatusAdmitted)',
  '    if err != nil {',
  '        return apperr.Wrap("waitingroom: transition", err)',
  '    }',
  '    tok, err := u.livekit.ParticipantToken(req.LessonID, req.GuestName)',
  '    if err != nil { return err }',
  '    // Token Redis\'da 5 daqiqa turadi: mehmon kech ulansa ham oladi.',
  '    if err := u.cache.Set(ctx, admitKey(id), tok, 5*time.Minute); err != nil {',
  '        u.log.Warn(ctx, "admit token cache", logger.SafeString("err", err.Error()))',
  '    }',
  '    u.hub.Send(req.ID.String(), websocket.AdmitMessage(tok))',
  '    return nil',
  '}',
  '',
  '// Qatlamlar: 1280x720@15 (yuqori) + 640x360@3 (past) — simulcast.',
  '// Past internetda o\'quvchi ekranni YO\'QOTMAYDI, faqat mayda ko\'radi.',
  'const layers = [ScreenSharePresets.h720fps15, ScreenSharePresets.h360fps3]',
]

const TABLE = [
  ['Servis', 'Host port', 'Konteyner', 'Holat'],
  ['PostgreSQL', '5442', '5432', 'healthy'],
  ['Redis', '6399', '6379', 'up'],
  ['MinIO', '9020/9021', '9000/9001', 'up'],
  ['RabbitMQ', '5682/15682', '5672/15672', 'up'],
  ['LiveKit', '7880/7881', 'host-net', 'up'],
]

/**
 * Bitta kadrni chizadi.
 * @param {CanvasRenderingContext2D} ctx
 * @param {number} w kenglik
 * @param {number} h balandlik
 * @param {number} frameIndex kadr raqami (0..4095, aylanadi)
 * @param {'scroll'|'static'} mode  scroll = uzluksiz harakat (eng og'ir holat),
 *                                  static = slayd 3 soniyada bir almashadi
 */
export function renderScene(ctx, w, h, frameIndex, mode = 'scroll') {
  const s = w / 1280 // masshtab: 1280 uchun mo'ljallangan, 1920'da ham to'g'ri ko'rinadi
  const scroll = mode === 'scroll' ? (frameIndex * 2) % 600 : Math.floor(frameIndex / 45) * 120
  const slide = Math.floor(frameIndex / 45) % 3

  ctx.fillStyle = '#ffffff'
  ctx.fillRect(0, 0, w, h)

  // Sarlavha paneli
  ctx.fillStyle = '#0f172a'
  ctx.fillRect(0, ROI_TOP, w, 56 * s)
  ctx.fillStyle = '#ffffff'
  ctx.font = `${Math.round(26 * s)}px sans-serif`
  ctx.textBaseline = 'top'
  ctx.fillText(`Jonly — media sifat sinovi · slayd ${slide + 1} · kadr ${frameIndex}`, 16 * s, ROI_TOP + 14 * s)

  // Chap ustun: MAYDA MONOSPACE KOD — asosiy o'qilishlik sinovi
  ctx.save()
  ctx.beginPath()
  ctx.rect(0, ROI_TOP + 60 * s, w * 0.62, h - ROI_TOP - 60 * s)
  ctx.clip()
  const lineH = 20 * s
  for (let i = 0; i < 60; i++) {
    const line = CODE[i % CODE.length]
    const y = ROI_TOP + 70 * s + i * lineH - scroll * s
    if (y < ROI_TOP + 50 * s || y > h) continue
    ctx.fillStyle = '#94a3b8'
    ctx.font = `${Math.round(13 * s)}px monospace`
    ctx.fillText(String(i + 1).padStart(3, ' '), 8 * s, y)
    ctx.fillStyle = line.trimStart().startsWith('//') ? '#16a34a' : '#0f172a'
    ctx.font = `${Math.round(14 * s)}px monospace`
    ctx.fillText(line, 46 * s, y)
  }
  ctx.restore()

  // O'ng ustun: jadval (ingichka chiziqlar — siqilishda birinchi yo'qoladi)
  const tx = w * 0.64
  let ty = ROI_TOP + 70 * s
  ctx.font = `${Math.round(15 * s)}px sans-serif`
  for (let r = 0; r < TABLE.length; r++) {
    ctx.fillStyle = r === 0 ? '#e2e8f0' : r % 2 ? '#f8fafc' : '#ffffff'
    ctx.fillRect(tx, ty, w * 0.34, 26 * s)
    ctx.strokeStyle = '#cbd5e1'
    ctx.lineWidth = 1
    ctx.strokeRect(tx + 0.5, ty + 0.5, w * 0.34, 26 * s)
    ctx.fillStyle = '#0f172a'
    const colw = (w * 0.34) / 4
    for (let c = 0; c < 4; c++) ctx.fillText(TABLE[r][c], tx + 6 * s + c * colw, ty + 5 * s)
    ty += 26 * s
  }

  // Ingichka chiziqli test-panjara — keskinlikning halol o'lchovi
  ty += 24 * s
  ctx.fillStyle = '#0f172a'
  ctx.font = `${Math.round(12 * s)}px sans-serif`
  ctx.fillText('1px panjara (keskinlik testi):', tx, ty)
  ty += 18 * s
  for (let i = 0; i < 60; i++) {
    ctx.fillStyle = i % 2 ? '#000000' : '#ffffff'
    ctx.fillRect(tx + i * 2 * s, ty, 2 * s, 40 * s)
  }
  ty += 52 * s
  // 8pt/10pt/12pt matn — "slayd matni o'qiladimi" savolining o'zi
  for (const px of [10, 12, 16, 20]) {
    ctx.fillStyle = '#0f172a'
    ctx.font = `${Math.round(px * s)}px sans-serif`
    ctx.fillText(`${px}px: Ustoz ekranidagi mayda matn o'qiladimi?`, tx, ty)
    ty += (px + 8) * s
  }

  drawMarker(ctx, frameIndex, Date.now() % 60000)
}

/** Marker bloklarini chizadi (kadr raqami + vaqt tamg'asi). */
export function drawMarker(ctx, frameIndex, tsMod) {
  const bits = []
  bits.push(1, 0) // kalibrovka: oq, qora
  for (let i = 11; i >= 0; i--) bits.push((frameIndex >> i) & 1)
  for (let i = 15; i >= 0; i--) bits.push((tsMod >> i) & 1)
  for (let i = 0; i < MARKER_BLOCKS; i++) {
    ctx.fillStyle = bits[i] ? '#ffffff' : '#000000'
    ctx.fillRect(i * MARKER_BLOCK, 0, MARKER_BLOCK, MARKER_BLOCK)
  }
  // Marker chegarasi — enkoder uni fonga qo'shib yubormasin
  ctx.fillStyle = '#808080'
  ctx.fillRect(MARKER_BLOCKS * MARKER_BLOCK, 0, MARKER_BLOCK, MARKER_BLOCK)
}

/**
 * Dekodlangan kadrdagi markerni o'qiydi.
 * @param {ImageData} img asl o'lchamga qayta kattalashtirilgan kadr
 * @returns {{frameIndex:number, tsMod:number}|null}
 */
export function readMarker(img) {
  const { data, width } = img
  const vals = []
  for (let i = 0; i < MARKER_BLOCKS; i++) {
    // blok markazining 40% i — chekkalardagi siqilish artefaktlari chetlab o'tiladi
    const x0 = Math.round(i * MARKER_BLOCK + MARKER_BLOCK * 0.3)
    const x1 = Math.round(i * MARKER_BLOCK + MARKER_BLOCK * 0.7)
    const y0 = Math.round(MARKER_BLOCK * 0.3)
    const y1 = Math.round(MARKER_BLOCK * 0.7)
    let sum = 0
    let n = 0
    for (let y = y0; y < y1; y++) {
      for (let x = x0; x < x1; x++) {
        const o = (y * width + x) * 4
        sum += 0.299 * data[o] + 0.587 * data[o + 1] + 0.114 * data[o + 2]
        n++
      }
    }
    vals.push(sum / n)
  }
  const hi = vals[0]
  const lo = vals[1]
  // Kalibrovka bloklari ajralmasa marker ishonchsiz — kadrni tashlab yuboramiz
  if (hi - lo < 60) return null
  const mid = (hi + lo) / 2
  let frameIndex = 0
  for (let i = 2; i < 14; i++) frameIndex = (frameIndex << 1) | (vals[i] > mid ? 1 : 0)
  let tsMod = 0
  for (let i = 14; i < 30; i++) tsMod = (tsMod << 1) | (vals[i] > mid ? 1 : 0)
  return { frameIndex, tsMod }
}

/** ROI (marker ostidagi mazmun) uchun luma massivi. */
function luma(img, w, h) {
  const out = new Float32Array(w * (h - ROI_TOP))
  const { data } = img
  let k = 0
  for (let y = ROI_TOP; y < h; y++) {
    for (let x = 0; x < w; x++) {
      const o = (y * w + x) * 4
      out[k++] = 0.299 * data[o] + 0.587 * data[o + 1] + 0.114 * data[o + 2]
    }
  }
  return out
}

/**
 * PSNR (dB) va keskinlik nisbati.
 *
 * · **PSNR** — umumiy sodiqlik. Matn uchun 30 dB dan past = ko'zga tashlanadigan buzilish.
 * · **sharp** — Tenengrad gradient energiyasining nisbati (dekod/asl). 1.0 = asl
 *   keskinlik; 0.5 = qirralarning yarmi yo'qolgan (matn "yuvilgan").
 *   PSNR yolg'iz kamlik qiladi: bir tekis xiralashuv PSNR'ni unchalik tushirmaydi,
 *   lekin MATNNI o'qib bo'lmas qiladi. Aynan shuni gradient energiyasi ushlaydi.
 */
export function compare(decoded, reference, w, h) {
  const a = luma(decoded, w, h)
  const b = luma(reference, w, h)
  const rows = h - ROI_TOP
  let se = 0
  for (let i = 0; i < a.length; i++) {
    const d = a[i] - b[i]
    se += d * d
  }
  const mse = se / a.length
  const psnr = mse === 0 ? 99 : 10 * Math.log10((255 * 255) / mse)

  const grad = (arr) => {
    let g = 0
    for (let y = 1; y < rows - 1; y++) {
      for (let x = 1; x < w - 1; x++) {
        const i = y * w + x
        const gx = arr[i + 1] - arr[i - 1]
        const gy = arr[i + w] - arr[i - w]
        g += gx * gx + gy * gy
      }
    }
    return g / ((rows - 2) * (w - 2))
  }
  const gb = grad(b)
  const sharp = gb === 0 ? 0 : grad(a) / gb
  return { psnr, sharp }
}
