@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.room

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.FiberManualRecord
import androidx.compose.material.icons.filled.Poll
import androidx.compose.material.icons.filled.Stop
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * «Ko'proq» paneli — boshqaruv panelida joyi qolmagan amallar.
 *
 * ## Nega alohida panel, yangi tugmalar emas
 * Pastdagi panel telefon portret rejimida allaqachon chekkasida edi (qurilma
 * sinovida 7 ta tugma zo'rg'a sig'gan, QA topilmasi). Har yangi imkoniyat
 * uchun tugma qo'shish yo'li oxir-oqibat o'qib bo'lmaydigan panelga olib
 * borardi. Shu sabab kam ishlatiladigan (lekin kerakli) amallar bitta
 * «Ko'proq» ostiga yig'iladi — Zoom ham aynan shunday qiladi.
 */
@Composable
fun MoreSheet(
    onDismiss: () -> Unit,
    onReaction: (String) -> Unit,
    onOpenPoll: () -> Unit,
    /** Hozir yozib olinyaptimi (Record tugmasi holati). */
    recording: Boolean,
    /** Yozuvni boshlash mumkinmi (ekran ulashilganmi). */
    canRecord: Boolean,
    onToggleRecording: () -> Unit,
    autoBackground: Boolean,
    onAutoBackgroundChange: (Boolean) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(Modifier.padding(horizontal = 16.dp).padding(bottom = 24.dp)) {
            Text("Reaksiya", style = MaterialTheme.typography.titleMedium)
            Text(
                "Xonadagilarning hammasi ko'radi. Saqlanmaydi — bir necha soniyada yo'qoladi.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(10.dp))
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceEvenly,
            ) {
                // Ro'yxat KLIENT konstantasi va server uni ustidan tekshiradi —
                // qarang: `Reactions`. Panel bosilgandan keyin YOPILADI: ustoz
                // reaksiyani odatda bir marta yuboradi va ochiq qolgan panel
                // sahnani to'sib turardi.
                Reactions.ALLOWED.forEach { emoji ->
                    Box(
                        Modifier
                            .size(44.dp)
                            .clip(CircleShape)
                            .background(MaterialTheme.colorScheme.surfaceVariant)
                            .clickable {
                                onReaction(emoji)
                                onDismiss()
                            },
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(emoji, fontSize = 22.sp)
                    }
                }
            }

            // ⭐ RECORD — dars yozib olishni QO'LDA boshqarish (Zoom kabi).
            //
            // Avto-yozuv dars sozlamasida yoniq bo'lsa yozuv o'zi boshlanadi, lekin
            // ustoz istagan payt shu tugma bilan to'xtatishi/qayta boshlashi mumkin.
            // Yozuv EKRANni yozadi — shuning uchun ekran ulashilmagan bo'lsa tugma
            // o'chiq (bosilganda RoomViewModel ustozga "avval ekran ulashing" deydi).
            Spacer(Modifier.height(20.dp))
            Text("Dars yozib olish", style = MaterialTheme.typography.titleMedium)
            Text(
                if (recording) "Hozir yozib olinyapti — telefoningizda to'liq sifatda."
                else "Ekranni to'liq sifatda yozadi. Tugagach Arxivга tushadi.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(8.dp))
            Button(
                onClick = {
                    onToggleRecording()
                    onDismiss()
                },
                enabled = recording || canRecord,
                colors = if (recording) {
                    ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error)
                } else {
                    ButtonDefaults.buttonColors()
                },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Icon(
                    if (recording) Icons.Default.Stop else Icons.Default.FiberManualRecord,
                    contentDescription = null,
                    modifier = Modifier.size(18.dp),
                )
                Spacer(Modifier.width(8.dp))
                Text(if (recording) "Yozib olishni to'xtatish" else "Yozib olishni boshlash")
            }

            Spacer(Modifier.height(20.dp))
            Text("Dars vositalari", style = MaterialTheme.typography.titleMedium)
            Spacer(Modifier.height(8.dp))
            OutlinedButton(
                onClick = {
                    onDismiss()
                    onOpenPoll()
                },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Icon(Icons.Default.Poll, contentDescription = null, modifier = Modifier.size(18.dp))
                Spacer(Modifier.width(8.dp))
                Text("So'rovnoma")
            }

            // ⭐ №25 — ULASHISHDA ILOVA FONGA O'TSINMI.
            //
            // Sozlama aynan SHU YERDA: u dars ichida, ulashishdan bir necha
            // soniya oldin kerak bo'ladi. Kabinetdagi umumiy sozlamalarga
            // qo'yilsa ustoz uni dars o'rtasida topa olmasdi (dars ekranidan
            // chiqish esa "Yakunlaysizmi?" savolidan o'tadi).
            Spacer(Modifier.height(20.dp))
            Text("Ekran ulashish", style = MaterialTheme.typography.titleMedium)
            Spacer(Modifier.height(8.dp))
            Row(
                Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(Modifier.weight(1f)) {
                    Text("Ulashganda ilova fonga o'tsin", style = MaterialTheme.typography.bodyMedium)
                    Text(
                        "Zoom kabi: efir boshlanishi bilan Jonly yig'iladi va siz " +
                            "materialingizda qolasiz. O'chirsangiz ilova ekranda qoladi " +
                            "va o'quvchilar uni ko'radi.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                Spacer(Modifier.width(12.dp))
                Switch(checked = autoBackground, onCheckedChange = onAutoBackgroundChange)
            }
        }
    }
}
