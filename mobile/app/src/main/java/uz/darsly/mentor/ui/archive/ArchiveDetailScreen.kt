@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.archive

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Download
import androidx.compose.material.icons.filled.Fullscreen
import androidx.compose.material.icons.filled.FullscreenExit
import androidx.compose.material.icons.filled.InsertDriveFile
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Send
import androidx.compose.material.icons.filled.Speed
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TextField
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.media3.common.MediaItem
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.ui.PlayerView
import uz.darsly.mentor.data.api.ArchiveChatMessageDto
import uz.darsly.mentor.data.api.ArchiveRecordingDto
import uz.darsly.mentor.util.RecordingFormat
import uz.darsly.mentor.util.Share
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

/**
 * Dars arxivi — video (ilova ichida ExoPlayer) + Telegram-uslub chat paneli.
 *
 * Layout MOSLASHUVCHAN: tik holatda video tepada / chat tagida; keng ekranda
 * (yon/planshet) video chapda / chat o'ngda. To'liq-ekranda video butun
 * ekranni egallaydi, chat yashirinadi (Zoom naqshi). Chat videoga SINXRON emas
 * — alohida o'qish paneli.
 */
@androidx.annotation.OptIn(androidx.media3.common.util.UnstableApi::class)
@Composable
fun ArchiveDetailScreen(
    lessonId: String,
    lessonTitle: String,
    onBack: () -> Unit,
    vm: ArchiveDetailViewModel = hiltViewModel(),
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val context = LocalContext.current

    LaunchedEffect(lessonId) { vm.start(lessonId, lessonTitle) }

    // Tashqi ochish (yuklab olish / Telegram) — tizim ilovasiga uzatiladi.
    LaunchedEffect(state.openUrl) {
        state.openUrl?.let { url ->
            Share.openUrl(context, url)
            vm.openHandled()
        }
    }

    val exoPlayer = remember { ExoPlayer.Builder(context).build() }
    androidx.compose.runtime.DisposableEffect(Unit) {
        onDispose { exoPlayer.release() }
    }

    val recording = state.archive?.recording
    val videoUrl = recording?.takeIf { it.status == RecordingFormat.STATUS_READY }?.url

    LaunchedEffect(videoUrl) {
        if (videoUrl != null) {
            exoPlayer.setMediaItem(MediaItem.fromUri(videoUrl))
            exoPlayer.prepare()
        }
    }

    var fullscreen by rememberSaveable { mutableStateOf(false) }

    // To'liq ekran — Scaffold'siz, faqat pleyer + chiqish tugmasi.
    if (fullscreen && videoUrl != null) {
        Box(Modifier.fillMaxSize().background(Color.Black)) {
            VideoPlayer(exoPlayer, Modifier.fillMaxSize())
            IconButton(
                onClick = { fullscreen = false },
                modifier = Modifier.align(Alignment.TopEnd).padding(8.dp),
            ) {
                Icon(Icons.Default.FullscreenExit, contentDescription = "Kichraytirish", tint = Color.White)
            }
        }
        return
    }

    Scaffold(
        topBar = {
            TopAppBar(
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Orqaga")
                    }
                },
                title = {
                    Text(
                        state.lessonTitle.ifBlank { "Dars arxivi" },
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                },
            )
        },
    ) { padding ->
        Box(Modifier.fillMaxSize().padding(padding)) {
            when {
                state.loading && state.archive == null -> Box(
                    Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center,
                ) { CircularProgressIndicator() }

                state.error != null && state.archive == null ->
                    CenterMessage(state.error!!, isError = true, onRetry = { vm.refresh() })

                else -> BoxWithConstraints(Modifier.fillMaxSize()) {
                    val wide = maxWidth >= 600.dp
                    val videoBlock: @Composable (Modifier) -> Unit = { mod ->
                        VideoBlock(
                            modifier = mod,
                            recording = recording,
                            hasVideo = videoUrl != null,
                            player = exoPlayer,
                            onFullscreen = { fullscreen = true },
                            onDownload = { videoUrl?.let(vm::requestOpen) },
                            onOpenTelegram = { recording?.telegramUrl?.let(vm::requestOpen) },
                        )
                    }
                    val chatBlock: @Composable (Modifier) -> Unit = { mod ->
                        ChatPanel(
                            modifier = mod,
                            messages = state.filteredChat,
                            query = state.chatQuery,
                            onQuery = vm::setChatQuery,
                            onOpenFile = vm::requestOpen,
                        )
                    }

                    if (wide) {
                        Row(Modifier.fillMaxSize()) {
                            videoBlock(Modifier.weight(0.62f).fillMaxHeight())
                            chatBlock(Modifier.weight(0.38f).fillMaxHeight())
                        }
                    } else {
                        Column(Modifier.fillMaxSize()) {
                            videoBlock(Modifier.fillMaxWidth())
                            chatBlock(Modifier.fillMaxWidth().weight(1f))
                        }
                    }
                }
            }
        }
    }
}

