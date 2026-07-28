@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.lessons

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import uz.darsly.mentor.data.api.Lesson

/**
 * Darsni tahrirlash va o'chirish.
 *
 * Yaratish oynasi bilan bir xil naqsh (to'liq ekranli `Dialog`) — ustoz uchun
 * ikki oyna bir xil ishlaydi va navigatsiya grafigiga yangi yo'nalish qo'shilmaydi.
 *
 * O'CHIRISH ikki bosqichli: tugma → tasdiq dialogi. Dars o'chirilsa ustoz
 * yuborgan barcha havolalar o'lik bo'ladi, shuning uchun bitta tasodifiy bosish
 * yetarli bo'lmasligi kerak. Tasdiq matni **oqibatni** aytadi, "ishonchingiz
 * komilmi?" degan bo'sh savolni emas.
 */
@Composable
fun EditLessonDialog(
    lesson: Lesson,
    onDismiss: () -> Unit,
    onSaved: (Lesson) -> Unit,
    onDeleted: (Lesson) -> Unit,
    vm: EditLessonViewModel = viewModel(),
) {
    val state by vm.state.collectAsStateWithLifecycle()
    var confirmDelete by rememberSaveable { mutableStateOf(false) }

    LaunchedEffect(lesson.id) { vm.load(lesson) }

    LaunchedEffect(state.saved) {
        state.saved?.let {
            onSaved(it)
            vm.reset()
        }
    }
    LaunchedEffect(state.deleted) {
        if (state.deleted) {
            onDeleted(lesson)
            vm.reset()
        }
    }

    val busy = state.submitting || state.deleting
    val dismiss = {
        vm.reset()
        onDismiss()
    }

    Dialog(
        onDismissRequest = { if (!busy) dismiss() },
        properties = DialogProperties(usePlatformDefaultWidth = false),
    ) {
        Scaffold(
            modifier = Modifier.fillMaxSize(),
            topBar = {
                TopAppBar(
                    title = { Text("Darsni tahrirlash") },
                    navigationIcon = {
                        IconButton(onClick = dismiss, enabled = !busy) {
                            Icon(Icons.Default.Close, contentDescription = "Yopish")
                        }
                    },
                    actions = {
                        TextButton(onClick = { vm.submit() }, enabled = state.canSubmit) {
                            if (state.submitting) {
                                CircularProgressIndicator(Modifier.height(18.dp))
                            } else {
                                Text("Saqlash")
                            }
                        }
                    },
                )
            },
        ) { padding ->
            Column(
                Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
            ) {
                LessonFields(
                    input = state.input,
                    errors = state.errors,
                    showErrors = state.showErrors,
                    onEdit = vm::edit,
                    serverError = state.serverError,
                    // Backend PATCH'da "vaqtni o'chir" signali yo'q
                    // (`LessonForm.toUpdateRequest` izohi) — tugma ko'rsatilmaydi.
                    allowClearTime = false,
                    passcodeEnabled = !state.removePasscode,
                    passcodeLabel = if (lesson.hasPasscode) "Yangi parol" else "Parol (ixtiyoriy)",
                    passcodeHint = if (lesson.hasPasscode) {
                        "Bo'sh qoldirilsa hozirgi parol o'zgarmaydi"
                    } else {
                        "Parol qo'yilsa o'quvchi havoladan tashqari parol ham kiritadi"
                    },
                    passcodeSlot = {
                        if (lesson.hasPasscode) {
                            PasscodeState(
                                removing = state.removePasscode,
                                onRemovingChange = vm::setRemovePasscode,
                            )
                        }
                    },
                )

                // O'chirish — eng pastda va boshqa rangda: tasodifan bosilmasin.
                OutlinedButton(
                    onClick = { confirmDelete = true },
                    enabled = !busy,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (state.deleting) {
                        CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                    } else {
                        Icon(
                            Icons.Default.Delete,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.error,
                            modifier = Modifier.size(18.dp),
                        )
                        Text("  Darsni o'chirish", color = MaterialTheme.colorScheme.error)
                    }
                }
            }
        }
    }

    if (confirmDelete) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            title = { Text("Darsni o'chirasizmi?") },
            text = {
                Text(
                    "\"${lesson.title}\" o'chiriladi va o'quvchilarga yuborgan havolangiz " +
                        "ishlamay qoladi. Dars yozuvlari ham ochilmaydi. Bu amalni ilovadan " +
                        "qaytarib bo'lmaydi.",
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    confirmDelete = false
                    vm.delete()
                }) {
                    Text("O'chirish", color = MaterialTheme.colorScheme.error)
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmDelete = false }) { Text("Bekor qilish") }
            },
        )
    }
}

/**
 * Parol holati va uni olib tashlash tanlovi.
 *
 * Ustoz "parol o'rnatilganmi" degan savolga javob ko'rishi kerak: server hash'ni
 * qaytarmaydi, ya'ni bo'sh maydon "paroli yo'q" degani EMAS. Buni aytmasak,
 * bo'sh maydonni ko'rgan ustoz parol o'chgan deb o'ylardi.
 */
@Composable
private fun PasscodeState(
    removing: Boolean,
    onRemovingChange: (Boolean) -> Unit,
) {
    Surface(
        color = MaterialTheme.colorScheme.surfaceVariant,
        shape = MaterialTheme.shapes.medium,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Row(
            Modifier.padding(horizontal = 12.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(Modifier.weight(1f)) {
                Text("Parol o'rnatilgan", style = MaterialTheme.typography.bodyMedium)
                Text(
                    if (removing) {
                        "Saqlaganingizda parol olib tashlanadi"
                    } else {
                        "O'quvchi havoladan tashqari parolni ham kiritadi"
                    },
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            TextButton(onClick = { onRemovingChange(!removing) }) {
                Text(if (removing) "Bekor qilish" else "Olib tashlash")
            }
        }
    }
}
