@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.schedule

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import uz.darsly.mentor.data.api.Lesson
import uz.darsly.mentor.ui.lessons.CreateLessonViewModel
import uz.darsly.mentor.ui.lessons.LessonFields

/**
 * Dars REJALASHTIRISH (M21) — vaqti oldindan belgilangan dars.
 *
 * «Dars yaratish» (Darslar bo'limi) endi tezkor: vaqt so'ramaydi va darhol
 * xonaga kiritadi. Vaqtli oqim shu yerga — Jadval bo'limiga ko'chdi, chunki
 * mahsulot nuqtai nazaridan «keyinroqqa dars qo'yish» aynan jadval bilan
 * ishlash. Yaratilgach xonaga KIRILMAYDI — dars jadvalda ko'rinadi.
 *
 * ViewModel [CreateLessonViewModel] bilan UMUMIY (so'rov bir xil), farq faqat
 * talabda: bu yerda vaqt MAJBURIY — tugma vaqt tanlanmaguncha yoqilmaydi.
 * `LessonForm.validate` da bu majburiylik yo'q (u tezkor oqim uchun ham xizmat
 * qiladi), shuning uchun talab shu ekran darajasida qo'llanadi.
 */
@Composable
fun ScheduleLessonDialog(
    onDismiss: () -> Unit,
    onScheduled: (Lesson) -> Unit,
    vm: CreateLessonViewModel = hiltViewModel(),
) {
    val state by vm.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.created) {
        state.created?.let {
            onScheduled(it)
            vm.reset()
        }
    }

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
                    title = { Text("Dars rejalashtirish") },
                    navigationIcon = {
                        IconButton(onClick = dismiss, enabled = !state.submitting) {
                            Icon(Icons.Default.Close, contentDescription = "Yopish")
                        }
                    },
                    actions = {
                        TextButton(
                            onClick = { vm.submit() },
                            enabled = !state.submitting &&
                                state.errors.isValid &&
                                state.input.scheduledAtMillis != null,
                        ) {
                            if (state.submitting) {
                                CircularProgressIndicator(Modifier.height(18.dp))
                            } else {
                                Text("Rejalashtirish")
                            }
                        }
                    },
                )
            },
        ) { padding ->
            LessonFields(
                input = state.input,
                errors = state.errors,
                showErrors = state.showErrors,
                onEdit = vm::edit,
                timeHint = "Rejalashtirish uchun boshlanish vaqtini tanlang",
                serverError = state.serverError,
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
            )
        }
    }
}
