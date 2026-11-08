package io.github.thomas1607haja.rezeptplaner.ui

import android.content.Intent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import io.github.thomas1607haja.rezeptplaner.CrashLog
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.ui.common.tr

/** Hinweis auf den letzten Absturz mit „Bericht teilen“; unsichtbar, wenn es keinen gab. */
@Composable
fun CrashReportCard() {
    val context = LocalContext.current
    var report by remember { mutableStateOf(CrashLog.read(context)) }
    val text = report ?: return

    Card(
        Modifier.fillMaxWidth(),
        border = CardDefaults.outlinedCardBorder(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.errorContainer),
    ) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(tr(R.string.settings_crash_title), style = MaterialTheme.typography.titleMedium)
            Text(
                tr(R.string.settings_crash_text),
                style = MaterialTheme.typography.bodySmall,
            )
            Text(text.lineSequence().drop(5).firstOrNull().orEmpty(), style = MaterialTheme.typography.bodySmall)
            val subject = tr(R.string.settings_crash_subject)
            val share = tr(R.string.settings_crash_share)
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = {
                    val send = Intent(Intent.ACTION_SEND).apply {
                        type = "text/plain"
                        putExtra(Intent.EXTRA_SUBJECT, subject)
                        putExtra(Intent.EXTRA_TEXT, text)
                    }
                    context.startActivity(Intent.createChooser(send, share))
                }) { Text(share) }
                OutlinedButton(onClick = { CrashLog.clear(context); report = null }) { Text(tr(R.string.settings_crash_discard)) }
            }
        }
    }
}
