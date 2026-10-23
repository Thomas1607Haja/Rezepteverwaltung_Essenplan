package io.github.thomas1607haja.rezeptplaner.ui.edit

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.data.Outcome
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import kotlinx.coroutines.launch

/** Erste Webadresse in einem geteilten Text („Schau mal: https://…“). */
fun extractUrl(text: String): String? = Regex("""https?://\S+""").find(text)?.value?.trimEnd('.', ',', ')', '"', '\'')

/**
 * Rezept aus einer Webadresse: Server liest die Seite, die App öffnet den
 * Entwurf zum Prüfen (gespeichert wird erst dort). [initialUrl] kommt vom
 * Teilen aus Browser oder Chefkoch-App; leer = Eingabefeld.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun UrlImportScreen(initialUrl: String, onDraft: () -> Unit, onClose: () -> Unit) {
    val app = app()
    val scope = rememberCoroutineScope()
    var url by rememberSaveable { mutableStateOf(initialUrl) }
    var busy by rememberSaveable { mutableStateOf(false) }
    var error by rememberSaveable { mutableStateOf<String?>(null) }

    fun load() {
        val target = extractUrl(url) ?: run { error = "Bitte eine Webadresse eingeben (https://…)."; return }
        busy = true
        error = null
        scope.launch {
            when (val r = app.recipeRepository.importUrl(target)) {
                is Outcome.Error -> error = r.message
                is Outcome.Ok -> {
                    app.recipeRepository.pendingDraft = r.value
                    onDraft()
                }
            }
            busy = false
        }
    }
    // Geteilter Link: gleich laden.
    LaunchedEffect(Unit) { if (initialUrl.isNotEmpty()) load() }

    Column(Modifier.fillMaxSize()) {
        TopAppBar(
            title = { Text("Rezept aus dem Internet") },
            navigationIcon = { IconButton(onClick = onClose) { Icon(painterResource(R.drawable.ic_back), contentDescription = "Zurück") } },
            windowInsets = WindowInsets(0),
        )
        Column(Modifier.fillMaxWidth().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text("Adresse eines Rezepts, z. B. von Chefkoch. Danach kannst du alles prüfen, bevor es gespeichert wird.")
            OutlinedTextField(
                url, { url = it; error = null }, label = { Text("Webadresse") }, singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri), modifier = Modifier.fillMaxWidth(),
            )
            error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Button(onClick = ::load, enabled = !busy && url.isNotBlank()) { Text("Einlesen") }
                if (busy) {
                    CircularProgressIndicator(Modifier.size(24.dp))
                    Text("Seite wird gelesen …")
                }
            }
        }
    }
}
