@file:OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)

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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material3.Icon
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.FiberManualRecord
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.TopAppBar
import androidx.compose.ui.text.style.TextOverflow
import uz.darsly.mentor.BuildConfig
import uz.darsly.mentor.data.livekit.LessonSessionHolder
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
@Composable
fun RoomScreen(
    lessonId: String,
    onLeave: () -> Unit,
    vm: RoomViewModel = viewModel(),
    // M27: kutish xonasi alohida ViewModel — u faqat so'rovlar bilan ishlaydi
    // va `RoomViewModel` (LiveKit, MediaProjection, foreground servis) bilan
    // aralashmaydi. Ikkalasini birlashtirish bu fayldagi eng murakkab sinfni
    // yana kattalashtirardi va WS mantiqini media mantig'i bilan chalkashtirardi.
    waitingVm: WaitingRoomViewModel = viewModel(),
) {
    val ctx = LocalContext.current
    val state by vm.state.collectAsStateWithLifecycle()
    val waiting by waitingVm.state.collectAsStateWithLifecycle()
    val waitingSnackbar = remember { SnackbarHostState() }

    LaunchedEffect(lessonId) { waitingVm.start(lessonId) }

    LaunchedEffect(waiting.notice) {
        waiting.notice?.let {
            waitingSnackbar.showSnackbar(it)
            waitingVm.noticeShown()
        }
    }

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
    // Panellar `rememberSaveable` EMAS: ekran burilganda ochiq qolishi shart emas,
    // va ModalBottomSheet holati baribir qayta yaratiladi.
    var showParticipants by remember { mutableStateOf(false) }
    var showChat by remember { mutableStateOf(false) }

    // B-1: tizim dialogidan OLDIN ko'rsatiladigan tushuntirish holati.
    val uiPrefs = remember { PrefsUiPrefs.create(ctx) }
    var shareTipOpen by rememberSaveable { mutableStateOf(false) }
    var tipMuted by rememberSaveable { mutableStateOf(false) }

    /**
     * Ekran ulashishni boshlash: kerak bo'lsa avval tushuntirish, keyin tizim dialogi.
     *
     * [skipTip] — uzilgan ulashishni TIKLASH uchun `true`: ustoz "Butun ekran"
     * tushuntirishini shu darsda allaqachon ko'rgan va hozir undan kutilayotgani
     * bitta bosish. Ikkinchi tushuntirish darsni yana bir necha soniyaga cho'zardi.
     */
    fun requestScreenShare(skipTip: Boolean = false) {
        if (uiPrefs.screenShareTipEnabled && !skipTip) {
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

    // ⭐ CHIQISH — IKKI XIL (Zoom "Leave" ↔ "End meeting for all" naqshi).
    //
    // Avval faqat "Chiqish" bor edi va u darsni YAKUNLAMASDI. Natijada mobil
    // ilovada darsni yakunlash yo'li UMUMAN yo'q edi: dars abadiy `live` bo'lib
    // qolardi, ro'yxatda "Jonli · Davom etish" bo'lib turardi va bosilganda
    // xona qaytadan ochilardi. Ustoz uchun bu "dars tugamayapti" degani.
    if (confirmLeave) {
        AlertDialog(
            onDismissRequest = { confirmLeave = false },
            title = { Text("Darsni yakunlaysizmi?") },
            text = {
                Column {
                    Text(
                        "\"Yakunlash\" — dars tugaydi, o'quvchilar chiqariladi va yozuv " +
                            "saqlanadi. Bu darsga qaytib kirib bo'lmaydi.",
                    )
                    Spacer(Modifier.height(12.dp))
                    Text(
                        if (state.screenOn) {
                            "\"Vaqtincha chiqish\" — ekran ulashish to'xtaydi, lekin dars " +
                                "davom etadi va qaytib kirishingiz mumkin."
                        } else {
                            "\"Vaqtincha chiqish\" — dars davom etadi, qaytib kirishingiz mumkin."
                        },
                        style = MaterialTheme.typography.bodyMedium,
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    confirmLeave = false
                    vm.endLesson(lessonId)
                    onLeave()
                }) { Text("Yakunlash") }
            },
            dismissButton = {
                Row {
                    TextButton(onClick = { confirmLeave = false }) { Text("Bekor qilish") }
                    TextButton(onClick = {
                        confirmLeave = false
                        vm.leave()
                        onLeave()
                    }) { Text("Vaqtincha chiqish") }
                }
            },
        )
    }

    // ─── YANGI MAKET (R-1…R-5) ────────────────────────────────────────────────
    // Avval bu ekran diagnostika paneli edi: UUID qatorlari, uzun matnli tugmalar
    // ro'yxati va monospace jurnal. Endi Zoom naqshi: sarlavha → sahna → boshqaruv.
    Scaffold(
        snackbarHost = { SnackbarHost(waitingSnackbar) },
        topBar = { RoomTopBar(state, onLeave = { if (state.lessonActive) confirmLeave = true else { vm.leave(); onLeave() } }) },
        bottomBar = {
            ControlBar(
                micOn = state.micOn,
                camOn = state.camOn,
                screenOn = state.screenOn,
                enabled = state.connState == "connected",
                onToggleMic = { vm.toggleMic() },
                onToggleCam = { vm.toggleCam() },
                onFlipCamera = { vm.flipCamera() },
                onToggleShare = { if (state.screenOn) vm.stopScreenShare() else requestScreenShare() },
                onLeave = { if (state.lessonActive) confirmLeave = true else { vm.leave(); onLeave() } },
                onOpenParticipants = { showParticipants = true },
                onOpenChat = {
                    showChat = true
                    vm.markChatRead()
                },
                participantCount = state.participantCount,
                handsCount = state.hands.size,
                unreadChat = state.unreadChat,
            )
        },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding)) {

            if (state.connecting) LinearProgressIndicator(Modifier.fillMaxWidth())

            // M27: kutish xonasi — sahnadan YUQORIDA. O'quvchi eshik ortida
            // turganda buni ustoz darhol ko'rishi kerak; pastda bo'lsa u
            // boshqaruv paneli ostida qolib ketardi.
            WaitingRoomPanel(
                state = waiting,
                onAdmit = waitingVm::admit,
                onReject = waitingVm::reject,
                onAdmitAll = waitingVm::admitAll,
            )

            // M16: aloqa indikatori — yaxshi bo'lganda ko'rinmaydi (Zoom xulqi).
            //
            // C-11: tarmoq almashganda ("Mobil internetga o'tildi") shu qatorda
            // ko'rsatiladi va aloqa yozuvidan USTUN turadi: ustoz uchun "nima
            // bo'ldi" savoliga javob "aloqa sifati qanday" dan muhimroq —
            // usiz u ilovani yopib qayta ochadi va dars haqiqatan uziladi.
            (state.networkNote ?: state.linkLabel)?.let { label ->
                Surface(color = MaterialTheme.colorScheme.errorContainer, modifier = Modifier.fillMaxWidth()) {
                    Row(
                        Modifier.padding(horizontal = 16.dp, vertical = 6.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        if (state.reconnecting) {
                            CircularProgressIndicator(
                                Modifier.size(14.dp),
                                strokeWidth = 2.dp,
                                color = MaterialTheme.colorScheme.onErrorContainer,
                            )
                            Spacer(Modifier.width(8.dp))
                        }
                        Text(
                            label,
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onErrorContainer,
                        )
                    }
                }
            }

            // B-2: mikrofonsiz dars o'tib bo'lmaydi.
            if (state.micDenied) {
                RoomNotice(
                    title = "Mikrofonga ruxsat berilmadi",
                    body = "Mikrofonsiz darsni boshlab bo'lmaydi. \"Ruxsat so'rash\" tugmasini bosing " +
                        "yoki telefon sozlamalarida Darsly Mentor uchun mikrofonni yoqing.",
                    primaryText = "Ruxsat so'rash",
                    onPrimary = {
                        vm.clearPermissionError()
                        permissionLauncher.launch(permissions)
                    },
                    secondaryText = "Sozlamalar",
                    onSecondary = {
                        ctx.startActivity(
                            Intent(
                                Settings.ACTION_APPLICATION_DETAILS_SETTINGS,
                                Uri.fromParts("package", ctx.packageName, null),
                            ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
                        )
                    },
                )
            }

            // C-11: qayta ulanish ekran ulashishni oldi — bir bosishlik tiklash.
            //
            // Android 14+ da rozilikni qayta so'ramasdan tiklab bo'lmaydi
            // (`ScreenSharePlan` ga qarang), shuning uchun bu karta AVTOMATIK
            // tiklashning o'rnini bosadi: ustoz "Davom ettirish" ni bosadi va
            // tizim oynasidan boshqa hech narsa qilmaydi.
            if (state.restoreShare) {
                RoomNotice(
                    title = "Ekran ulashish uzildi",
                    body = "Tarmoq almashgani uchun ulashish to'xtadi. Android har safar " +
                        "yangi ruxsat so'raydi — \"Davom ettirish\" ni bosing.",
                    primaryText = "Davom ettirish",
                    onPrimary = { requestScreenShare(skipTip = true) },
                    secondaryText = "Keyinroq",
                    onSecondary = { vm.dismissRestoreShare() },
                )
            }

            // B-6: dars tugadi — resurslar bo'shatilgan.
            state.endedMessage?.let { message ->
                RoomNotice(
                    title = "Dars tugadi",
                    body = message,
                    primaryText = "Qayta boshlash",
                    onPrimary = { vm.retryJoin(lessonId, withCamera = granted(Manifest.permission.CAMERA)) },
                    secondaryText = "Darslarga qaytish",
                    onSecondary = { vm.leave(); onLeave() },
                )
            }

            // Ulanish xatosi.
            state.error?.let { message ->
                if (state.endedMessage == null) {
                    RoomNotice(
                        title = "Ulanmadi",
                        body = message,
                        primaryText = "Qayta urinish",
                        onPrimary = { vm.retryJoin(lessonId, withCamera = granted(Manifest.permission.CAMERA)) },
                    )
                }
            }

            // ⭐ Asosiy sahna — o'quvchilar videosi yoki ulashish holati.
            RoomStage(
                room = LessonSessionHolder.session?.room,
                state = state,
                modifier = Modifier.weight(1f),
            )

            // R-4: diagnostika faqat DEBUG build'da va bosib ochiladigan qilib.
            if (BuildConfig.DEBUG) {
                DiagnosticsPanel(state.log)
            }
        }

        if (showParticipants) {
            ParticipantsSheet(
                state = state,
                onDismiss = { showParticipants = false },
                onMuteAll = vm::muteAll,
                onMute = vm::muteParticipant,
                onRemove = vm::removeParticipant,
                onAllowSpeak = vm::allowSpeak,
                onLowerHand = vm::lowerHand,
                onLowerAllHands = vm::lowerAllHands,
                onRefresh = vm::refreshRoster,
            )
        }

        if (showChat) {
            ChatSheet(
                state = state,
                onDismiss = { showChat = false },
                onSend = { vm.sendChat(it) },
            )
        }
    }
}

/**
 * Sarlavha (R-5): **dars nomi**, UUID emas.
 *
 * Avval bu yerda `lesson_cd16acaf-4d8b-…` va `Ideac246b01-…` qatorlari turardi va
 * ular maketni buzib, satr o'rtasidan uzilib ketardi (QA topilmasi B-7).
 */
@Composable
private fun RoomTopBar(state: RoomUiState, onLeave: () -> Unit) {
    TopAppBar(
        navigationIcon = {
            IconButton(onClick = onLeave) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Orqaga")
            }
        },
        title = {
            Column {
                Text(
                    state.lessonTitle ?: "Dars",
                    style = MaterialTheme.typography.titleMedium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    when {
                        state.connecting -> "ulanmoqda…"
                        state.connState == "connected" -> "${state.participantCount} ishtirokchi"
                        else -> RoomStatus.connLabel(state.connState)
                    },
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        },
        actions = {
            // Yozuv indikatori — darsning O'Z sozlamasidan (`is_recording_enabled`).
            // Yozib olish default yoniq va server uni avtomatik boshlaydi, lekin
            // ustoz o'chirgan bo'lsa indikator ham yonmasligi kerak: maxfiylik
            // masalasida interfeys yolg'on gapirmasligi shart.
            if (state.connState == "connected" && state.recordingEnabled) {
                Surface(
                    color = MaterialTheme.colorScheme.error,
                    shape = RoundedCornerShape(percent = 50),
                    modifier = Modifier.padding(end = 8.dp),
                ) {
                    Row(
                        Modifier.padding(horizontal = 10.dp, vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(
                            Icons.Default.FiberManualRecord,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.onError,
                            modifier = Modifier.size(10.dp),
                        )
                        Spacer(Modifier.width(4.dp))
                        Text(
                            "REC",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onError,
                        )
                    }
                }
            }
            if (state.screenOn) {
                Surface(
                    color = MaterialTheme.colorScheme.primary,
                    shape = RoundedCornerShape(percent = 50),
                    modifier = Modifier.padding(end = 12.dp),
                ) {
                    Text(
                        "EFIRDA",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onPrimary,
                        modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp),
                    )
                }
            }
        },
    )
}

/** Xona ekranidagi xabar kartasi — bir xil ko'rinish uchun yagona komponent. */
@Composable
private fun RoomNotice(
    title: String,
    body: String,
    primaryText: String,
    onPrimary: () -> Unit,
    secondaryText: String? = null,
    onSecondary: (() -> Unit)? = null,
) {
    Card(Modifier.fillMaxWidth().padding(12.dp)) {
        Column(Modifier.padding(14.dp)) {
            Text(title, style = MaterialTheme.typography.titleSmall, color = MaterialTheme.colorScheme.error)
            Spacer(Modifier.height(4.dp))
            Text(body, style = MaterialTheme.typography.bodySmall)
            Spacer(Modifier.height(10.dp))
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = onPrimary) { Text(primaryText) }
                if (secondaryText != null && onSecondary != null) {
                    OutlinedButton(onClick = onSecondary) { Text(secondaryText) }
                }
            }
        }
    }
}

/**
 * Diagnostika — R-4: faqat DEBUG build'da va **yig'ilgan holda**.
 *
 * QA uchun kerak, ustozga esa keraksiz: avval monospace jurnal xona ekranining
 * pastini doim egallab turardi.
 */
@Composable
private fun DiagnosticsPanel(log: List<String>) {
    var open by rememberSaveable { mutableStateOf(false) }
    Column(Modifier.fillMaxWidth().padding(horizontal = 12.dp)) {
        TextButton(onClick = { open = !open }) {
            Text(
                if (open) "Diagnostikani yashirish" else "Diagnostika (debug)",
                style = MaterialTheme.typography.labelSmall,
            )
        }
        if (open) {
            Card(Modifier.fillMaxWidth().height(140.dp)) {
                Column(Modifier.padding(10.dp).verticalScroll(rememberScrollState())) {
                    log.forEach {
                        Text(it, style = MaterialTheme.typography.bodySmall, fontFamily = FontFamily.Monospace)
                    }
                }
            }
        }
    }
}
