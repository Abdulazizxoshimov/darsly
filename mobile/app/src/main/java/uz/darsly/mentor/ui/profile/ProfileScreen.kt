@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.profile

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.foundation.BorderStroke
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
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
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.hilt.navigation.compose.hiltViewModel
import uz.darsly.mentor.BuildConfig
import uz.darsly.mentor.data.api.User
import uz.darsly.mentor.util.LessonFormat
import java.time.ZoneId

/**
 * Shaxsiy kabinet.
 *
 * Uch bo'lim: **kim ekanligingiz** (o'zgarmas — email, rol) → **sozlamalar**
 * (ism, til, mintaqa) → **xavfsizlik va chiqish**. Tartib ataylab shunday:
 * eng ko'p qaraladigan ma'lumot yuqorida, eng xavfli amal (chiqish) pastda.
 */
@Composable
fun ProfileScreen(
    onOpenBlocklist: () -> Unit,
    vm: ProfileViewModel = hiltViewModel(),
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val snackbar = remember { SnackbarHostState() }
    var passwordOpen by rememberSaveable { mutableStateOf(false) }
    var confirmLogout by rememberSaveable { mutableStateOf(false) }
    var confirmDelete by rememberSaveable { mutableStateOf(false) }

    LaunchedEffect(Unit) { vm.start() }
    LaunchedEffect(state.notice) {
        state.notice?.let {
            snackbar.showSnackbar(it)
            vm.noticeShown()
        }
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbar) },
        topBar = { TopAppBar(title = { Text("Shaxsiy kabinet") }) },
    ) { padding ->
        val user = state.user
        when {
            state.loading && user == null -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) { CircularProgressIndicator() }

            user == null -> Column(
                Modifier.fillMaxSize().padding(padding).padding(24.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(
                    state.error ?: "Profil yuklanmadi",
                    color = MaterialTheme.colorScheme.error,
                )
                Spacer(Modifier.height(12.dp))
                Button(onClick = { vm.load() }) { Text("Qayta urinish") }
            }

            else -> Column(
                Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                IdentityCard(user)
                SettingsCard(state, vm)
                ModerationCard(onOpenBlocklist = onOpenBlocklist)
                SecurityCard(
                    loggingOut = state.loggingOut,
                    onChangePassword = { passwordOpen = true },
                    onLogout = { confirmLogout = true },
                )
                DangerZoneCard(
                    deleting = state.deletingAccount,
                    onDeleteAccount = { confirmDelete = true },
                )
                AppVersion()
            }
        }
    }

    // Muvaffaqiyatda VM aniq signal beradi — oyna shunga qarab yopiladi.
    LaunchedEffect(state.passwordChanged) {
        if (state.passwordChanged) {
            passwordOpen = false
            vm.passwordChangeHandled()
        }
    }

    if (passwordOpen) {
        ChangePasswordDialog(
            state = state,
            vm = vm,
            onDismiss = {
                passwordOpen = false
                // Kiritilgan parollar ViewModel xotirasida qolib ketmasin.
                vm.resetPasswordForm()
            },
        )
    }

    if (confirmLogout) {
        AlertDialog(
            onDismissRequest = { confirmLogout = false },
            title = { Text("Tizimdan chiqasizmi?") },
            text = {
                Text(
                    "Saqlangan darslar ro'yxati ham telefondan o'chiriladi. " +
                        "Qaytib kirish uchun email va parol kerak bo'ladi.",
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    confirmLogout = false
                    vm.logout()
                }) { Text("Chiqish", color = MaterialTheme.colorScheme.error) }
            },
            dismissButton = {
                TextButton(onClick = { confirmLogout = false }) { Text("Bekor qilish") }
            },
        )
    }

    // M5 — akkauntni o'chirish tasdiqi. Qaytarib bo'lmaydigan amal, shuning uchun
    // ogohlantirish aniq va tugma matni "O'chirish" (adashib bosilmasin).
    if (confirmDelete) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            title = { Text("Akkauntni o'chirasizmi?") },
            text = {
                Text(
                    "Bu amalni QAYTARIB BO'LMAYDI. Hisobingiz va unga bog'liq ma'lumotlar " +
                        "o'chiriladi, barcha darslaringiz havolalari ishlamay qoladi. " +
                        "Qaytadan foydalanish uchun yangi hisob ochishingiz kerak bo'ladi.",
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    confirmDelete = false
                    vm.deleteAccount()
                }) { Text("O'chirish", color = MaterialTheme.colorScheme.error) }
            },
            dismissButton = {
                TextButton(onClick = { confirmDelete = false }) { Text("Bekor qilish") }
            },
        )
    }
}

