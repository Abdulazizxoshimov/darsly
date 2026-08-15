package uz.darsly.mentor.ui.login

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.OutlinedTextField
import androidx.compose.ui.platform.LocalContext
import uz.darsly.mentor.BuildConfig
import uz.darsly.mentor.ui.common.rememberHapticClick
import uz.darsly.mentor.ui.theme.DarslyTheme
import uz.darsly.mentor.ui.theme.PulseMark
import uz.darsly.mentor.ui.theme.neonGlow
import uz.darsly.mentor.util.DevServer

@Composable
fun LoginScreen(
    onLoggedIn: () -> Unit,
    onForgotPassword: () -> Unit,
    vm: LoginViewModel = hiltViewModel(),
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val notice by vm.notice.collectAsStateWithLifecycle()

    // Debug build'da local.properties'dan kelgan test hisobi bilan oldindan to'ldiramiz
    // (sirlar kodda yo'q — BuildConfig'ga local.properties orqali tushadi).
    var email by rememberSaveable { mutableStateOf(BuildConfig.TEST_EMAIL) }
    var password by rememberSaveable { mutableStateOf(BuildConfig.TEST_PASSWORD) }

    LaunchedEffect(state.success) {
        if (state.success) onLoggedIn()
    }

    // M10: login — birinchi ekran. Yuqoridan yumshoq mint "jonli efir" nuri
    // (web auth-center'dagi radial-gradient bilan bir tilda). Brend belgisiga
    // nur QO'YILMAYDI — glow qoidasi (3 joy) buzilmasin; nur faqat "Kirish"
    // tugmasida (u — ruxsat etilgan "asosiy harakat tugmasi").
    Box(
        Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
            .background(
                Brush.verticalGradient(
                    0f to DarslyTheme.colors.neon.copy(alpha = 0.10f),
                    0.4f to Color.Transparent,
                )
            ),
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(24.dp),
            verticalArrangement = Arrangement.Center,
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
        // Brend belgisi — puls (B «Jonli efir» tili).
        PulseMark(size = 44.dp)
        Spacer(Modifier.height(14.dp))
        Text("Jonly Mentor", style = MaterialTheme.typography.headlineMedium)
        Spacer(Modifier.height(4.dp))
        Text(
            "versiya ${BuildConfig.VERSION_NAME}",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(32.dp))

        // ⭐ NEGA chiqarilgan edik (`SESSION_REVOKED`).
        //
        // Forma USTIDA turadi — pastdagi xato qatori bilan chalkashmasin: bu
        // hozirgi urinishning xatosi emas, o'tgan sessiyaning sababi. Yopish
        // tugmasi bor, chunki ustoz uni o'qib bo'lgach u faqat xalaqit beradi.
        notice?.let { text ->
            Surface(
                color = MaterialTheme.colorScheme.errorContainer,
                shape = RoundedCornerShape(12.dp),
                modifier = Modifier.fillMaxWidth().widthIn(max = 420.dp),
            ) {
                Row(
                    Modifier.padding(start = 14.dp, top = 10.dp, bottom = 10.dp, end = 4.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onErrorContainer,
                        modifier = Modifier.weight(1f),
                    )
                    IconButton(onClick = { vm.noticeShown() }) {
                        Icon(
                            Icons.Default.Close,
                            contentDescription = "Yopish",
                            tint = MaterialTheme.colorScheme.onErrorContainer,
                        )
                    }
                }
            }
            Spacer(Modifier.height(16.dp))
        }

        OutlinedTextField(
            value = email,
            onValueChange = { email = it },
            label = { Text("Email") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Email,
                imeAction = ImeAction.Next,
            ),
            modifier = Modifier.fillMaxWidth().widthIn(max = 420.dp),
        )
        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = password,
            onValueChange = { password = it },
            label = { Text("Parol") },
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Password,
                imeAction = ImeAction.Done,
            ),
            modifier = Modifier.fillMaxWidth().widthIn(max = 420.dp),
        )
        Spacer(Modifier.height(20.dp))

        val btnShape = MaterialTheme.shapes.medium
        Button(
            onClick = rememberHapticClick { vm.login(email.trim(), password) },
            enabled = !state.loading && email.isNotBlank() && password.isNotBlank(),
            shape = btnShape,
            modifier = Modifier
                .fillMaxWidth()
                .widthIn(max = 420.dp)
                .height(52.dp)
                // Asosiy harakat tugmasi — glow'ga ruxsat etilgan uch joydan biri.
                .neonGlow(color = DarslyTheme.colors.neon, shape = btnShape, elevation = 10.dp),
        ) {
            if (state.loading) {
                CircularProgressIndicator(modifier = Modifier.height(20.dp))
            } else {
                Text("Kirish")
            }
        }

        // Parolni tiklash — xato chiqishidan OLDIN emas, keyin ham emas: har
        // doim ko'rinadi. Ustoz parolni unutganini "Kirish" bosgandan keyin
        // emas, formani ko'rgan zahoti eslaydi.
        Spacer(Modifier.height(8.dp))
        TextButton(onClick = onForgotPassword) { Text("Parolni unutdingizmi?") }

        state.error?.let {
            Spacer(Modifier.height(16.dp))
            Text(it, color = MaterialTheme.colorScheme.error)
        }

        // DEV: server manzilini shu yerdan almashtirish (faqat debug build).
        // Tarmoq/IP o'zgarganda APK qayta build qilinmaydi — qarang: DevServer.
        if (BuildConfig.DEBUG) {
            val ctx = LocalContext.current
            var serverDialogOpen by remember { mutableStateOf(false) }
            var serverBase by remember { mutableStateOf(DevServer.currentBase(ctx)) }
            Spacer(Modifier.height(24.dp))
            TextButton(onClick = { serverDialogOpen = true }) {
                Text(
                    "Server: $serverBase",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            if (serverDialogOpen) {
                var input by remember { mutableStateOf(serverBase) }
                var invalid by remember { mutableStateOf(false) }
                AlertDialog(
                    onDismissRequest = { serverDialogOpen = false },
                    title = { Text("Server manzili (dev)") },
                    text = {
                        Column {
                            OutlinedTextField(
                                value = input,
                                onValueChange = { input = it; invalid = false },
                                label = { Text("http://IP:8087") },
                                isError = invalid,
                                supportingText = {
                                    Text(
                                        if (invalid) "Yaroqsiz URL" else
                                            "Bo'sh qoldirilsa build'dagi manzilga qaytadi. Restart shart emas.",
                                    )
                                },
                                singleLine = true,
                            )
                        }
                    },
                    confirmButton = {
                        TextButton(onClick = {
                            if (DevServer.set(ctx, input)) {
                                serverBase = DevServer.currentBase(ctx)
                                serverDialogOpen = false
                            } else {
                                invalid = true
                            }
                        }) { Text("Saqlash") }
                    },
                    dismissButton = {
                        TextButton(onClick = { serverDialogOpen = false }) { Text("Bekor") }
                    },
                )
            }
        }
        }
    }
}
