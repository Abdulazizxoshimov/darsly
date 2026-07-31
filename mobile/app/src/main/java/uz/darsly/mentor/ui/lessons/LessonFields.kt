@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.lessons

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.FiberManualRecord
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.Surface
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TimePicker
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.material3.rememberTimePickerState
import androidx.compose.runtime.Composable
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
import uz.darsly.mentor.util.LessonFormat
import java.time.Instant
import java.time.ZoneId
import java.time.ZoneOffset

/**
 * Dars formasining maydonlari — **yaratish va tahrirlash uchun bitta manba**.
 *
 * NEGA UMUMIY: ikki forma bir xil chegaralarni (`LessonForm`) va bir xil
 * ko'rinishni talab qiladi. Nusxa ko'chirilgan bo'lsa, biri o'zgarganda ikkinchisi
 * jimgina orqada qolardi — bu aynan "web'da bor, mobilda yo'q" turdagi farqlarni
 * tug'diradi.
 *
 * Farqlar parametrlar orqali beriladi va ularning har biri **haqiqiy** farq:
 *  · [allowClearTime] — tahrirlashda vaqtni olib tashlab bo'lmaydi (backend
 *    cheklovi, `LessonForm.toUpdateRequest` izohiga qara);
 *  · [passcodeSlot] — tahrirlashda parol maydoni ustida "hozir o'rnatilgan" holati
 *    va "olib tashlash" tanlovi turadi;
 *  · [instant] — tezkor dars (darhol boshlanadi): tavsif, vaqt va davomiylik
 *    yashiriladi — Zoom'ning "New meeting"i kabi faqat nom + parol + ikki toggle.
 */
@Composable
fun LessonFields(
    input: LessonForm.Input,
    errors: LessonForm.Errors,
    showErrors: Boolean,
    onEdit: ((LessonForm.Input) -> LessonForm.Input) -> Unit,
    modifier: Modifier = Modifier,
    instant: Boolean = false,
    allowClearTime: Boolean = true,
    timeHint: String = "Vaqt belgilanmasa dars istalgan paytda boshlanishi mumkin",
    passcodeEnabled: Boolean = true,
    passcodeLabel: String = "Parol (ixtiyoriy)",
    passcodeHint: String = "Parol qo'yilsa o'quvchi havoladan tashqari parol ham kiritadi",
    passcodeSlot: @Composable (() -> Unit)? = null,
    serverError: String? = null,
) {
    var datePickerOpen by rememberSaveable { mutableStateOf(false) }
    var timePickerOpen by rememberSaveable { mutableStateOf(false) }
    // Sana tanlangach vaqt so'raladi; ikkisi qo'shilib bitta lahzaga aylanadi.
    var pendingDateMillis by rememberSaveable { mutableStateOf<Long?>(null) }

    Column(modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {

        OutlinedTextField(
            value = input.title,
            onValueChange = { v -> onEdit { it.copy(title = v) } },
            label = { Text("Dars nomi") },
            isError = showErrors && errors.title != null,
            supportingText = { if (showErrors) errors.title?.let { Text(it) } },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
            modifier = Modifier.fillMaxWidth(),
        )

        if (!instant) {
            OutlinedTextField(
                value = input.description,
                onValueChange = { v -> onEdit { it.copy(description = v) } },
                label = { Text("Tavsif (ixtiyoriy)") },
                isError = showErrors && errors.description != null,
                supportingText = { if (showErrors) errors.description?.let { Text(it) } },
                minLines = 2,
                maxLines = 4,
                modifier = Modifier.fillMaxWidth(),
            )

            // ── Vaqt ──────────────────────────────────────────────────────────
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
                if (allowClearTime && input.scheduledAtMillis != null) {
                    TextButton(onClick = { onEdit { it.copy(scheduledAtMillis = null) } }) {
                        Text("Tozalash")
                    }
                }
            }
            if (input.scheduledAtMillis == null) {
                Text(
                    timeHint,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            } else if (!allowClearTime) {
                // Yolg'on tugma qo'yib, bosilganda jimgina ishlamagandan ko'ra —
                // sababni aytish. Backend PATCH'da "vaqtni o'chir" signali yo'q.
                Text(
                    "Vaqtni boshqa vaqtga o'zgartirish mumkin, lekin butunlay olib tashlab bo'lmaydi",
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
                isError = showErrors && errors.duration != null,
                supportingText = {
                    val text = if (showErrors) errors.duration else null
                    Text(text ?: "Bo'sh qoldirilsa ${LessonForm.DURATION_DEFAULT} daqiqa")
                },
                singleLine = true,
                keyboardOptions = KeyboardOptions(
                    keyboardType = KeyboardType.Number,
                    imeAction = ImeAction.Next,
                ),
                modifier = Modifier.fillMaxWidth(),
            )
        }

        passcodeSlot?.invoke()

        OutlinedTextField(
            value = input.passcode,
            onValueChange = { v -> onEdit { it.copy(passcode = v) } },
            label = { Text(passcodeLabel) },
            enabled = passcodeEnabled,
            isError = showErrors && errors.passcode != null,
            supportingText = {
                val text = if (showErrors) errors.passcode else null
                Text(text ?: passcodeHint)
            },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
            modifier = Modifier.fillMaxWidth(),
        )

        LessonSwitchRow(
            title = "Kutish xonasi",
            subtitle = "O'quvchini siz qabul qilganingizdan keyin kiritadi",
            checked = input.waitingRoom,
            onChange = { v -> onEdit { it.copy(waitingRoom = v) } },
        )

        // Yozib olish — DEFAULT YONIQ, lekin o'chirsa bo'ladi.
        LessonSwitchRow(
            title = "Yozib olish",
            subtitle = if (input.recording) {
                "Dars boshlanishi bilan avtomatik yoziladi — hech narsa bosish shart emas"
            } else {
                "O'chirilgan — bu dars yozilmaydi"
            },
            checked = input.recording,
            onChange = { v -> onEdit { it.copy(recording = v) } },
        )

        // ── Ovoz (№11) ────────────────────────────────────────────────────────
        // TEZKOR darsda ATAYLAB YO'Q: "New meeting" bosgan ustoz sozlamalar
        // ro'yxatini emas, darhol xonani kutadi. Default'lar (ikkalasi ham
        // yoniq) tezkor darsda ham qo'llanadi — ular server default'i bilan
        // bir xil bo'lgani uchun hech narsa yo'qolmaydi.
        if (!instant) {
            LessonSwitchRow(
                title = "Kirganda mikrofon o'chiq bo'lsin",
                subtitle = if (input.muteOnEntry) {
                    "Kech kelgan o'quvchi darsga shovqin bilan kirmaydi"
                } else {
                    "O'chirilgan — o'quvchi kirganda mikrofoni yoniq bo'ladi"
                },
                checked = input.muteOnEntry,
                onChange = { v -> onEdit { it.copy(muteOnEntry = v) } },
            )

            LessonSwitchRow(
                title = "O'quvchi mikrofonni o'zi yoqa olsin",
                subtitle = if (input.allowSelfUnmute) {
                    "O'quvchi savol berish uchun mikrofonini o'zi yoqadi"
                } else {
                    "Faqat siz ruxsat berganingizda gapira oladi (ma'ruza rejimi)"
                },
                checked = input.allowSelfUnmute,
                onChange = { v -> onEdit { it.copy(allowSelfUnmute = v) } },
            )
        }

        serverError?.let { Text(it, color = MaterialTheme.colorScheme.error) }
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
fun LessonSwitchRow(
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
        .atStartOfDay(ZoneOffset.UTC).toInstant().toEpochMilli()
