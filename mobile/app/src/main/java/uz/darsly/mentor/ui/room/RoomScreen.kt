package uz.darsly.mentor.ui.room

import android.Manifest
import android.app.Activity
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.media.projection.MediaProjectionManager
import android.net.Uri
import android.os.Build
import android.provider.Settings
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import uz.darsly.mentor.data.store.PrefsUiPrefs

/**
 * R0 spike xonasi. Bu ATAYLAB chiroyli emas — maqsad ekran ulashishning
 * native'da ishlashini tasdiqlash va diagnostikani ko'rsatish.
 *
 * TODO(R1 · M20/M43): haqiqiy sahna (VideoTrackView/gallery), telefon+planshet
 *   portret/landscape layout, suzuvchi boshqaruv paneli.
 */
@OptIn(ExperimentalLayoutApi::class) // FlowRow — boshqaruv tugmalari o'ralishi uchun
@Composable
fun RoomScreen(
    lessonId: String,
    onLeave: () -> Unit,
    vm: RoomViewModel = viewModel(),
) {
    val ctx = LocalContext.current
    val state by vm.state.collectAsStateWithLifecycle()

    // 1) Media ruxsatlari (kamera, mikrofon, Android 13+ bildirishnoma).
    val permissions = buildList {
        add(Manifest.permission.CAMERA)
        add(Manifest.permission.RECORD_AUDIO)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            add(Manifest.permission.POST_NOTIFICATIONS)
        }
    }.toTypedArray()

    fun granted(p: String) =
        ContextCompat.checkSelfPermission(ctx, p) == PackageManager.PERMISSION_GRANTED

    /**
     * B-2 TUZATISH: ilgari `permissionLauncher.launch()` va `vm.join()` KETMA-KET
     * chaqilardi — ruxsat dialogi hali ekranda turganda xonaga ulanish boshlanardi va
     * `LessonService` `FOREGROUND_SERVICE_MICROPHONE` `SecurityException` bilan o'lardi
     * (fon rejimi o'sha sessiyada kafolatlanmasdi). Endi ulanish FAQAT natija
     * kelgandan keyin boshlanadi.
     */
    val permissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions(),
    ) { result ->
        val mic = result[Manifest.permission.RECORD_AUDIO] ?: granted(Manifest.permission.RECORD_AUDIO)
        val cam = result[Manifest.permission.CAMERA] ?: granted(Manifest.permission.CAMERA)
        if (mic) {
            // Kamera rad etilgan bo'lsa ham dars o'tish mumkin (faqat ovoz + ekran).
            vm.join(lessonId, withCamera = cam)
        } else {
            vm.onMicrophoneDenied()
        }
    }

    // 2) ⭐ MediaProjection ruxsat oynasi (tizim dialogi).
    //    Natija OK bo'lsa → ViewModel FGS'ni mediaProjection tipi bilan qayta
    //    ishga tushiradi, KEYIN setScreenShareEnabled chaqiradi.
    val projectionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.StartActivityForResult(),
    ) { result ->
        val data = result.data
        if (result.resultCode == Activity.RESULT_OK && data != null) {
            vm.startScreenShare(data)
        }
        // Bekor qilish xato emas — jim o'tamiz (web Controls.jsx bilan bir xil xulq).
    }

    LaunchedEffect(Unit) {
        val missing = permissions.filterNot { granted(it) }
        if (missing.isEmpty()) {
            vm.join(lessonId, withCamera = granted(Manifest.permission.CAMERA))
        } else {
            permissionLauncher.launch(missing.toTypedArray())
        }
    }

    // 🟢L: TIZIM "ORQAGA" TUGMASI.
    //
    // Avval "Orqaga" shunchaki navigatsiya stekini qaytarardi: ekran yopilardi, lekin
    // `LessonSessionHolder` (LiveKit `Room`) va `LessonService` (foreground servis)
    // TIRIK qolardi — ustoz "chiqdim" deb o'ylardi, aslida xonada turardi va
    // MediaProjection ekranni yozishda davom etardi (maxfiylik).
    //
    // Endi "Orqaga" ham "Chiqish" bilan bir xil ishlaydi. Dars jonli bo'lsa avval
    // TASDIQ so'raladi: tasodifiy bosilgan "Orqaga" 90 daqiqalik darsni uzib
    // qo'ymasligi kerak.
    var confirmLeave by rememberSaveable { mutableStateOf(false) }

    // B-1: tizim dialogidan OLDIN ko'rsatiladigan tushuntirish holati.
    val uiPrefs = remember { PrefsUiPrefs.create(ctx) }
    var shareTipOpen by rememberSaveable { mutableStateOf(false) }
    var tipMuted by rememberSaveable { mutableStateOf(false) }

    /** Ekran ulashishni boshlash: kerak bo'lsa avval tushuntirish, keyin tizim dialogi. */
    fun requestScreenShare() {
        if (uiPrefs.screenShareTipEnabled) {
            shareTipOpen = true
        } else {
            val mgr = ctx.getSystemService(Context.MEDIA_PROJECTION_SERVICE) as MediaProjectionManager
            projectionLauncher.launch(mgr.createScreenCaptureIntent())
        }
    }

    BackHandler {
        if (state.lessonActive) {
            confirmLeave = true
        } else {
            vm.leave()
            onLeave()
        }
    }

    // ⭐ B-1: ANDROID 14+ DIALOGI SUKUT BO'YICHA "BITTA ILOVA" NI TANLAYDI.
    //
    // Bu jimgina ishlamaydigan holat: ustoz "Boshlash" ni bosadi, ulashish
    // BOSHLANADI, lekin u PDF yoki GeoGebra'ga o'tganda o'quvchi **hech narsa
    // ko'rmaydi** — faqat Darsly ekranini ko'rib turadi. Tizim dialogining
    // default'ini ilova o'zgartira olmaydi, shuning uchun yagona yechim —
    // oldindan tushuntirish. Ustoz "eslatmang" desa boshqa ko'rsatilmaydi.
    if (shareTipOpen) {
        AlertDialog(
            onDismissRequest = { shareTipOpen = false },
            title = { Text("\"Butun ekran\" ni tanlang") },
            text = {
                Column {
                    Text(
                        "Keyingi oynada Android \"Bitta ilova\" ni tanlab qo'yadi. " +
                            "Shu holda o'quvchilar faqat Darsly ekranini ko'radi — " +
                            "siz PDF yoki GeoGebra'ga o'tsangiz ular hech narsa ko'rmaydi.",
                    )
                    Spacer(Modifier.height(12.dp))
                    Text(
                        "Shuning uchun oynada \"Butun ekran\" (Entire screen) ni tanlab, " +
                            "so'ng \"Boshlash\" ni bosing.",
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    Spacer(Modifier.height(16.dp))
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Checkbox(checked = tipMuted, onCheckedChange = { tipMuted = it })
                        Spacer(Modifier.width(4.dp))
                        Text("Boshqa eslatilmasin", style = MaterialTheme.typography.bodyMedium)
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    if (tipMuted) uiPrefs.screenShareTipEnabled = false
                    shareTipOpen = false
                    val mgr = ctx.getSystemService(Context.MEDIA_PROJECTION_SERVICE)
                        as MediaProjectionManager
                    projectionLauncher.launch(mgr.createScreenCaptureIntent())
                }) { Text("Tushunarli, davom etish") }
            },
            dismissButton = {
                TextButton(onClick = { shareTipOpen = false }) { Text("Bekor qilish") }
            },
        )
    }

    if (confirmLeave) {
        AlertDialog(
            onDismissRequest = { confirmLeave = false },
            title = { Text("Darsdan chiqasizmi?") },
            text = {
                Text(
                    if (state.screenOn) {
                        "Ekran ulashish to'xtaydi va o'quvchilar sizni ko'rmay qoladi. " +
                            "Dars xonasi yopilmaydi — qaytib kirishingiz mumkin."
                    } else {
                        "Xonadan chiqasiz. Dars yakunlanmaydi — qaytib kirishingiz mumkin."
                    },
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    confirmLeave = false
                    vm.leave()
                    onLeave()
                }) { Text("Chiqish") }
            },
            dismissButton = {
                TextButton(onClick = { confirmLeave = false }) { Text("Darsda qolish") }
            },
        )
    }

    Column(
        Modifier
            .fillMaxSize()
            .padding(16.dp)
            .verticalScroll(rememberScrollState()),
    ) {
        Text("Xona", style = MaterialTheme.typography.headlineSmall)
        Spacer(Modifier.height(8.dp))

        if (state.connecting) {
            LinearProgressIndicator(Modifier.fillMaxWidth())
            Spacer(Modifier.height(8.dp))
        }

        Card(Modifier.fillMaxWidth()) {
            Column(Modifier.padding(12.dp)) {
                Row2("Holat", state.connState)
                Row2("Xona", state.roomName ?: "—")
                Row2("Identity", state.identity ?: "—")
                Row2("Ishtirokchi", state.participantCount.toString())
                Row2("Mikrofon", if (state.micOn) "yoniq" else "o'chiq")
                Row2(
                    "Kamera",
                    if (state.camOn) {
                        if (state.cameraFront) "yoniq (old)" else "yoniq (orqa)"
                    } else {
                        "o'chiq"
                    },
                )
                Row2("Ekran", if (state.screenOn) "ULASHILMOQDA" else "o'chiq")
                // B-4: bu qator YOLG'ON gapirmasligi kerak — mikrofon o'chirilgan bo'lsa
                // "yoniq" emas, sababi bilan ko'rsatiladi (ScreenAudioPolicy).
                Row2("Ekran audiosi", state.screenAudioLabel)
            }
        }

        // M16: aloqa indikatori. Yaxshi bo'lganda KO'RINMAYDI (Zoom xulqi) — faqat
        // qayta ulanish yoki sifat pasayganda chiqadi, aks holda bezovta qiladi.
        state.linkLabel?.let { label ->
            Spacer(Modifier.height(8.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                if (state.reconnecting) {
                    CircularProgressIndicator(Modifier.height(16.dp))
                    Spacer(Modifier.width(8.dp))
                }
                Text(
                    label,
                    style = MaterialTheme.typography.bodyMedium,
                    color = if (state.linkIsWarning) {
                        MaterialTheme.colorScheme.error
                    } else {
                        MaterialTheme.colorScheme.onSurfaceVariant
                    },
                )
            }
        }

        // B-6: dars tugadi — resurslar (MediaProjection, FGS) allaqachon bo'shatilgan.
        // Ustoz nima bo'lganini va endi nima qilishni bilishi kerak.
        state.endedMessage?.let { message ->
            Spacer(Modifier.height(12.dp))
            Card(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(12.dp)) {
                    Text(
                        "Dars tugadi",
                        style = MaterialTheme.typography.titleSmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                    Spacer(Modifier.height(4.dp))
                    Text(message, style = MaterialTheme.typography.bodySmall)
                    Spacer(Modifier.height(8.dp))
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Button(onClick = {
                            vm.retryJoin(lessonId, withCamera = granted(Manifest.permission.CAMERA))
                        }) { Text("Qayta boshlash") }
                        OutlinedButton(onClick = { vm.leave(); onLeave() }) { Text("Darslarga qaytish") }
                    }
                }
            }
        }

        state.error?.let {
            Spacer(Modifier.height(8.dp))
            Text(it, color = MaterialTheme.colorScheme.error)
            // Ulanish xatosidan keyin qayta urinish MUMKIN bo'lishi kerak
            // (avval qorovul bloklab qo'yardi — QA 🟡2).
            if (!state.connecting && state.connState != "connected") {
                Spacer(Modifier.height(8.dp))
                Button(onClick = {
                    vm.retryJoin(lessonId, withCamera = granted(Manifest.permission.CAMERA))
                }) { Text("Qayta urinish") }
            }
        }

        // B-2: mikrofonsiz dars o'tib bo'lmaydi — tushunarli xabar va yo'l ko'rsatish.
        if (state.micDenied) {
            Spacer(Modifier.height(12.dp))
            Card(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(12.dp)) {
                    Text(
                        "Mikrofonga ruxsat berilmadi",
                        style = MaterialTheme.typography.titleSmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                    Spacer(Modifier.height(4.dp))
                    Text(
                        "Mikrofonsiz darsni boshlab bo'lmaydi. \"Ruxsat so'rash\" tugmasini " +
                            "bosing yoki telefon sozlamalarida Darsly Mentor uchun mikrofonni yoqing.",
                        style = MaterialTheme.typography.bodySmall,
                    )
                    Spacer(Modifier.height(8.dp))
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Button(onClick = {
                            vm.clearPermissionError()
                            permissionLauncher.launch(permissions)
                        }) { Text("Ruxsat so'rash") }
                        OutlinedButton(onClick = {
                            ctx.startActivity(
                                Intent(
                                    Settings.ACTION_APPLICATION_DETAILS_SETTINGS,
                                    Uri.fromParts("package", ctx.packageName, null),
                                ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
                            )
                        }) { Text("Sozlamalar") }
                    }
                }
            }
        }

        Spacer(Modifier.height(16.dp))

        FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedButton(onClick = { vm.toggleMic() }) {
                Text(if (state.micOn) "Mikrofonni o'chirish" else "Mikrofon")
            }
            OutlinedButton(onClick = { vm.toggleCam() }) {
                Text(if (state.camOn) "Kamerani o'chirish" else "Kamera")
            }
            // M12: yozuv QAYERGA o'tishni aytadi, hozirgi holatni emas; kamera
            // o'chiq bo'lsa almashtirishning ma'nosi yo'q — tugma ham o'chiq.
            OutlinedButton(onClick = { vm.flipCamera() }, enabled = state.camOn) {
                Text(if (state.cameraFront) "Orqa kameraga" else "Old kameraga")
            }
        }

        Spacer(Modifier.height(12.dp))

        // ⭐ Spike'ning asosiy tugmasi.
        Button(
            onClick = {
                if (state.screenOn) vm.stopScreenShare() else requestScreenShare()
            },
            enabled = state.connState == "connected",
            modifier = Modifier.fillMaxWidth().height(56.dp),
            colors = if (state.screenOn) {
                ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error)
            } else {
                ButtonDefaults.buttonColors()
            },
        ) {
            Text(if (state.screenOn) "EKRAN ULASHISHNI TO'XTATISH" else "EKRANNI ULASHISH")
        }

        Spacer(Modifier.height(12.dp))
        OutlinedButton(
            onClick = { vm.leave(); onLeave() },
            modifier = Modifier.fillMaxWidth(),
        ) { Text("Chiqish") }

        Spacer(Modifier.height(20.dp))
        Text("Diagnostika", style = MaterialTheme.typography.titleSmall)
        Spacer(Modifier.height(4.dp))
        Card(Modifier.fillMaxWidth()) {
            Column(Modifier.padding(10.dp)) {
                if (state.log.isEmpty()) {
                    Text("—", style = MaterialTheme.typography.bodySmall)
                }
                state.log.forEach {
                    Text(
                        it,
                        style = MaterialTheme.typography.bodySmall,
                        fontFamily = FontFamily.Monospace,
                    )
                }
            }
        }
    }
}

@Composable
private fun Row2(label: String, value: String) {
    androidx.compose.foundation.layout.Row(Modifier.fillMaxWidth()) {
        Text(
            label,
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.weight(1f),
        )
        Text(value, style = MaterialTheme.typography.bodyMedium)
    }
}
