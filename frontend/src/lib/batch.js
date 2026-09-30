// Ko'p so'rovni CHEKLANGAN parallellik va tezlik bilan bajarish.
//
// Yozuvlar sahifasi har dars uchun alohida so'rov yuboradi (backend'da
// "hamma yozuvlar" endpoint'i yo'q). 100 dars = 100 parallel so'rov, mentor
// limiti esa 30 so'rov/s (burst 60) — sahifa o'zi o'zini 429 ga urardi va
// jadvalning yarmi «yuklab bo'lmadi» bo'lib chiqardi.

/**
 * `items` ni `fn` bilan qayta ishlaydi. Natija massivi kirish tartibida.
 * Har element uchun `{ ok: true, value }` yoki `{ ok: false, error }` —
 * bitta yiqilgan so'rov qolganlarini to'xtatmaydi.
 *
 * @param {T[]} items
 * @param {(item: T, index: number) => Promise<V>} fn
 * @param {{ concurrency?: number, minIntervalMs?: number, sleep?: (ms:number)=>Promise<void> }} opts
 *   concurrency  — bir vaqtdagi maksimal so'rov;
 *   minIntervalMs — ketma-ket ikki so'rov BOSHLANISHI orasidagi eng kam vaqt
 *                   (butun to'plam uchun umumiy) — barqaror tezlik chegarasi.
 * @template T, V
 */
export async function mapLimited(items, fn, { concurrency = 4, minIntervalMs = 0, sleep = wait } = {}) {
  const results = new Array(items.length)
  let next = 0
  let lastStart = -Infinity

  async function worker() {
    for (;;) {
      const i = next++
      if (i >= items.length) return
      if (minIntervalMs > 0) {
        // Navbat SINXRON band qilinadi (await'dan OLDIN) — aks holda bir
        // vaqtda uyg'ongan ikki worker bir xil bo'sh oraliqni ko'rib birga
        // yuborardi va chegara qog'ozda qolardi.
        const now = Date.now()
        const slot = Math.max(now, lastStart + minIntervalMs)
        lastStart = slot
        if (slot > now) await sleep(slot - now)
      }
      try {
        results[i] = { ok: true, value: await fn(items[i], i) }
      } catch (error) {
        results[i] = { ok: false, error }
      }
    }
  }

  const n = Math.max(1, Math.min(concurrency, items.length))
  await Promise.all(Array.from({ length: n }, worker))
  return results
}

function wait(ms) {
  return new Promise((r) => setTimeout(r, ms))
}
