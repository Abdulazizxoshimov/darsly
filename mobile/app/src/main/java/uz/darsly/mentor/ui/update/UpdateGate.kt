package uz.darsly.mentor.ui.update

import android.content.ActivityNotFoundException
import android.content.Intent
import android.net.Uri
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.window.DialogProperties
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import io.livekit.android.util.LKLog
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import uz.darsly.mentor.BuildConfig
import uz.darsly.mentor.data.api.Net

/** Ishga tushganda `GET /api/v1/app-config` ni bir marta so'raydi (M42). */
class UpdateViewModel : ViewModel() {

    private val _state = MutableStateFlow<UpdateState>(UpdateState.None)
    val state: StateFlow<UpdateState> = _state.asStateFlow()

    init { check() }

    fun check() {
        viewModelScope.launch {
            val cfg = runCatching { Net.api.appConfig().data?.android }
                .onFailure { LKLog.w(it) { "app-config o'qilmadi — fail-open" } }
                .getOrNull()
            _state.value = UpdateDecider.decide(BuildConfig.VERSION_NAME, cfg)
        }
    }
}

/**
 * Yangilanish dialogi. Butun UI ustiga qo'yiladi (`MainActivity`).
 *
 * · [UpdateState.Required] → **yopib bo'lmaydi**: orqaga tugmasi ham, tashqariga
 *   bosish ham dialogni yopmaydi va "Keyinroq" tugmasi yo'q.
 * · [UpdateState.Optional] → yopiladigan eslatma (sessiyada bir marta).
 */
@Composable
fun UpdateGate(vm: UpdateViewModel = viewModel()) {
    val state by vm.state.collectAsStateWithLifecycle()
    val ctx = LocalContext.current
    var optionalDismissed by remember { mutableStateOf(false) }

    fun openApk(url: String) {
        if (url.isBlank()) return
        runCatching {
            ctx.startActivity(
                Intent(Intent.ACTION_VIEW, Uri.parse(url)).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
            )
        }.onFailure { e ->
            if (e is ActivityNotFoundException) LKLog.w(e) { "APK havolasi ochilmadi" }
        }
    }

    when (val s = state) {
        is UpdateState.Required -> AlertDialog(
            onDismissRequest = { /* ataylab bo'sh — bloklovchi dialog */ },
            properties = DialogProperties(
                dismissOnBackPress = false,
                dismissOnClickOutside = false,
            ),
            title = { Text("Yangilanish talab qilinadi") },
            text = {
                Text(
                    buildString {
                        append("Ilovaning bu versiyasi (${BuildConfig.VERSION_NAME}) endi ")
                        append("qo‘llab-quvvatlanmaydi.")
                        if (s.minVersion.isNotBlank()) {
                            append(" Kamida ${s.minVersion} versiyasi kerak.")
                        }
                        if (s.notes.isNotBlank()) append("\n\n${s.notes}")
                        if (s.apkUrl.isBlank()) {
                            append("\n\nYangi APK havolasi hali berilmagan — administratorga murojaat qiling.")
                        }
                    },
                )
            },
            confirmButton = {
                if (s.apkUrl.isNotBlank()) {
                    TextButton(onClick = { openApk(s.apkUrl) }) { Text("Yangilash") }
                }
            },
        )

        is UpdateState.Optional -> if (!optionalDismissed) {
            AlertDialog(
                onDismissRequest = { optionalDismissed = true },
                title = { Text("Yangi versiya mavjud") },
                text = {
                    Text(
                        buildString {
                            append("Yangi versiya: ${s.latest} (sizda ${BuildConfig.VERSION_NAME}).")
                            if (s.notes.isNotBlank()) append("\n\n${s.notes}")
                        },
                    )
                },
                confirmButton = {
                    if (s.apkUrl.isNotBlank()) {
                        TextButton(onClick = { optionalDismissed = true; openApk(s.apkUrl) }) {
                            Text("Yangilash")
                        }
                    }
                },
                dismissButton = {
                    TextButton(onClick = { optionalDismissed = true }) { Text("Keyinroq") }
                },
            )
        }

        UpdateState.None -> Unit
    }
}
