@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.auth

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import uz.darsly.mentor.ui.profile.ProfileForm

/**
 * Parolni tiklash — uch qadamli sehrgar (email → xat → yangi parol).
 *
 * Qadamlar bitta ekranda: har biri o'z sarlavhasi va bitta asosiy tugmasiga ega.
 * Alohida navigatsiya yo'nalishlariga bo'lish bu yerda foyda bermasdi — oqim
 * chiziqli va orqaga qaytish faqat bir qadam.
 */
@Composable
fun PasswordResetScreen(
    onBack: () -> Unit,
    onDone: () -> Unit,
    vm: PasswordResetViewModel = viewModel(),
) {
    val state by vm.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.step) {
        if (state.step == ResetStep.DONE) onDone()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                navigationIcon = {
                    IconButton(onClick = {
                        // CONFIRM qadamidan "Orqaga" — oldingi qadamga, ekrandan emas.
                        if (state.step == ResetStep.CONFIRM) vm.goToRequest() else onBack()
                    }) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Orqaga")
                    }
                },
                title = { Text("Parolni tiklash") },
            )
        },
    ) { padding ->
        Column(
            Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            when (state.step) {
                ResetStep.REQUEST -> RequestStep(state, vm)
                ResetStep.SENT -> SentStep(state, vm)
                ResetStep.CONFIRM -> ConfirmStep(state, vm)
                // DONE — `LaunchedEffect` darhol ekrandan chiqaradi; bu holat
                // faqat bir kadr davom etadi.
                ResetStep.DONE -> CircularProgressIndicator()
            }
        }
    }
}

@Composable
private fun RequestStep(state: PasswordResetUiState, vm: PasswordResetViewModel) {
    Text("Parolni unutdingizmi?", style = MaterialTheme.typography.headlineSmall)
    Spacer(Modifier.height(8.dp))
    Text(
        "Hisobingiz emailini kiriting — tiklash havolasini yuboramiz.",
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
    Spacer(Modifier.height(24.dp))

    OutlinedTextField(
        value = state.email,
        onValueChange = vm::setEmail,
        label = { Text("Email") },
        isError = state.emailError != null,
        supportingText = { state.emailError?.let { Text(it) } },
        singleLine = true,
        keyboardOptions = KeyboardOptions(
            keyboardType = KeyboardType.Email,
            imeAction = ImeAction.Done,
        ),
        modifier = Modifier.fillMaxWidth().widthIn(max = 420.dp),
    )
    Spacer(Modifier.height(16.dp))

    Button(
        onClick = { vm.requestReset() },
        enabled = !state.submitting,
        modifier = Modifier.fillMaxWidth().widthIn(max = 420.dp).height(52.dp),
    ) {
        if (state.submitting) {
            CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp)
        } else {
            Text("Havola yuborish")
        }
    }

    state.error?.let {
        Spacer(Modifier.height(16.dp))
        Text(it, color = MaterialTheme.colorScheme.error)
    }

    Spacer(Modifier.height(8.dp))
    TextButton(onClick = { vm.goToConfirm() }) {
        Text("Kod allaqachon bor")
    }
}

@Composable
private fun SentStep(state: PasswordResetUiState, vm: PasswordResetViewModel) {
    Text("Xatni tekshiring", style = MaterialTheme.typography.headlineSmall)
    Spacer(Modifier.height(8.dp))
    // MATN ATAYLAB SHARTLI: backend "topildi/topilmadi" ni ayirmaydi
    // (hisob mavjudligini aniqlash oracle'i bo'lmasligi uchun). Biz ham
    // bilmaymiz, shuning uchun bilgandek gapirmaymiz.
    Text(
        "Agar ${state.email.trim()} ro'yxatdan o'tgan bo'lsa, unga parolni " +
            "tiklash havolasi yuborildi. Xat kelmasa \"Spam\" papkasini ham tekshiring.",
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
    Spacer(Modifier.height(24.dp))
    Text(
        "Havolani brauzerda ochib parolni o'zgartirishingiz mumkin. Yoki " +
            "havolani nusxalab shu yerga yopishtiring.",
        style = MaterialTheme.typography.bodySmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
    Spacer(Modifier.height(16.dp))

    Button(
        onClick = { vm.goToConfirm() },
        modifier = Modifier.fillMaxWidth().widthIn(max = 420.dp).height(52.dp),
    ) { Text("Havolani yopishtirish") }

    Spacer(Modifier.height(8.dp))
    TextButton(onClick = { vm.goToRequest() }) { Text("Boshqa email kiritish") }
}

@Composable
private fun ConfirmStep(state: PasswordResetUiState, vm: PasswordResetViewModel) {
    val errors = state.errors
    val show = state.showErrors

    Text("Yangi parol", style = MaterialTheme.typography.headlineSmall)
    Spacer(Modifier.height(8.dp))
    Text(
        "Emaildagi havolani to'liq yopishtiring — kodni o'zimiz ajratib olamiz.",
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
    Spacer(Modifier.height(20.dp))

    OutlinedTextField(
        value = state.input.token,
        onValueChange = { v -> vm.edit { it.copy(token = v) } },
        label = { Text("Havola yoki kod") },
        isError = show && errors.token != null,
        supportingText = { if (show) errors.token?.let { Text(it) } },
        minLines = 2,
        maxLines = 3,
        modifier = Modifier.fillMaxWidth().widthIn(max = 420.dp),
    )
    Spacer(Modifier.height(12.dp))

    OutlinedTextField(
        value = state.input.password,
        onValueChange = { v -> vm.edit { it.copy(password = v) } },
        label = { Text("Yangi parol") },
        isError = show && errors.password != null,
        supportingText = {
            Text((if (show) errors.password else null) ?: "Kamida ${ProfileForm.PASSWORD_MIN} belgi")
        },
        singleLine = true,
        visualTransformation = PasswordVisualTransformation(),
        keyboardOptions = KeyboardOptions(
            keyboardType = KeyboardType.Password,
            imeAction = ImeAction.Next,
        ),
        modifier = Modifier.fillMaxWidth().widthIn(max = 420.dp),
    )
    Spacer(Modifier.height(12.dp))

    OutlinedTextField(
        value = state.input.confirm,
        onValueChange = { v -> vm.edit { it.copy(confirm = v) } },
        label = { Text("Parolni takrorlang") },
        isError = show && errors.confirm != null,
        supportingText = { if (show) errors.confirm?.let { Text(it) } },
        singleLine = true,
        visualTransformation = PasswordVisualTransformation(),
        keyboardOptions = KeyboardOptions(
            keyboardType = KeyboardType.Password,
            imeAction = ImeAction.Done,
        ),
        modifier = Modifier.fillMaxWidth().widthIn(max = 420.dp),
    )
    Spacer(Modifier.height(20.dp))

    Button(
        onClick = { vm.confirm() },
        enabled = !state.submitting,
        modifier = Modifier.fillMaxWidth().widthIn(max = 420.dp).height(52.dp),
    ) {
        if (state.submitting) {
            CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp)
        } else {
            Text("Parolni o'zgartirish")
        }
    }

    state.error?.let {
        Spacer(Modifier.height(16.dp))
        Text(it, color = MaterialTheme.colorScheme.error)
    }
}
