package io.github.thomas1607haja.rezeptplaner.ui.common

import android.content.Context
import androidx.annotation.PluralsRes
import androidx.annotation.StringRes
import androidx.compose.runtime.Composable
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.ui.platform.LocalResources
import io.github.thomas1607haja.rezeptplaner.api.ErrorTexts
import java.util.Locale

/**
 * Texte der App (Mehrsprachigkeit, docs/PLANUNG.md).
 *
 * Die Texte stehen in res/values…/strings….xml; values/ ist Englisch und
 * Rückfallsprache. Android wählt die Sprache des Systems bzw. die unter
 * „Sprache der App“ eingestellte (ab Android 13, res/resources.properties).
 *
 * Platzhalter heißen wie im Web {name}: tr(R.string.x, "title" to "Lasagne").
 * In Composables tr()/trPlural() benutzen; außerhalb (ViewModels,
 * Repositorys) Texts.get()/Texts.plural() – sie nehmen die Ressourcen der
 * Anwendung und damit die aktuelle Sprache.
 */
object Texts {
    private var app: Context? = null

    /** Einmal in RezeptplanerApplication.onCreate. */
    fun init(context: Context) {
        app = context.applicationContext
    }

    private val context: Context
        get() = checkNotNull(app) { "Texts.init(context) fehlt" }

    /** Text mit eingesetzten Platzhaltern. */
    fun get(@StringRes id: Int, vararg params: Pair<String, Any?>): String =
        fill(context.getString(id), params)

    /** Pluralform zu count; {count} wird eingesetzt. */
    fun plural(@PluralsRes id: Int, count: Int, vararg params: Pair<String, Any?>): String =
        fill(context.resources.getQuantityString(id, count), arrayOf("count" to count, *params))

    /**
     * Text zu einem Fehlercode des Servers (server/internal/apierr) mit
     * seinen Parametern; unbekannter Code → [fallback] (englischer Text des
     * Servers).
     */
    fun error(code: String?, params: Map<String, Any?> = emptyMap(), fallback: String = code.orEmpty()): String {
        val id = code?.let { ErrorTexts.ids[it] } ?: return fallback
        return fill(context.getString(id), params.toList().toTypedArray())
    }

    /**
     * Wie [error], aber null, wenn der Code unbekannt ist oder Texts noch
     * nicht bereit ist (reine JVM-Tests). Kennt auch die Codes der App
     * ("client_…").
     */
    fun errorOrNull(code: String, params: Map<String, Any?>): String? {
        val ctx = app ?: return null
        val id = ErrorTexts.ids[code] ?: io.github.thomas1607haja.rezeptplaner.api.clientErrorTexts[code] ?: return null
        return fill(ctx.getString(id), params.toList().toTypedArray())
    }

    /** Sprache der Oberfläche als Locale (für Zahlen und Daten). */
    fun locale(): Locale = app?.resources?.configuration?.locales?.get(0) ?: Locale.getDefault()

    /** Setzt {name} ein; ganze Zahlen ohne Tausenderpunkt, andere nach Sprache. */
    fun fill(text: String, params: Array<out Pair<String, Any?>>): String {
        var result = text
        for ((name, value) in params) {
            val shown = when (value) {
                is Double -> io.github.thomas1607haja.rezeptplaner.ui.formatNumber(value)
                else -> value?.toString().orEmpty()
            }
            result = result.replace("{$name}", shown)
        }
        return result
    }
}

/** Text in der Sprache der Oberfläche, mit Platzhaltern {name}; LocalResources folgt einem Sprachwechsel. */
@Composable
@ReadOnlyComposable
fun tr(@StringRes id: Int, vararg params: Pair<String, Any?>): String {
    return Texts.fill(LocalResources.current.getString(id), params)
}

/** Pluralform in der Sprache der Oberfläche; {count} wird eingesetzt. */
@Composable
@ReadOnlyComposable
fun trPlural(@PluralsRes id: Int, count: Int, vararg params: Pair<String, Any?>): String {
    return Texts.fill(LocalResources.current.getQuantityString(id, count), arrayOf("count" to count, *params))
}
