package io.github.thomas1607haja.rezeptplaner.ui.common

import android.content.Context
import android.util.Log
import com.google.android.gms.common.moduleinstall.ModuleInstall
import com.google.android.gms.common.moduleinstall.ModuleInstallRequest
import com.google.mlkit.common.MlKitException
import com.google.mlkit.vision.barcode.common.Barcode
import com.google.mlkit.vision.codescanner.GmsBarcodeScannerOptions
import com.google.mlkit.vision.codescanner.GmsBarcodeScanning
import io.github.thomas1607haja.rezeptplaner.R

private const val TAG = "Rezeptplaner"

/** Ergebnis eines QR-Scans. */
sealed interface ScanResult {
    data class Text(val value: String) : ScanResult
    data object Cancelled : ScanResult
    /** Verständliche Meldung, z. B. „Scanner wird nachgeladen“. */
    data class Failed(val message: String) : ScanResult
}

/**
 * Öffnet den Google-Code-Scanner (Play-Dienste, ohne Kamera-Berechtigung –
 * derselbe wie bei der Einrichtung) für einen QR-Code. Fehlt das
 * Scanner-Modul noch auf dem Gerät, stößt die Funktion das Nachladen an und
 * meldet das, statt still nichts zu tun.
 */
fun scanQrCode(context: Context, onResult: (ScanResult) -> Unit) {
    val unavailable = Texts.get(R.string.settings_qr_unavailable)
    try {
        val options = GmsBarcodeScannerOptions.Builder().setBarcodeFormats(Barcode.FORMAT_QR_CODE).build()
        val scanner = GmsBarcodeScanning.getClient(context, options)
        scanner.startScan()
            .addOnSuccessListener { barcode ->
                onResult(barcode.rawValue?.let { ScanResult.Text(it) } ?: ScanResult.Failed(Texts.get(R.string.settings_qr_no_text)))
            }
            .addOnCanceledListener { onResult(ScanResult.Cancelled) }
            .addOnFailureListener { e ->
                Log.w(TAG, "QR-Scanner meldet Fehler", e)
                if (e is MlKitException && e.errorCode == MlKitException.UNAVAILABLE) {
                    // Das Scanner-Modul der Play-Dienste fehlt noch: nachladen lassen.
                    runCatching {
                        ModuleInstall.getClient(context)
                            .installModules(ModuleInstallRequest.newBuilder().addApi(scanner).build())
                    }
                    onResult(
                        ScanResult.Failed(Texts.get(R.string.settings_qr_installing)),
                    )
                } else {
                    onResult(ScanResult.Failed("$unavailable (${e.message ?: e.javaClass.simpleName})"))
                }
            }
    } catch (e: RuntimeException) {
        // Fehler der Play-Dienste beim Start sollen die App nicht beenden.
        Log.w(TAG, "QR-Scanner startet nicht", e)
        onResult(ScanResult.Failed("$unavailable (${e.message ?: e.javaClass.simpleName})"))
    }
}
