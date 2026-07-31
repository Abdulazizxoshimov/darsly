@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.lessons

import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.CloudOff
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Share
import androidx.compose.material.icons.filled.VideoLibrary
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.hilt.navigation.compose.hiltViewModel
import uz.darsly.mentor.BuildConfig
import uz.darsly.mentor.data.api.Lesson
import uz.darsly.mentor.ui.theme.DarslyTheme
import uz.darsly.mentor.ui.theme.PulseMark
import uz.darsly.mentor.ui.theme.liveCardGlow
import uz.darsly.mentor.ui.theme.neonGlow
import uz.darsly.mentor.util.LessonFormat
import uz.darsly.mentor.util.Share

/**
 * Darslar ro'yxati (M6 · M7 · M8).
 *
 * Holatlar (mezon B-3): yuklanish · xato · bo'sh · ro'yxat — to'rttasi ham alohida.
 * Ro'yxat keshdan kelgan bo'lsa va tarmoq yo'q bo'lsa yuqorida offline chizig'i turadi.
 */
@Composable
fun LessonsScreen(
    onOpenLesson: (Lesson) -> Unit,
    onOpenRecordings: (Lesson) -> Unit,
    vm: LessonsViewModel = hiltViewModel(),
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val context = LocalContext.current
    val snackbar = remember { SnackbarHostState() }
    var createOpen by rememberSaveable { mutableStateOf(false) }
    // Tahrirlanayotgan darsning ID'si — `Lesson` obyektining o'zi emas.
    // Sabab: ro'yxat yangilanganda (pull-to-refresh yoki WS) saqlangan obyekt
    // eskirib qolardi va forma eski qiymatlarni ko'rsatardi. ID bo'yicha har
    // kompozitsiyada joriy nusxa topiladi.
    var editingId by rememberSaveable { mutableStateOf<String?>(null) }
    val editing = editingId?.let { id -> state.lessons.firstOrNull { it.id == id } }

    LaunchedEffect(Unit) { vm.start() }

    LaunchedEffect(state.notice) {
        state.notice?.let {
            snackbar.showSnackbar(it)
            vm.noticeShown()
        }
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbar) },
        // "Chiqish" bu yerdan OLIB TASHLANDI — u endi shaxsiy kabinetda
        // (`ProfileScreen`). Zoom'da ham chiqish sozlamalar ichida: asosiy
        // ekrandagi doimiy "Chiqish" tugmasi tasodifan bosiladigan va hech qachon
        // kerak bo'lmaydigan tugma edi.
        topBar = {
            TopAppBar(title = {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    // Brend har ekranda: puls belgisi sarlavha yonida (B tili).
                    PulseMark(size = 22.dp)
                    Spacer(Modifier.width(10.dp))
                    Text("Darslar")
                }
            })
        },
        floatingActionButton = {
            // Asosiy harakat — glow'ga ruxsat berilgan uch joydan biri.
            ExtendedFloatingActionButton(
                onClick = { createOpen = true },
                icon = { Icon(Icons.Default.Add, contentDescription = null) },
                text = { Text("Dars yaratish") },
                containerColor = MaterialTheme.colorScheme.primary,
                contentColor = MaterialTheme.colorScheme.onPrimary,
                modifier = Modifier.neonGlow(
                    color = DarslyTheme.colors.neon,
                    shape = MaterialTheme.shapes.large,
                ),
            )
        },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding)) {
            if (state.offline) OfflineBar(state.cachedAtMillis)

            PullToRefreshBox(
                isRefreshing = state.refreshing,
                onRefresh = { vm.refresh() },
                modifier = Modifier.fillMaxSize(),
            ) {
                when {
                    state.loading && state.lessons.isEmpty() -> LoadingState()
                    state.error != null && state.lessons.isEmpty() ->
                        ErrorState(state.error!!) { vm.refresh() }
                    state.lessons.isEmpty() -> EmptyState { createOpen = true }
                    else -> LessonList(
                        lessons = state.lessons,
                        onOpen = onOpenLesson,
                        onEdit = { lesson -> editingId = lesson.id },
                        onRecordings = onOpenRecordings,
                        onShare = { lesson ->
                            Share.sendText(
                                context,
                                LessonFormat.shareText(lesson, BuildConfig.WEB_BASE_URL),
                            )
                        },
                        onCopy = { lesson ->
                            val url = LessonFormat.joinUrl(BuildConfig.WEB_BASE_URL, lesson.joinSlug)
                            if (url == null) {
                                vm.showNotice("Bu darsning havolasi yo'q")
                            } else if (Share.copyToClipboard(context, url)) {
                                vm.showNotice("Havola nusxalandi")
                            }
                        },
                    )
                }
            }
        }
    }

    if (createOpen) {
        CreateLessonDialog(
            onDismiss = { createOpen = false },
            onCreated = { lesson ->
                createOpen = false
                vm.onLessonCreated(lesson)
                // Tezkor dars: yaratildi → DARHOL xonaga (Zoom "New meeting" oqimi).
                // `onOpenLesson` holatga qarab yo'naltiradi; yangi dars uchun bu
                // har doim xona (`LessonActions.primary(scheduled) == START`).
                onOpenLesson(lesson)
            },
        )
    }

    editing?.let { lesson ->
        EditLessonDialog(
            lesson = lesson,
            onDismiss = { editingId = null },
            onSaved = {
                editingId = null
                vm.onLessonUpdated(it)
            },
            onDeleted = {
                editingId = null
                vm.onLessonDeleted(it)
            },
        )
    }
}

