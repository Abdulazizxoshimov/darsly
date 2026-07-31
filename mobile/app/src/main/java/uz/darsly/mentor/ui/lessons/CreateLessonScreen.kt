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
import androidx.hilt.navigation.compose.hiltViewModel
import uz.darsly.mentor.data.api.Lesson

/**
 * Tezkor dars yaratish (M7, M21'da soddalashtirildi).
 *
 * Mahsulot qarori: «Dars yaratish» = darhol boshlash (Zoom'ning "New meeting"i).
 * Shuning uchun formada FAQAT nom, parol (ixtiyoriy), kutish xonasi va yozib
 * olish bor — tavsif/vaqt/davomiylik so'ralmaydi. Vaqtli dars endi Jadval
 * bo'limidagi «Dars rejalashtirish» orqali ([uz.darsly.mentor.ui.schedule.ScheduleLessonDialog]).
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
    vm: CreateLessonViewModel = hiltViewModel(),
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
                    title = { Text("Yangi dars — hoziroq") },
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
                                Text("Boshlash")
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
                instant = true,
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
