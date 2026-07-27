package uz.darsly.mentor.util

/**
 * Semver solishtirish (M42).
 *
 * NEGA ALOHIDA KOD: `"1.10.0" < "1.9.0"` — satr sifatida solishtirsak NOTO'G'RI
 * natija chiqadi va majburiy yangilanish butun ustozlar guruhini noto'g'ri bloklaydi.
 *
 * Qoidalar:
 *  · `v` prefiksi va bo'shliqlar tashlanadi (`" v1.2.3 "` → `1.2.3`).
 *  · Raqamli qismlar 3 tagacha to'ldiriladi: `"1.0"` == `"1.0.0"`.
 *  · Build metadata (`+abc`) e'tiborga olinmaydi (semver talabi).
 *  · Pre-release (`-spike`, `-rc1`) o'sha raqamli versiyadan **past**:
 *    `1.0.0-spike < 1.0.0`. Shu sabab spike build'i `min_version=1.0.0` da bloklanadi.
 *  · Buzuq yoki bo'sh qiymat → [compare] `null` qaytaradi. Chaqiruvchi buni
 *    "solishtirib bo'lmadi" deb ko'radi va **fail-open** qiladi (ilova ishlaydi).
 */
object Semver {

    private class Parsed(val nums: List<Int>, val pre: String?)

    private fun parse(raw: String?): Parsed? {
        val v = raw?.trim()?.removePrefix("v")?.removePrefix("V") ?: return null
        if (v.isEmpty()) return null

        // Build metadata semver'da solishtirishga ta'sir qilmaydi.
        val noBuild = v.substringBefore('+')
        val core = noBuild.substringBefore('-')
        val pre = noBuild.substringAfter('-', "").takeIf { it.isNotEmpty() }

        val parts = core.split('.')
        if (parts.isEmpty() || parts.size > 4) return null

        val nums = ArrayList<Int>(3)
        for (p in parts) {
            if (p.isEmpty() || p.length > 9 || p.any { !it.isDigit() }) return null
            nums += p.toInt()
        }
        while (nums.size < 3) nums += 0
        return Parsed(nums, pre)
    }

    /**
     * `a` < `b` → manfiy, teng → 0, `a` > `b` → musbat.
     * Qiymatlardan biri buzuq bo'lsa — `null`.
     */
    fun compare(a: String?, b: String?): Int? {
        val pa = parse(a) ?: return null
        val pb = parse(b) ?: return null

        for (i in 0 until maxOf(pa.nums.size, pb.nums.size)) {
            val x = pa.nums.getOrElse(i) { 0 }
            val y = pb.nums.getOrElse(i) { 0 }
            if (x != y) return x.compareTo(y)
        }

        // Raqamlar teng: pre-release relizdan past.
        return when {
            pa.pre == null && pb.pre == null -> 0
            pa.pre == null -> 1
            pb.pre == null -> -1
            else -> pa.pre.compareTo(pb.pre)
        }
    }

    /** `current` `required`dan eskimi? Solishtirib bo'lmasa `false` (fail-open). */
    fun isOlder(current: String?, required: String?): Boolean =
        (compare(current, required) ?: 0) < 0
}