/** B-4: keshdagi ro'yxat ko'rsatilyapti — ustoz buni ko'rib turishi kerak. */
@Composable
private fun OfflineBar(cachedAtMillis: Long) {
    Surface(color = MaterialTheme.colorScheme.errorContainer, modifier = Modifier.fillMaxWidth()) {
        Row(
            Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Icon(
                Icons.Default.CloudOff,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onErrorContainer,
                modifier = Modifier.size(18.dp),
            )
            Column {
                Text(
                    "Internet yo'q — saqlangan ro'yxat",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onErrorContainer,
                )
                if (cachedAtMillis > 0) {
                    LessonFormat.scheduleLabel(LessonForm.rfc3339Utc(cachedAtMillis))?.let {
                        Text(
                            "Oxirgi yangilanish: $it",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onErrorContainer,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun LoadingState() {
    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        CircularProgressIndicator()
    }
}

@Composable
private fun ErrorState(message: String, onRetry: () -> Unit) {
    // Scroll qilinadigan konteyner: pull-to-refresh xato ekranida ham ishlasin.
    LazyColumn(Modifier.fillMaxSize()) {
        item {
            Column(
                Modifier.fillParentMaxSize().padding(24.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(message, color = MaterialTheme.colorScheme.error)
                Spacer(Modifier.height(12.dp))
                Button(onClick = onRetry) { Text("Qayta urinish") }
            }
        }
    }
}

@Composable
private fun EmptyState(onCreate: () -> Unit) {
    LazyColumn(Modifier.fillMaxSize()) {
        item {
            Column(
                Modifier.fillParentMaxSize().padding(24.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text("Hali dars yo'q", style = MaterialTheme.typography.titleMedium)
                Spacer(Modifier.height(4.dp))
                Text(
                    "Birinchi darsni shu yerda yarating va havolasini o'quvchilarga yuboring",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Spacer(Modifier.height(16.dp))
                Button(onClick = onCreate) { Text("Dars yaratish") }
            }
        }
    }
}

@Composable
private fun LessonList(
    lessons: List<Lesson>,
    onOpen: (Lesson) -> Unit,
    onEdit: (Lesson) -> Unit,
    onRecordings: (Lesson) -> Unit,
    onShare: (Lesson) -> Unit,
    onCopy: (Lesson) -> Unit,
) {
    LazyColumn(
        Modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        // Bo'limlar tartibi `LessonFormat.sortForDisplay` bilan bir xil bo'lgani
        // uchun ro'yxatni qayta tartiblamaymiz — faqat sarlavha qo'yamiz.
        LessonFormat.Section.entries.forEach { section ->
            val group = lessons.filter { LessonFormat.sectionOf(it) == section }
            if (group.isEmpty()) return@forEach
            item(key = "section-$section") {
                Text(
                    LessonFormat.sectionTitle(section),
                    style = MaterialTheme.typography.titleSmall,
                    color = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.padding(top = 4.dp),
                )
            }
            items(group, key = { it.id }) { lesson ->
                LessonCard(
                    lesson = lesson,
                    onClick = { onOpen(lesson) },
                    onEdit = { onEdit(lesson) },
                    onRecordings = { onRecordings(lesson) },
                    onShare = { onShare(lesson) },
                    onCopy = { onCopy(lesson) },
                )
            }
        }
    }
}

/**
 * Dars holati nishoni — web'dagi `.badge--live` / `.badge--scheduled` / `.badge--ended`
 * ning aynan ko'chirmasi (`frontend/src/styles.css:219-237`).
 *
 * Web'da holat **rangli nishon** bilan ko'rsatiladi: to'liq dumaloq (`--r-full`),
 * 0.15 shaffoflikdagi fon, qalin kichik matn. Avval mobilda bu oddiy matn edi —
 * shuning uchun ikkala platforma boshqa mahsulotdek ko'rinardi.
 */
@Composable
private fun StatusBadge(status: String) {
    val c = DarslyTheme.colors
    if (status == "live") {
        // B tili: LIVE — to'ldirilgan qizil nishon, oq matn, pulslanuvchi nuqta.
        // Glow'ga ruxsat berilgan uch joydan biri.
        val pulse = rememberInfiniteTransition(label = "live-pulse")
        val dotAlpha by pulse.animateFloat(
            initialValue = 1f,
            targetValue = 0.25f,
            animationSpec = infiniteRepeatable(
                animation = tween(durationMillis = 800),
                repeatMode = RepeatMode.Reverse,
            ),
            label = "live-dot",
        )
        Surface(
            shape = RoundedCornerShape(percent = 50),
            color = c.liveRed,
            modifier = Modifier.neonGlow(c.liveRed, RoundedCornerShape(percent = 50), elevation = 8.dp),
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(5.dp),
                modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp),
            ) {
                Box(
                    Modifier
                        .size(6.dp)
                        .background(Color.White.copy(alpha = dotAlpha), CircleShape),
                )
                Text(
                    LessonFormat.statusLabel(status),
                    style = MaterialTheme.typography.labelMedium,
                    color = Color.White,
                )
            }
        }
        return
    }
    val (bg, fg) = when (status) {
        "ended" -> c.endedSoft to c.textMuted
        "cancelled" -> c.cancelledSoft to c.warning
        else -> c.scheduledSoft to c.info
    }
    Surface(shape = RoundedCornerShape(percent = 50), color = bg) {
        Text(
            LessonFormat.statusLabel(status),
            style = MaterialTheme.typography.labelMedium,
            color = fg,
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp),
        )
    }
}

/**
 * Dars xususiyati yorlig'i — "Parol", "Kutish xonasi", "Yozib olish", "Qulflangan".
 *
 * 🟢K: avval bu yerda `AssistChip` ishlatilgan edi. `AssistChip` `onClick` TALAB
 * qiladi va bosilganda darsni ochib yuborardi — ya'ni yorliq filtr yoki tugmaga
 * o'xshab ko'rinardi. Bu shunchaki **ma'lumot**, shuning uchun bosilmaydigan
 * `Surface`: interfeys nimani bosish mumkinligi haqida yolg'on aytmasligi kerak.
 */
@Composable
private fun LessonBadge(label: String) {
    Surface(
        shape = RoundedCornerShape(percent = 50),
        color = MaterialTheme.colorScheme.surfaceVariant,
        contentColor = MaterialTheme.colorScheme.onSurfaceVariant,
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline),
    ) {
        Text(
            label,
            style = MaterialTheme.typography.labelSmall,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
        )
    }
}

@Composable
private fun LessonCard(
    lesson: Lesson,
    onClick: () -> Unit,
    onEdit: () -> Unit,
    onRecordings: () -> Unit,
    onShare: () -> Unit,
    onCopy: () -> Unit,
) {
    var menuOpen by rememberSaveable { mutableStateOf(false) }

    // B tili: JONLI dars «nafas oladi» (neon chegara + mint nur), qolgan
    // kartalar tinch, ingichka chegarali. Glow faqat shu holatda — qoidaga qara
    // (`ui/theme/Glow.kt`).
    val isLive = lesson.status == "live"
    val cardShape = MaterialTheme.shapes.large
    val cardModifier = if (isLive) {
        Modifier.fillMaxWidth().liveCardGlow(cardShape)
    } else {
        Modifier.fillMaxWidth()
    }
    Card(
        modifier = cardModifier.clickable(onClick = onClick),
        shape = cardShape,
        border = if (isLive) null else BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant),
    ) {
        Column(Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    lesson.title,
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.weight(1f),
                )
                // Ortiqcha amallar menyusi: kartada to'rt tugma bo'lib ketmasin.
                // Zoom ham ro'yxatdagi uchrashuvda aynan shu naqshni ishlatadi.
                Box {
                    IconButton(onClick = { menuOpen = true }) {
                        Icon(Icons.Default.MoreVert, contentDescription = "Boshqa amallar")
                    }
                    DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                        DropdownMenuItem(
                            text = { Text("Tahrirlash") },
                            leadingIcon = { Icon(Icons.Default.Edit, contentDescription = null) },
                            onClick = {
                                menuOpen = false
                                onEdit()
                            },
                        )
                        DropdownMenuItem(
                            text = { Text("Yozuvlar") },
                            leadingIcon = {
                                Icon(Icons.Default.VideoLibrary, contentDescription = null)
                            },
                            onClick = {
                                menuOpen = false
                                onRecordings()
                            },
                        )
                    }
                }
            }
            Spacer(Modifier.height(6.dp))

            Row(
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                StatusBadge(lesson.status)
                // Uzun sana + davomiylik telefonda sig'masdi: "1 soat" ikki qatorga
                // bo'linardi. Sana qisqaradi (`weight` + ellipsis), davomiylik esa
                // hech qachon bo'linmaydi — u eng qisqa va eng kerakli ma'lumot.
                Text(
                    LessonFormat.scheduleLabel(lesson.scheduledAt) ?: "Vaqti belgilanmagan",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f, fill = false),
                )
                LessonFormat.durationLabel(lesson.durationMin)?.let {
                    Text(
                        it,
                        style = MaterialTheme.typography.labelMedium,
                        color = DarslyTheme.colors.textMuted,
                        maxLines = 1,
                    )
                }
            }

            val badges = buildList {
                if (lesson.hasPasscode) add("Parol")
                if (lesson.isWaitingRoomEnabled) add("Kutish xonasi")
                if (lesson.isRecordingEnabled) add("Yozib olish")
                if (lesson.isLocked) add("Qulflangan")
            }
            if (badges.isNotEmpty()) {
                Spacer(Modifier.height(8.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    badges.forEach { label -> LessonBadge(label) }
                }
            }

            Spacer(Modifier.height(8.dp))
            // Tugmalar eni EKRANDAN hisoblanadi (ControlBar'dagi kabi): telefonda
            // "Davom etish" yozuvi ikki qatorga bo'linib tugmani baland qilardi
            // (qurilmada 2026-07-28 da ko'rindi; planshetda ko'rinmagan). Tor
            // kartada "Ulashish" yozuvi olib tashlanadi — ikonka o'zi yetadi.
            BoxWithConstraints(Modifier.fillMaxWidth()) {
                val compact = maxWidth < 320.dp
                Row(
                    Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    // Yakunlangan darsda tugma "Yozuvlar" bo'ladi va xona OCHILMAYDI —
                    // yo'nalish `MainActivity` da, qoida esa `LessonActions` da (sof,
                    // test ostida). Avval har qanday dars xonani ochib, keyin serverdan
                    // `lesson is not active` (400) qaytardi.
                    val primary = LessonActions.primary(lesson.status)
                    if (isLive) {
                        // Jonli darsda harakat YAGONA va yorqin — to'ldirilgan mint.
                        Button(
                            onClick = onClick,
                            modifier = Modifier.weight(1f),
                        ) {
                            Text(LessonActions.label(primary), maxLines = 1)
                        }
                    } else {
                        // B tili: jonli bo'lmagan kartada tugma kontur (outline) —
                        // ekranda bir vaqtda faqat bitta narsa «yonadi».
                        OutlinedButton(
                            onClick = onClick,
                            enabled = primary != LessonActions.Primary.NONE,
                            border = BorderStroke(1.dp, DarslyTheme.colors.mintOutline),
                            colors = ButtonDefaults.outlinedButtonColors(
                                contentColor = MaterialTheme.colorScheme.primary,
                            ),
                            modifier = Modifier.weight(1f),
                        ) {
                            Text(LessonActions.label(primary), maxLines = 1)
                        }
                    }
                    // M8 — join havolasi Telegramga tizim "Ulashish" oynasi orqali.
                    TextButton(onClick = onShare, enabled = lesson.joinSlug != null) {
                        Icon(
                            Icons.Default.Share,
                            contentDescription = "Havolani ulashish",
                            modifier = Modifier.size(18.dp),
                        )
                        if (!compact) {
                            // 🟢J: gorizontal qator — bo'shliq `width` bo'lishi kerak
                            // (`height(0.dp)` hech narsa qilmaydigan qator edi).
                            Spacer(Modifier.width(6.dp))
                            Text("Ulashish")
                        }
                    }
                    TextButton(onClick = onCopy, enabled = lesson.joinSlug != null) {
                        Icon(Icons.Default.ContentCopy, contentDescription = "Havolani nusxalash", modifier = Modifier.size(18.dp))
                    }
                }
            }
        }
    }
}