@androidx.annotation.OptIn(androidx.media3.common.util.UnstableApi::class)
@Composable
private fun VideoPlayer(player: ExoPlayer, modifier: Modifier = Modifier) {
    AndroidView(
        modifier = modifier,
        factory = { ctx ->
            PlayerView(ctx).apply {
                this.player = player
                useController = true
            }
        },
    )
}

@androidx.annotation.OptIn(androidx.media3.common.util.UnstableApi::class)
@Composable
private fun VideoBlock(
    modifier: Modifier,
    recording: ArchiveRecordingDto?,
    hasVideo: Boolean,
    player: ExoPlayer,
    onFullscreen: () -> Unit,
    onDownload: () -> Unit,
    onOpenTelegram: () -> Unit,
) {
    Column(modifier.background(Color.Black)) {
        if (hasVideo) {
            VideoPlayer(player, Modifier.fillMaxWidth().aspectRatio(16f / 9f))
            ControlBar(
                player = player,
                hasTelegram = recording?.telegramUrl != null,
                onFullscreen = onFullscreen,
                onDownload = onDownload,
                onOpenTelegram = onOpenTelegram,
            )
        } else {
            // Yozuv yo'q yoki tayyor emas — holatga qarab tushunarli xabar.
            Box(
                Modifier.fillMaxWidth().aspectRatio(16f / 9f),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    videoPlaceholder(recording),
                    color = Color.White,
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.padding(24.dp),
                )
            }
        }
    }
}

