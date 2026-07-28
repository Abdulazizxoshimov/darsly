@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.lessons

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
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import uz.darsly.mentor.data.api.Lesson

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
            // Maydonlar tahrirlash ekrani bilan UMUMIY (`LessonFields`) — ikkala
            // forma bir xil chegara va bir xil ko'rinishda qolishi uchun.
            LessonFields(
                input = state.input,
                errors = state.errors,
                showErrors = state.showErrors,
                onEdit = vm::edit,
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
