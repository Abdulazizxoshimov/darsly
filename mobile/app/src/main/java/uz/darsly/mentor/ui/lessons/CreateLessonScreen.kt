@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.lessons

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TimePicker
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.material3.rememberTimePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import uz.darsly.mentor.data.api.Lesson
import uz.darsly.mentor.util.LessonFormat
import java.time.Instant
import java.time.ZoneId

/**
 * Dars yaratish (M7).
 *
 * NEGA to'liq ekranli `Dialog`, alohida navigatsiya yo'nalishi emas: forma bitta
 * qadamdan iborat va `MainActivity` navigatsiya grafigiga tegmaslik kerak edi
 * (u boshqa vazifada tahrirlanmoqda). Xulq foydalanuvchi uchun bir xil —
 * "Orqaga" tugmasi ham ekranni yopadi.
 */
@Composable
fun CreateLessonDialog(
    onDismiss: () -> Unit,
    onCreated: (Lesson) -> Unit,
    vm: CreateLessonViewModel = viewModel(),
) {
    val state by vm.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.created) {
        state.created?.let {
            onCreated(it)
            vm.reset()
        }
    }

    // 🟢H: yopishda forma tozalanadi. ViewModel navigatsiya yozuviga bog'langan,
    // ya'ni tozalanmasa keyingi ochilishda eski sarlavha va sana turib qolardi.
    val dismiss = {
        vm.reset()
        onDismiss()
    }

    Dialog(
        onDismissRequest = { if (!state.submitting) dismiss() },
        properties = DialogProperties(usePlatformDefaultWidth = false),
    ) {
        Scaffold(
            modifier = Modifier.fillMaxSize(),
            topBar = {
                TopAppBar(
                    title = { Text("Yangi dars") },
                    navigationIcon = {
                        IconButton(onClick = dismiss, enabled = !state.submitting) {
                            Icon(Icons.Default.Close, contentDescription = "Yopish")
                        }
                    },
                    actions = {
                        TextButton(
                            onClick = { vm.submit() },
                            enabled = !state.submitting && state.errors.isValid,
                        ) {
                            if (state.submitting) {
                                CircularProgressIndicator(Modifier.height(18.dp))
                            } else {
                                Text("Yaratish")
                            }
                        }
                    },
                )
            },
        ) { padding ->
            CreateLessonForm(
                state = state,
                onEdit = vm::edit,
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
            )
        }
    }
}

