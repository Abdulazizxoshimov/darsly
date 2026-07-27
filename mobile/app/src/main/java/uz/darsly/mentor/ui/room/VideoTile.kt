package uz.darsly.mentor.ui.room

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import io.livekit.android.renderer.TextureViewRenderer
import io.livekit.android.room.Room
import io.livekit.android.room.track.VideoTrack

/**
 * Bitta video plitkasi (M20).
 *
 * ## Render qilish hayot sikli — bu yerda xato qilish oson
 * `TextureViewRenderer` **GL resurslarini** egallaydi va u `release()` qilinmasa
 * xotira/GPU oqadi (bir necha o'quvchi qo'shilib-chiqib turganda tez sezilib qoladi).
 * Shuning uchun:
 *  · view **bir marta** yasaladi va eslab qolinadi (`remember`);
 *  · trek o'zgarsa — eskisidan renderer olib tashlanadi, yangisiga qo'shiladi;
 *  · kompozitsiyadan chiqganda `release()`.
 *
 * `AndroidView(update = …)` da `addRenderer` chaqirish **noto'g'ri** bo'lardi: u har
 * rekompozitsiyada ishlaydi va bitta trekka o'nlab renderer qo'shilib ketardi.
 *
 * ## Nega `TextureView`, `SurfaceView` emas
 * SDK ikkalasini ham beradi. `TextureViewRenderer` oddiy View ierarxiyasida ishlaydi —
 * ustiga boshqa element qo'yish (ism yorlig'i, mikrofon ikonkasi, PiP) muammosiz.
 * `SurfaceView` esa alohida qatlamda chiziladi va ustidagi elementlar bilan
 * to'qnashadi.
 */
@Composable
fun VideoTile(
    room: Room,
    track: VideoTrack?,
    modifier: Modifier = Modifier,
    label: String? = null,
    placeholder: String = "Kamera o'chiq",
) {
    Box(
        modifier
            .clip(RoundedCornerShape(14.dp))
            .background(MaterialTheme.colorScheme.surfaceVariant),
        contentAlignment = Alignment.Center,
    ) {
        if (track == null) {
            Text(
                placeholder,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        } else {
            val context = LocalContext.current
            val view = remember { TextureViewRenderer(context).also { room.initVideoRenderer(it) } }

            DisposableEffect(track, view) {
                track.addRenderer(view)
                onDispose { track.removeRenderer(view) }
            }
            DisposableEffect(view) {
                onDispose { view.release() }
            }

            AndroidView(factory = { view }, modifier = Modifier.fillMaxSize())
        }

        label?.let {
            Text(
                it,
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurface,
                modifier = Modifier
                    .align(Alignment.BottomStart)
                    .padding(8.dp)
                    .clip(RoundedCornerShape(percent = 50))
                    .background(MaterialTheme.colorScheme.scrim)
                    .padding(horizontal = 8.dp, vertical = 3.dp),
            )
        }
    }
}