/**
 * Kim ekanligingiz — **o'zgartirib bo'lmaydigan** maydonlar.
 *
 * Email va rol ataylab tahrirlanmaydi: email o'zgartirish endpoint'i backend'da
 * yo'q, rolni esa server `PUT /users/me` da majburan tashlab yuboradi (privilege
 * escalation himoyasi, `v1/user.go:198`). Tahrirlanadigandek ko'rinadigan, lekin
 * jimgina ishlamaydigan maydon eng yomon interfeys yolg'oni bo'lardi.
 */
@Composable
private fun IdentityCard(user: User) {
    Card(Modifier.fillMaxWidth()) {
        Row(
            Modifier.padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(14.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Avatar(user)
            Column(Modifier.weight(1f)) {
                Text(user.fullName, style = MaterialTheme.typography.titleMedium)
                Text(
                    user.email,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Spacer(Modifier.height(6.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    RoleBadge(user.role)
                    LessonFormat.scheduleLabel(user.createdAt)?.let {
                        Text(
                            "Ro'yxatdan o'tgan: $it",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
            }
        }
    }
}

/**
 * Ism bosh harflaridan avatar.
 *
 * Rasm yuklash bu bosqichda yo'q (backend `avatar_url` ni faqat qabul qiladi,
 * fayl yuklash endpoint'i yo'q), shuning uchun web'dagi kabi harfli doira —
 * bo'sh kulrang doiradan ko'ra ma'noliroq va tanib olinadi.
 */
@Composable
private fun Avatar(user: User) {
    Surface(
        shape = CircleShape,
        color = MaterialTheme.colorScheme.primary,
        modifier = Modifier.size(52.dp),
    ) {
        Box(contentAlignment = Alignment.Center) {
            Text(
                ProfileForm.initials(user.fullName),
                style = MaterialTheme.typography.titleMedium,
                color = MaterialTheme.colorScheme.onPrimary,
            )
        }
    }
}

@Composable
private fun RoleBadge(role: String) {
    Surface(
        shape = MaterialTheme.shapes.small,
        color = MaterialTheme.colorScheme.surfaceVariant,
    ) {
        Text(
            ProfileForm.roleLabel(role),
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp),
        )
    }
}

@Composable
private fun SettingsCard(state: ProfileUiState, vm: ProfileViewModel) {
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text("Ma'lumotlar", style = MaterialTheme.typography.titleSmall)

            OutlinedTextField(
                value = state.input.fullName,
                onValueChange = { v -> vm.edit { it.copy(fullName = v) } },
                label = { Text("Ism familiya") },
                isError = state.showErrors && state.errors.fullName != null,
                supportingText = { if (state.showErrors) state.errors.fullName?.let { Text(it) } },
                singleLine = true,
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                modifier = Modifier.fillMaxWidth(),
            )

            // TIL VA VAQT MINTAQASI SOZLAMADAN OLIB TASHLANDI.
            //
            // Ilova butunlay o'zbekcha va foydalanuvchilar O'zbekistonda — ya'ni
            // bu ikki maydon hech qachon o'zgartirilmaydi, lekin har bir ustozga
            // "nimadir sozlash kerakmi?" degan savol tug'dirardi. Qiymatlar
            // `ProfileForm.DEFAULT_LANGUAGE` / `DEFAULT_TIMEZONE` dan saqlanadi.
            Text(
                "Dars vaqtlari Toshkent vaqtida ko'rsatiladi",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            Button(
                onClick = { vm.save() },
                enabled = state.canSave,
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (state.saving) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text("Saqlash")
                }
            }
        }
    }
}

/** Ochiladigan ro'yxatli maydon — erkin matn o'rniga yopiq tanlov. */
@Composable
private fun PickerField(
    label: String,
    value: String,
    options: List<Pair<String, String>>,
    onSelect: (String) -> Unit,
) {
    var open by remember { mutableStateOf(false) }
    Column(Modifier.fillMaxWidth()) {
        Text(
            label,
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(4.dp))
        Box {
            OutlinedButton(onClick = { open = true }, modifier = Modifier.fillMaxWidth()) {
                Text(value, modifier = Modifier.weight(1f))
                Text("▾")
            }
            DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
                options.forEach { (code, title) ->
                    DropdownMenuItem(
                        text = { Text(title) },
                        onClick = {
                            open = false
                            onSelect(code)
                        },
                    )
                }
            }
        }
    }
}

/**
 * Moderatsiya — hisobga tegishli (darsga emas) sozlamalar.
 *
 * Qora ro'yxat aynan SHU YERDA, xona ekranida emas: ban dars tugagandan keyin
 * ham kuchda qoladi va uni qaytarish odatda boshqa kuni, sovuq boshda esga
 * tushadi. Xona ichida bo'lsa, ustoz uni faqat dars paytida topa olardi.
 */
@Composable
private fun ModerationCard(onOpenBlocklist: () -> Unit) {
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("Moderatsiya", style = MaterialTheme.typography.titleSmall)
            Text(
                "Doimiy ban qo'yilgan ishtirokchilar ro'yxati — qaytarib olish uchun",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            OutlinedButton(onClick = onOpenBlocklist, modifier = Modifier.fillMaxWidth()) {
                Text("Qora ro'yxat")
            }
        }
    }
}

@Composable
private fun SecurityCard(
    loggingOut: Boolean,
    onChangePassword: () -> Unit,
    onLogout: () -> Unit,
) {
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("Xavfsizlik", style = MaterialTheme.typography.titleSmall)
            OutlinedButton(onClick = onChangePassword, modifier = Modifier.fillMaxWidth()) {
                Text("Parolni o'zgartirish")
            }
            HorizontalDivider()
            OutlinedButton(
                onClick = onLogout,
                enabled = !loggingOut,
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (loggingOut) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text("Tizimdan chiqish", color = MaterialTheme.colorScheme.error)
                }
            }
        }
    }
}

