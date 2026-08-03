package uz.darsly.mentor.util

import android.app.Activity
import android.content.Context
import android.content.ContextWrapper

/**
 * Compose'dagi `LocalContext` ortidan Activity'ni topadi.
 *
 * NEGA SHUNCHAKI `as Activity` EMAS: `LocalContext.current` odatda Activity
 * bo'ladi, lekin har doim emas — dialog, `ContextThemeWrapper` yoki
 * `LocalContext` ni almashtirgan preview/test muhitida u o'ralgan kontekst
 * bo'lib chiqadi va to'g'ridan-to'g'ri cast `ClassCastException` beradi.
 *
 * `null` qaytishi normal holat: chaqiruvchi Activity talab qiladigan amalni
 * (masalan `moveTaskToBack`) shunchaki o'tkazib yuboradi.
 */
fun Context.findActivity(): Activity? {
    var current: Context? = this
    while (current is ContextWrapper) {
        if (current is Activity) return current
        current = current.baseContext
    }
    return null
}
