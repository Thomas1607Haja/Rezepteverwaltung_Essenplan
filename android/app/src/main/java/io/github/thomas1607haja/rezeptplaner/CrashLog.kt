package io.github.thomas1607haja.rezeptplaner

import android.content.Context
import android.os.Build
import java.io.File
import java.io.PrintWriter
import java.io.StringWriter
import java.time.ZonedDateTime

/**
 * Hält den letzten Absturz fest, damit er sich ohne USB-Debugging aus der
 * App heraus teilen lässt (Einrichtung und Einstellungen zeigen ihn an).
 * Es wird nur der Fehler mit Geräteangaben gespeichert – keine Daten der
 * App, kein API-Schlüssel – und nichts automatisch verschickt.
 */
object CrashLog {
    private const val FILE = "last_crash.txt"

    /** Ab jetzt jeden unbehandelten Fehler festhalten, dann wie gewohnt beenden. */
    fun install(context: Context) {
        val file = File(context.filesDir, FILE)
        val previous = Thread.getDefaultUncaughtExceptionHandler()
        Thread.setDefaultUncaughtExceptionHandler { thread, error ->
            runCatching { file.writeText(report(thread, error)) }
            previous?.uncaughtException(thread, error)
        }
    }

    /** Bericht des letzten Absturzes oder null. */
    fun read(context: Context): String? =
        File(context.filesDir, FILE).takeIf { it.exists() }?.readText()

    fun clear(context: Context) {
        File(context.filesDir, FILE).delete()
    }

    private fun report(thread: Thread, error: Throwable): String {
        val trace = StringWriter().also { error.printStackTrace(PrintWriter(it)) }
        return buildString {
            appendLine("Rezeptplaner ${BuildConfig.VERSION_NAME} (${BuildConfig.VERSION_CODE})")
            appendLine("Zeit: ${ZonedDateTime.now()}")
            appendLine("Gerät: ${Build.MANUFACTURER} ${Build.MODEL}, Android ${Build.VERSION.RELEASE} (API ${Build.VERSION.SDK_INT})")
            appendLine("Thread: ${thread.name}")
            appendLine()
            append(trace)
        }
    }
}
