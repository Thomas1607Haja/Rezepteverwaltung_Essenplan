package io.github.thomas1607haja.rezeptplaner.ui.photo

import android.content.Context
import android.net.Uri
import android.util.Log
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.google.android.gms.tasks.Task
import com.google.mlkit.common.MlKitException
import com.google.mlkit.vision.common.InputImage
import com.google.mlkit.vision.text.TextRecognition
import com.google.mlkit.vision.text.latin.TextRecognizerOptions
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.RezeptplanerApplication
import io.github.thomas1607haja.rezeptplaner.api.Ingredient
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.Step
import io.github.thomas1607haja.rezeptplaner.api.UrlDraft
import io.github.thomas1607haja.rezeptplaner.data.ImageChange
import io.github.thomas1607haja.rezeptplaner.data.Outcome
import io.github.thomas1607haja.rezeptplaner.ui.common.Texts
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

/** Zustand des Foto-Imports. [anchor]: zuletzt angetippte Zeile (Anfang eines Bereichs). */
data class PhotoState(
    val lines: List<OcrLine> = emptyList(),
    val kinds: List<LineKind> = emptyList(),
    val pages: Int = 0,
    val brush: LineKind = LineKind.Ingredient,
    val anchor: Int? = null,
    val dish: ImageChange.Upload? = null,
    val busy: String? = null,
    val error: String? = null,
)

/**
 * Foto-Import (7.7): Seiten fotografieren, Text auf dem Handy erkennen,
 * Zeilen zuordnen. „Weiter“ lässt die Zutaten vom Server zerlegen und öffnet
 * den Entwurf im Bearbeiten-Bildschirm (wie beim URL-Import).
 */
class PhotoImportViewModel(private val app: RezeptplanerApplication) : ViewModel() {
    private val _state = MutableStateFlow(PhotoState())
    val state: StateFlow<PhotoState> = _state.asStateFlow()

    fun addPages(uris: List<Uri>) {
        if (uris.isEmpty()) return
        _state.update { it.copy(busy = Texts.get(R.string.photo_recognizing), error = null) }
        viewModelScope.launch {
            var error: String? = null
            for (uri in uris) {
                val page = _state.value.pages
                try {
                    val found = recognizeText(app, uri, page)
                    if (found.isEmpty()) error = Texts.get(R.string.photo_no_text)
                    _state.update { s ->
                        val all = s.lines + found
                        // Vorschläge nur für die neuen Zeilen; Korrekturen bleiben.
                        s.copy(lines = all, kinds = s.kinds + guessKinds(all).drop(s.lines.size), pages = page + 1)
                    }
                } catch (e: Exception) {
                    error = ocrMessage(e)
                }
            }
            _state.update { it.copy(busy = null, error = error) }
        }
    }

    fun setBrush(kind: LineKind) = _state.update { it.copy(brush = kind) }

    /** Antippen: Zeile bekommt die gewählte Art. */
    fun tap(index: Int) = _state.update { s ->
        s.copy(kinds = s.kinds.mapIndexed { i, k -> if (i == index) s.brush else k }, anchor = index)
    }

    /** Lange drücken: alle Zeilen von der zuletzt angetippten bis hierher. */
    fun markRange(index: Int) = _state.update { s ->
        val from = s.anchor ?: index
        val range = minOf(from, index)..maxOf(from, index)
        s.copy(kinds = s.kinds.mapIndexed { i, k -> if (i in range) s.brush else k }, anchor = index)
    }

    fun setDish(image: ImageChange.Upload?) = _state.update { it.copy(dish = image) }

    fun clear() = _state.update { PhotoState(brush = it.brush) }

    /** Entwurf bauen und an den Bearbeiten-Bildschirm übergeben. */
    fun next(onDraft: () -> Unit) {
        val s = _state.value
        val draft = buildDraft(s.lines, s.kinds)
        if (draft.title.isEmpty() && draft.ingredientLines.isEmpty() && draft.steps.isEmpty()) {
            _state.update { it.copy(error = Texts.get(R.string.photo_mark_first)) }
            return
        }
        _state.update { it.copy(busy = Texts.get(R.string.photo_parsing), error = null) }
        viewModelScope.launch {
            val notes = mutableListOf(Texts.get(R.string.photo_note))
            val ingredients = if (draft.ingredientLines.isEmpty()) emptyList() else {
                when (val r = app.recipeRepository.parseIngredients(draft.ingredientLines)) {
                    is Outcome.Ok -> r.value
                    // Ohne Server: Zeilen unzerlegt als Zutatennamen.
                    is Outcome.Error -> {
                        notes += Texts.get(R.string.photo_not_parsed, "message" to r.message)
                        draft.ingredientLines.map { Ingredient(name = it.take(200)) }
                    }
                }
            }
            val recipe = Recipe(
                id = 0,
                title = draft.title.take(200),
                baseServings = app.localStore.snapshot.value.settings.familySize,
                ingredients = ingredients,
                steps = draft.steps.map { Step(it) },
                notes = notes.joinToString(" "),
            )
            app.recipeRepository.pendingDraft = UrlDraft(recipe)
            app.recipeRepository.pendingImage = s.dish
            _state.update { it.copy(busy = null) }
            onDraft()
        }
    }
}

/** Texterkennung auf dem Gerät (ML Kit über die Play-Dienste, lateinische Schrift). */
suspend fun recognizeText(context: Context, uri: Uri, page: Int): List<OcrLine> {
    // fromFilePath berücksichtigt die Ausrichtung aus den EXIF-Daten.
    val image = InputImage.fromFilePath(context, uri)
    val recognizer = TextRecognition.getClient(TextRecognizerOptions.DEFAULT_OPTIONS)
    try {
        val started = System.currentTimeMillis()
        val text = recognizer.process(image).await()
        val lines = text.textBlocks.flatMapIndexed { b, block -> block.lines.map { OcrLine(it.text, page, b) } }
        // Messwert für docs/PLANUNG.md (Messung nach Phase 7).
        Log.i("Rezeptplaner", "Texterkennung: ${lines.size} Zeilen in ${System.currentTimeMillis() - started} ms (${image.width}×${image.height})")
        return lines
    } finally {
        recognizer.close()
    }
}

private fun ocrMessage(e: Exception): String = when {
    e is MlKitException && e.errorCode == MlKitException.UNAVAILABLE ->
        Texts.get(R.string.photo_ocr_loading)
    e is java.io.IOException -> Texts.get(R.string.photo_open_failed)
    else -> Texts.get(R.string.photo_ocr_failed, "message" to (e.message ?: e.javaClass.simpleName))
}

private suspend fun <T> Task<T>.await(): T = suspendCancellableCoroutine { cont ->
    addOnSuccessListener { cont.resume(it) }
    addOnFailureListener { cont.resumeWithException(it) }
    addOnCanceledListener { cont.cancel() }
}
