package uz.darsly.mentor.ui.common

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import uz.darsly.mentor.util.NotificationFormat

/**
 * O'ng-tepadagi qo'ng'iroq — o'qilmagan bildirishnomalar badge'i bilan.
 *
 * Bildirishnomalar pastki panelda ALOHIDA bo'lim emas (ko'p ilovalardagi
 * naqsh): asosiy ekranlarning yuqori panelida shu qo'ng'iroq turadi, bosilsa
 * to'liq ekran bildirishnomalar ochiladi.
 */
@Composable
fun NotificationBell(unread: Int, onClick: () -> Unit) {
    IconButton(onClick = onClick) {
        BadgedBox(
            badge = { NotificationFormat.badgeLabel(unread)?.let { Badge { Text(it) } } },
        ) {
            Icon(Icons.Default.Notifications, contentDescription = "Bildirishnomalar")
        }
    }
}
