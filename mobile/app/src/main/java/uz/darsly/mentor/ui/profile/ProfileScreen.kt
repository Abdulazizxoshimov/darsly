@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.profile

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
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
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material.icons.filled.Block
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.Icon
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
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
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.hilt.navigation.compose.hiltViewModel
import uz.darsly.mentor.BuildConfig
import uz.darsly.mentor.data.api.User
import uz.darsly.mentor.ui.common.rememberHapticClick
import uz.darsly.mentor.util.LessonFormat

/**
 * Shaxsiy kabinet — sozlamalar-ro'yxati uslubi (iOS/Android naqshi).
 *
 * Yuqorida profil boshi (avatar, ism, email, rol + "Ismni tahrirlash"), pastida
 * guruhlangan qatorlar: **Moderatsiya** (qora ro'yxat) va **Xavfsizlik** (parol,
 * chiqish). Ism ALOHIDA dialogда tahrirlanadi (takror yo'q).
 *
 * Mentor o'zini O'CHIRA olmaydi va MOBILDAN o'chirish SO'ROVI ham yubormaydi
 * (2026-08-15): hisob o'chirish so'rovi FAQAT web'da (mentor web'ga kirganда).
 * Hisoblarni admin boshqaradi (admin paneli, `DELETE /users/:id`).
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
    var nameOpen by rememberSaveable { mutableStateOf(false) }

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
                ProfileHeader(user = user, onEditName = { nameOpen = true })

                SettingsSection("Moderatsiya") {
                    SettingsRow(
                        icon = Icons.Default.Block,
                        title = "Qora ro'yxat",
                        onClick = onOpenBlocklist,
                    )
                }

                SettingsSection("Xavfsizlik") {
                    SettingsRow(
                        icon = Icons.Default.Lock,
                        title = "Parolni o'zgartirish",
                        onClick = { passwordOpen = true },
                    )
                    HorizontalDivider()
                    SettingsRow(
                        icon = Icons.AutoMirrored.Filled.Logout,
                        title = "Tizimdan chiqish",
                        onClick = { confirmLogout = true },
                        destructive = true,
                        loading = state.loggingOut,
                    )
                }

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

    // Ism saqlangach dialog o'zi yopiladi (parol naqshi kabi).
    LaunchedEffect(state.profileSaved) {
        if (state.profileSaved) {
            nameOpen = false
            vm.profileSavedHandled()
        }
    }

    if (nameOpen) {
        EditNameDialog(
            state = state,
            vm = vm,
            onDismiss = {
                nameOpen = false
                // Saqlanmagan o'zgarishni bekor qilamiz — keyingi ochilishда joriy ism ko'rinsin.
                state.user?.let { u -> vm.edit { it.copy(fullName = u.fullName) } }
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

}

/**
 * Profil "boshi" — kim ekanligingiz + "Ismni tahrirlash" (alohida dialog).
 *
 * Email va rol ATAYLAB o'zgartirilmaydi: email uchun endpoint yo'q, rolni server
 * `PUT /users/me` da majburan tashlaydi (privilege escalation himoyasi,
 * `v1/user.go`). Faqat ism tahrirlanadi — u ham alohida dialogда (takror yo'q).
 */
@Composable
private fun ProfileHeader(user: User, onEditName: () -> Unit) {
    Card(Modifier.fillMaxWidth()) {
        Column(
            Modifier.fillMaxWidth().padding(20.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Avatar(user)
            Spacer(Modifier.height(12.dp))
            Text(user.fullName, style = MaterialTheme.typography.titleLarge)
            Spacer(Modifier.height(2.dp))
            Text(
                user.email,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(10.dp))
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                RoleBadge(user.role)
                LessonFormat.scheduleLabel(user.createdAt)?.let {
                    Text(
                        "Ro'yxatdan: $it",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
            Spacer(Modifier.height(16.dp))
            OutlinedButton(onClick = rememberHapticClick(onClick = onEditName)) {
                Text("Ismni tahrirlash")
            }
        }
    }
}

/** Sozlamalar-ro'yxati bo'limi: SARLAVHA + kartada qatorlar (iOS/Android naqshi). */
@Composable
private fun SettingsSection(title: String, content: @Composable ColumnScope.() -> Unit) {
    Column {
        Text(
            title.uppercase(),
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(start = 4.dp, bottom = 6.dp),
        )
        Card(Modifier.fillMaxWidth()) {
            Column(content = content)
        }
    }
}

/** Bosiladigan sozlama qatori: ikonka + matn + chevron (yoki spinner). */
@Composable
private fun SettingsRow(
    icon: ImageVector,
    title: String,
    onClick: () -> Unit,
    destructive: Boolean = false,
    loading: Boolean = false,
) {
    val color = if (destructive) {
        MaterialTheme.colorScheme.error
    } else {
        MaterialTheme.colorScheme.onSurface
    }
    val click = rememberHapticClick(onClick = onClick)
    Row(
        Modifier
            .fillMaxWidth()
            .clickable(enabled = !loading) { click() }
            .padding(horizontal = 16.dp, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        Icon(icon, contentDescription = null, tint = color, modifier = Modifier.size(22.dp))
        Text(
            title,
            color = color,
            style = MaterialTheme.typography.bodyLarge,
            modifier = Modifier.weight(1f),
        )
        if (loading) {
            CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
        } else {
            Icon(
                Icons.Default.ChevronRight,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

/** Ismni tahrirlash — alohida dialog (Saqlagach o'zi yopiladi, `profileSaved`). */
@Composable
private fun EditNameDialog(state: ProfileUiState, vm: ProfileViewModel, onDismiss: () -> Unit) {
    AlertDialog(
        onDismissRequest = { if (!state.saving) onDismiss() },
        title = { Text("Ismni tahrirlash") },
        text = {
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
        },
        confirmButton = {
            TextButton(onClick = { vm.save() }, enabled = state.canSave) {
                if (state.saving) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text("Saqlash")
                }
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss, enabled = !state.saving) { Text("Bekor qilish") }
        },
    )
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
        modifier = Modifier.size(64.dp),
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
