package io.github.thomas1607haja.rezeptplaner.ui.edit

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.github.thomas1607haja.rezeptplaner.RezeptplanerApplication
import io.github.thomas1607haja.rezeptplaner.api.DuplicateRecipe
import io.github.thomas1607haja.rezeptplaner.data.ImageChange
import io.github.thomas1607haja.rezeptplaner.data.Outcome
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * Zustand der Bearbeiten-Seite. [recipeId] 0 = neues Rezept; [fromDraft]:
 * Entwurf aus dem URL-Import (RecipeRepository.pendingDraft).
 */
class RecipeEditViewModel(
    private val app: RezeptplanerApplication,
    val recipeId: Long,
    fromDraft: Boolean,
) : ViewModel() {

    private val repo = app.recipeRepository
    private val existing = app.localStore.snapshot.value.recipes.firstOrNull { it.id == recipeId }
    private val draft = if (fromDraft) repo.pendingDraft.also { repo.pendingDraft = null } else null

    /** Formular beim Öffnen, um ungespeicherte Änderungen zu erkennen. */
    private val initial: RecipeForm = when {
        existing != null -> existing.toForm()
        draft != null -> draft.recipe.toForm()
        else -> RecipeForm(baseServings = app.localStore.snapshot.value.settings.familySize.toString())
    }

    private val _form = MutableStateFlow(initial)
    val form: StateFlow<RecipeForm> = _form.asStateFlow()

    /** Bild: aktuelles (Dateiname), neu gewähltes (Bytes) oder das der Quellseite. */
    val currentImage: String = existing?.thumbnail.orEmpty()
    private val initialImage: ImageChange = draft?.imageUrl?.takeIf { it.isNotEmpty() }?.let { ImageChange.FromUrl(it) } ?: ImageChange.Keep
    private val _image = MutableStateFlow(initialImage)
    val image: StateFlow<ImageChange> = _image.asStateFlow()

    val duplicates: List<DuplicateRecipe> = draft?.duplicates.orEmpty()

    private val _errors = MutableStateFlow<Map<String, String>>(emptyMap())
    val errors: StateFlow<Map<String, String>> = _errors.asStateFlow()
    private val _message = MutableStateFlow<String?>(null)
    val message: StateFlow<String?> = _message.asStateFlow()
    private val _saving = MutableStateFlow(false)
    val saving: StateFlow<Boolean> = _saving.asStateFlow()

    val isNew: Boolean get() = existing == null
    /** Ungespeicherte Änderungen? (Beim Entwurf zählt schon das Anlegen.) */
    val changed: Boolean get() = draft != null || _form.value != initial || _image.value != initialImage

    fun update(change: (RecipeForm) -> RecipeForm) = _form.update(change)

    fun updateItem(key: Long, change: (IngredientRow) -> IngredientRow) =
        _form.update { f -> f.copy(items = f.items.map { if (it.key == key) change(it) else it }) }

    fun setImage(change: ImageChange) {
        _image.value = change
    }

    private var pantryJob: Job? = null

    /**
     * Nach der Eingabe eines Zutatennamens: Steht er auf der Vorratsliste,
     * wird er nicht zum Einkaufen vorgemerkt (wie in der Weboberfläche).
     */
    fun checkPantry(key: Long, name: String) {
        pantryJob?.cancel()
        if (name.isBlank()) return
        pantryJob = viewModelScope.launch {
            delay(700)
            val inPantry = repo.pantryMatch(listOf(name.trim()))?.firstOrNull() ?: return@launch
            updateItem(key) { if (it.name == name) it.copy(shoppingRelevant = !inPantry) else it }
        }
    }

    /** Mehrere Zutatenzeilen einfügen; der Server zerlegt sie. */
    fun pasteLines(text: String) {
        val lines = text.lines().map { it.trim() }.filter { it.isNotEmpty() }
        if (lines.isEmpty()) return
        viewModelScope.launch {
            when (val r = repo.parseIngredients(lines)) {
                is Outcome.Error -> _message.value = r.message
                is Outcome.Ok -> _form.update { f ->
                    val rows = r.value.map {
                        IngredientRow(quantity = quantityText(it.quantity), unit = it.unit, name = it.name, note = it.note, shoppingRelevant = it.shoppingRelevant)
                    }
                    // Leere Startzeile ersetzen statt stehen lassen.
                    val kept = f.items.filterNot { !it.heading && it.name.isBlank() && it.quantity.isBlank() && it.unit.isBlank() && it.note.isBlank() }
                    f.copy(items = kept + rows)
                }
            }
        }
    }

    /** Speichert; [onSaved] bekommt die ID des Rezepts. */
    fun save(onSaved: (Long, String?) -> Unit) {
        val (input, errors) = _form.value.toInput()
        _errors.value = errors
        if (input == null) {
            _message.value = "Bitte die markierten Felder prüfen."
            return
        }
        _saving.value = true
        _message.value = null
        viewModelScope.launch {
            val result = repo.save(existing?.id, input, _image.value)
            _saving.value = false
            when (result) {
                is Outcome.Error -> _message.value = result.message
                is Outcome.Ok -> onSaved(result.value.first.id, result.value.second)
            }
        }
    }

    fun delete(onDeleted: () -> Unit) {
        val id = existing?.id ?: return
        _saving.value = true
        viewModelScope.launch {
            val result = repo.delete(id)
            _saving.value = false
            when (result) {
                is Outcome.Error -> _message.value = result.message
                is Outcome.Ok -> onDeleted()
            }
        }
    }
}