@androidx.annotation.OptIn(androidx.media3.common.util.UnstableApi::class)
@Composable
private fun ControlBar(
    player: ExoPlayer,
    hasTelegram: Boolean,
    onFullscreen: () -> Unit,
    onDownload: () -> Unit,
    onOpenTelegram: () -> Unit,
) {
    var speed by rememberSaveable { mutableFloatStateOf(1f) }
    var speedMenu by remember { mutableStateOf(false) }

    Row(
        Modifier.fillMaxWidth().padding(horizontal = 4.dp, vertical = 2.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box {
            TextButton(onClick = { speedMenu = true }) {
                Icon(Icons.Default.Speed, contentDescription = null, tint = Color.White, modifier = Modifier.size(18.dp))
                Spacer(Modifier.width(4.dp))
                Text(speedLabel(speed), color = Color.White)
            }
            DropdownMenu(expanded = speedMenu, onDismissRequest = { speedMenu = false }) {
                listOf(0.5f, 0.75f, 1f, 1.25f, 1.5f, 2f).forEach { s ->
                    DropdownMenuItem(
                        text = { Text(speedLabel(s)) },
                        onClick = {
                            speed = s
                            player.setPlaybackSpeed(s)
                            speedMenu = false
                        },
                    )
                }
            }
        }
        Spacer(Modifier.weight(1f))
        if (hasTelegram) {
            IconButton(onClick = onOpenTelegram) {
                Icon(Icons.Default.Send, contentDescription = "Telegramda ochish", tint = Color.White)
            }
        }
        IconButton(onClick = onDownload) {
            Icon(Icons.Default.Download, contentDescription = "Yuklab olish", tint = Color.White)
        }
        IconButton(onClick = onFullscreen) {
            Icon(Icons.Default.Fullscreen, contentDescription = "To'liq ekran", tint = Color.White)
        }
    }
}

@Composable
private fun ChatPanel(
    modifier: Modifier,
    messages: List<ArchiveChatMessageDto>,
    query: String,
    onQuery: (String) -> Unit,
    onOpenFile: (String) -> Unit,
) {
    Column(modifier.background(MaterialTheme.colorScheme.surface)) {
        TextField(
            value = query,
            onValueChange = onQuery,
            singleLine = true,
            leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) },
            placeholder = { Text("Chatdan qidirish") },
            modifier = Modifier.fillMaxWidth().padding(8.dp),
        )
        if (messages.isEmpty()) {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text(
                    if (query.isBlank()) "Chat bo'sh" else "Hech narsa topilmadi",
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        } else {
            LazyColumn(
                Modifier.fillMaxSize(),
                contentPadding = PaddingValues(horizontal = 10.dp, vertical = 6.dp),
                verticalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                items(messages, key = { it.id }) { ChatBubble(it, onOpenFile) }
            }
        }
    }
}

@Composable
private fun ChatBubble(msg: ArchiveChatMessageDto, onOpenFile: (String) -> Unit) {
    Surface(
        shape = RoundedCornerShape(12.dp),
        color = MaterialTheme.colorScheme.surfaceVariant,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(Modifier.padding(horizontal = 12.dp, vertical = 8.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    msg.senderName.ifBlank { "Ishtirokchi" },
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.SemiBold,
                    color = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.weight(1f),
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                chatTime(msg.createdAt)?.let {
                    Text(it, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
            if (msg.body.isNotBlank()) {
                Spacer(Modifier.height(2.dp))
                Text(msg.body, style = MaterialTheme.typography.bodyMedium)
            }
            msg.file?.let { file ->
                Spacer(Modifier.height(6.dp))
                OutlinedButton(onClick = { onOpenFile(file.url) }, modifier = Modifier.fillMaxWidth()) {
                    Icon(Icons.Default.InsertDriveFile, contentDescription = null, modifier = Modifier.size(18.dp))
                    Spacer(Modifier.width(6.dp))
                    Text(
                        file.name.ifBlank { "Fayl" },
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        }
    }
}

@Composable
private fun CenterMessage(message: String, isError: Boolean, onRetry: () -> Unit) {
    Column(
        Modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(message, color = if (isError) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurface)
        if (isError) {
            Spacer(Modifier.height(12.dp))
            OutlinedButton(onClick = onRetry) { Text("Qayta urinish") }
        }
    }
}

// ─── Yordamchilar ─────────────────────────────────────────────────────────────

private fun speedLabel(s: Float): String =
    if (s == s.toLong().toFloat()) "${s.toLong()}x" else "${s}x"

private fun videoPlaceholder(rec: ArchiveRecordingDto?): String = when (rec?.status) {
    null -> "Bu darsning yozuvi yo'q"
    RecordingFormat.STATUS_FAILED -> "Yozuv muvaffaqiyatsiz tugadi"
    "processing", "recording" -> "Yozuv hali tayyorlanmoqda — birozdan so'ng qaytadan oching"
    RecordingFormat.STATUS_EXPIRED -> "Yozuv serverdan o'chirilgan"
    else -> "Video mavjud emas"
}

private val timeFmt = DateTimeFormatter.ofPattern("HH:mm")

private fun chatTime(iso: String): String? = runCatching {
    Instant.parse(iso).atZone(ZoneId.systemDefault()).toLocalTime().format(timeFmt)
}.getOrNull()