/**
 * M5 — "Xavfli hudud": akkauntni o'chirish (Play Store majburiyati).
 * Xavfsizlik kartasidan ATAYLAB ajratilgan, chunki bu qaytarib bo'lmaydigan amal.
 */
@Composable
private fun DangerZoneCard(
    deleting: Boolean,
    onDeleteAccount: () -> Unit,
) {
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("Akkaunt", style = MaterialTheme.typography.titleSmall)
            Text(
                "Akkauntni butunlay o'chirish — qaytarib bo'lmaydi.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            OutlinedButton(
                onClick = onDeleteAccount,
                enabled = !deleting,
                colors = ButtonDefaults.outlinedButtonColors(
                    contentColor = MaterialTheme.colorScheme.error,
                ),
                border = BorderStroke(1.dp, MaterialTheme.colorScheme.error),
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (deleting) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text("Akkauntni o'chirish")
                }
            }
        }
    }
}

@Composable
private fun AppVersion() {
    Text(
        "Jonly Mentor · versiya ${BuildConfig.VERSION_NAME}",
        style = MaterialTheme.typography.labelSmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp),
    )
}

@Composable
private fun ChangePasswordDialog(
    state: ProfileUiState,
    vm: ProfileViewModel,
    onDismiss: () -> Unit,
) {
    val errors = state.passwordErrors
    val show = state.showPasswordErrors

    AlertDialog(
        onDismissRequest = { if (!state.changingPassword) onDismiss() },
        title = { Text("Parolni o'zgartirish") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                OutlinedTextField(
                    value = state.passwordInput.current,
                    onValueChange = { v -> vm.editPassword { it.copy(current = v) } },
                    label = { Text("Joriy parol") },
                    isError = show && errors.current != null,
                    supportingText = { if (show) errors.current?.let { Text(it) } },
                    singleLine = true,
                    visualTransformation = PasswordVisualTransformation(),
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Password,
                        imeAction = ImeAction.Next,
                    ),
                )
                OutlinedTextField(
                    value = state.passwordInput.new,
                    onValueChange = { v -> vm.editPassword { it.copy(new = v) } },
                    label = { Text("Yangi parol") },
                    isError = show && errors.new != null,
                    supportingText = {
                        Text(
                            (if (show) errors.new else null)
                                ?: "Kamida ${ProfileForm.PASSWORD_MIN} belgi",
                        )
                    },
                    singleLine = true,
                    visualTransformation = PasswordVisualTransformation(),
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Password,
                        imeAction = ImeAction.Next,
                    ),
                )
                OutlinedTextField(
                    value = state.passwordInput.confirm,
                    onValueChange = { v -> vm.editPassword { it.copy(confirm = v) } },
                    label = { Text("Yangi parolni takrorlang") },
                    isError = show && errors.confirm != null,
                    supportingText = { if (show) errors.confirm?.let { Text(it) } },
                    singleLine = true,
                    visualTransformation = PasswordVisualTransformation(),
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Password,
                        imeAction = ImeAction.Done,
                    ),
                )
                state.passwordError?.let {
                    Text(it, color = MaterialTheme.colorScheme.error)
                }
            }
        },
        confirmButton = {
            TextButton(onClick = { vm.changePassword() }, enabled = state.canChangePassword) {
                if (state.changingPassword) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text("O'zgartirish")
                }
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss, enabled = !state.changingPassword) {
                Text("Bekor qilish")
            }
        },
    )
}
