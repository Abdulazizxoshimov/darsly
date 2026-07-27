package uz.darsly.mentor.data.store

import android.content.SharedPreferences

/**
 * Oddiy `SharedPreferences` faqe'i (JVM unit test uchun).
 *
 * NEGA Robolectric emas: [PrefsTokenStore] mantiqini sinash uchun Android runtime
 * kerak emas — bizga faqat "kalit → qiymat" xulqi kerak. Bu test **ishlab
 * chiqarish klassining aynan o'zini** sinaydi, faqat shifrlash qatlamisiz
 * (shifrlash — `EncryptedSharedPreferences`, uni haqiqiy qurilma/emulyator
 * tekshiradi; §QA ro'yxatiga kiritilgan).
 */
class FakeSharedPreferences : SharedPreferences {

    private val map = LinkedHashMap<String, Any?>()

    /** Qayta ochilgan ilovani modellaydi: ayni "diskdagi" ma'lumot ustidan yangi obyekt. */
    fun reopen(): FakeSharedPreferences = FakeSharedPreferences().also { it.map.putAll(map) }

    override fun getAll(): MutableMap<String, *> = map
    override fun getString(key: String?, defValue: String?): String? =
        map[key] as? String ?: defValue

    override fun getStringSet(key: String?, defValues: MutableSet<String>?) =
        @Suppress("UNCHECKED_CAST") (map[key] as? MutableSet<String> ?: defValues)

    override fun getInt(key: String?, defValue: Int) = map[key] as? Int ?: defValue
    override fun getLong(key: String?, defValue: Long) = map[key] as? Long ?: defValue
    override fun getFloat(key: String?, defValue: Float) = map[key] as? Float ?: defValue
    override fun getBoolean(key: String?, defValue: Boolean) = map[key] as? Boolean ?: defValue
    override fun contains(key: String?) = map.containsKey(key)
    override fun edit(): SharedPreferences.Editor = FakeEditor()

    override fun registerOnSharedPreferenceChangeListener(
        listener: SharedPreferences.OnSharedPreferenceChangeListener?,
    ) = Unit

    override fun unregisterOnSharedPreferenceChangeListener(
        listener: SharedPreferences.OnSharedPreferenceChangeListener?,
    ) = Unit

    private inner class FakeEditor : SharedPreferences.Editor {
        private val pending = LinkedHashMap<String, Any?>()
        private val removed = mutableSetOf<String>()
        private var clearAll = false

        override fun putString(key: String, value: String?) = apply { pending[key] = value }
        override fun putStringSet(key: String, values: MutableSet<String>?) = apply { pending[key] = values }
        override fun putInt(key: String, value: Int) = apply { pending[key] = value }
        override fun putLong(key: String, value: Long) = apply { pending[key] = value }
        override fun putFloat(key: String, value: Float) = apply { pending[key] = value }
        override fun putBoolean(key: String, value: Boolean) = apply { pending[key] = value }
        override fun remove(key: String) = apply { removed += key }
        override fun clear() = apply { clearAll = true }

        override fun commit(): Boolean { write(); return true }
        override fun apply() = write()

        private fun write() = synchronized(map) {
            if (clearAll) map.clear()
            removed.forEach { map.remove(it) }
            map.putAll(pending)
        }
    }
}