@Composable
private fun CreateLessonForm(
    state: CreateLessonUiState,
    onEdit: ((LessonForm.Input) -> LessonForm.Input) -> Unit,
    modifier: Modifier = Modifier,
) {
    val input = state.input
    val errors = state.errors
    val show = state.showErrors

    var datePickerOpen by rememberSaveable { mutableStateOf(false) }
    var timePickerOpen by rememberSaveable { mutableStateOf(false) }
    // Sana tanlangach vaqt so'raladi; ikkisi qo'shilib bitta lahzaga aylanadi.
    var pendingDateMillis by rememberSaveable { mutableStateOf<Long?>(null) }

    Column(modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {

        OutlinedTextField(
            value = input.title,
            onValueChange = { v -> onEdit { it.copy(title = v) } },
            label = { Text("Dars nomi") },
            isError = show && errors.title != null,
            supportingText = { if (show) errors.title?.let { Text(it) } },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
            modifier = Modifier.fillMaxWidth(),
        )

        OutlinedTextField(
            value = input.description,
            onValueChange = { v -> onEdit { it.copy(description = v) } },
            label = { Text("Tavsif (ixtiyoriy)") },
            isError = show && errors.description != null,
            supportingText = { if (show) errors.description?.let { Text(it) } },
            minLines = 2,
            maxLines = 4,
            modifier = Modifier.fillMaxWidth(),
        )

        // ── Vaqt ──────────────────────────────────────────────────────────────
        Text("Boshlanish vaqti", style = MaterialTheme.typography.labelLarge)
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedButton(
                onClick = { datePickerOpen = true },
                modifier = Modifier.weight(1f),
            ) {
                Text(
                    input.scheduledAtMillis
                        ?.let { LessonFormat.scheduleLabel(LessonForm.rfc3339Utc(it)) }
                        ?: "Vaqtni tanlash",
                )
            }
            if (input.scheduledAtMillis != null) {
                TextButton(onClick = { onEdit { it.copy(scheduledAtMillis = null) } }) {
                    Text("Tozalash")
                }
            }
        }
        if (input.scheduledAtMillis == null) {
            Text(
                "Vaqt belgilanmasa dars istalgan paytda boshlanishi mumkin",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        errors.scheduleWarning?.let {
            Text(it, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error)
        }

        OutlinedTextField(
            value = input.duration,
            onValueChange = { v -> onEdit { it.copy(duration = v.filter(Char::isDigit)) } },
            label = { Text("Davomiyligi (daqiqa)") },
            isError = show && errors.duration != null,
            supportingText = {
                val text = if (show) errors.duration else null
                Text(text ?: "Bo'sh qoldirilsa ${LessonForm.DURATION_DEFAULT} daqiqa")
            },
            singleLine = true,
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Number,
                imeAction = ImeAction.Next,
            ),
            modifier = Modifier.fillMaxWidth(),
        )

        OutlinedTextField(
            value = input.passcode,
            onValueChange = { v -> onEdit { it.copy(passcode = v) } },
            label = { Text("Parol (ixtiyoriy)") },
            isError = show && errors.passcode != null,
            supportingText = {
                val text = if (show) errors.passcode else null
                Text(text ?: "Parol qo'yilsa o'quvchi havoladan tashqari parol ham kiritadi")
            },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
            modifier = Modifier.fillMaxWidth(),
        )

        SwitchRow(
            title = "Kutish xonasi",
            subtitle = "O'quvchini siz qabul qilganingizdan keyin kiritadi",
            checked = input.waitingRoom,
            onChange = { v -> onEdit { it.copy(waitingRoom = v) } },
        )
        SwitchRow(
            title = "Yozib olish",
            subtitle = "Dars serverda yoziladi (keyin yuklab olish mumkin)",
            checked = input.recording,
            onChange = { v -> onEdit { it.copy(recording = v) } },
        )

        state.serverError?.let {
            Text(it, color = MaterialTheme.colorScheme.error)
        }
        Spacer(Modifier.height(24.dp))
    }

    if (datePickerOpen) {
        val zone = remember { ZoneId.systemDefault() }
        val dateState = rememberDatePickerState(
            initialSelectedDateMillis = input.scheduledAtMillis?.let { toUtcMidnight(it, zone) },
        )
        DatePickerDialog(
            onDismissRequest = { datePickerOpen = false },
            confirmButton = {
                TextButton(
                    onClick = {
                        pendingDateMillis = dateState.selectedDateMillis
                        datePickerOpen = false
                        timePickerOpen = true
                    },
                    enabled = dateState.selectedDateMillis != null,
                ) { Text("Keyingisi") }
            },
            dismissButton = {
                TextButton(onClick = { datePickerOpen = false }) { Text("Bekor qilish") }
            },
        ) {
            DatePicker(state = dateState)
        }
    }

    if (timePickerOpen) {
        val zone = remember { ZoneId.systemDefault() }
        val initial = remember(input.scheduledAtMillis) {
            Instant.ofEpochMilli(input.scheduledAtMillis ?: System.currentTimeMillis()).atZone(zone)
        }
        val timeState = rememberTimePickerState(
            initialHour = initial.hour,
            initialMinute = initial.minute,
            is24Hour = true,
        )
        AlertDialog(
            onDismissRequest = { timePickerOpen = false },
            confirmButton = {
                TextButton(onClick = {
                    val date = pendingDateMillis
                    if (date != null) {
                        val millis = LessonForm.combineDateAndTime(
                            dateUtcMillis = date,
                            hour = timeState.hour,
                            minute = timeState.minute,
                            zone = zone,
                        )
                        onEdit { it.copy(scheduledAtMillis = millis) }
                    }
                    timePickerOpen = false
                }) { Text("Tayyor") }
            },
            dismissButton = {
                TextButton(onClick = { timePickerOpen = false }) { Text("Bekor qilish") }
            },
            title = { Text("Boshlanish soati") },
            text = { TimePicker(state = timeState) },
        )
    }
}

@Composable
private fun SwitchRow(
    title: String,
    subtitle: String,
    checked: Boolean,
    onChange: (Boolean) -> Unit,
) {
    Row(
        Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.bodyLarge)
            Text(
                subtitle,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        Switch(checked = checked, onCheckedChange = onChange)
    }
}

/** Tanlangan lahzadan sana tanlagich kutadigan "UTC yarim tun" qiymatini yasaydi. */
private fun toUtcMidnight(millis: Long, zone: ZoneId): Long =
    Instant.ofEpochMilli(millis).atZone(zone).toLocalDate()
        .atStartOfDay(java.time.ZoneOffset.UTC).toInstant().toEpochMilli()
