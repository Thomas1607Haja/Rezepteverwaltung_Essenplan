package io.github.thomas1607haja.rezeptplaner.share

import android.content.ClipData
import android.content.Context
import android.content.Intent
import androidx.core.content.FileProvider
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.data.ImageStore
import io.github.thomas1607haja.rezeptplaner.data.Snapshot
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File

/** Ordner im Cache für geteilte PDFs (res/xml/file_paths.xml, Pfad „share“). */
private fun shareDir(context: Context) = File(context.cacheDir, "share")

/** Wie lange ein geteiltes PDF liegen bleibt: Die Ziel-App liest es meist sofort. */
private const val KEEP_MILLIS = 24 * 60 * 60 * 1000L

/** Löscht geteilte PDFs, die älter als 24 Stunden sind (beim Teilen und beim Start der App). */
fun cleanSharedFiles(context: Context, now: Long = System.currentTimeMillis()) {
    shareDir(context).listFiles()?.filter { now - it.lastModified() > KEEP_MILLIS }?.forEach { it.delete() }
}

/**
 * Erzeugt das PDF aus den Daten auf dem Handy (geht auch offline) und
 * liefert die Datei. Bild: das große, sonst das Vorschaubild; fehlen
 * beide, entsteht das PDF ohne Bild.
 */
suspend fun createRecipePdf(
    context: Context,
    recipe: Recipe,
    snapshot: Snapshot,
    images: ImageStore,
    withNotes: Boolean,
): File = withContext(Dispatchers.IO) {
    cleanSharedFiles(context)
    val dir = shareDir(context).apply { mkdirs() }
    val input = PdfInput(
        recipe = recipe,
        image = images.file(recipe.image) ?: images.file(recipe.thumbnail),
        withNotes = withNotes,
        parentTitle = recipe.parentRecipeId?.let { id -> snapshot.recipes.firstOrNull { it.id == id }?.title },
        courseGroup = snapshot.tagGroups.firstOrNull { it.role == "course" }?.name.orEmpty(),
    )
    val file = File(dir, shareFileName(recipe.title))
    val tmp = File(dir, file.name + ".tmp")
    try {
        tmp.outputStream().use { writeRecipePdf(input, it) }
        file.delete()
        check(tmp.renameTo(file)) { "PDF konnte nicht gespeichert werden" }
    } finally {
        tmp.delete()
    }
    file
}

/** Öffnet das Android-Teilen-Menü für das PDF (WhatsApp, Signal, E-Mail, Dateien …). */
fun sharePdf(context: Context, file: File, title: String) {
    val uri = FileProvider.getUriForFile(context, "${context.packageName}.files", file)
    val send = Intent(Intent.ACTION_SEND).apply {
        type = "application/pdf"
        putExtra(Intent.EXTRA_STREAM, uri)
        putExtra(Intent.EXTRA_SUBJECT, "Rezept: $title") // Betreff in E-Mail-Apps
        // ClipData: Leserecht und Vorschau im Teilen-Menü auch für Apps,
        // die die Datei nicht über EXTRA_STREAM öffnen.
        clipData = ClipData.newRawUri(file.name, uri)
        addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
    }
    context.startActivity(Intent.createChooser(send, "Rezept teilen"))
}
