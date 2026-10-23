# Eigene R8-Regeln für den Release-Build. Die meisten Bibliotheken
# (kotlinx.serialization, Navigation, Compose) bringen ihre Regeln selbst mit.

# QR-Scanner der Play-Dienste (play-services-code-scanner 16.1.0): Die
# Bibliothek liefert keine eigenen R8-Regeln mit. ML Kit sucht seine
# Komponenten zur Laufzeit; ohne diese Regeln brach GmsBarcodeScanning.getClient()
# im Release-Build mit einer NullPointerException in
# com.google.android.gms.internal.mlkit_code_scanner.zzny.<init> ab (App 0.2.0/0.2.1,
# Debug-Build ohne R8 lief). Beide Pakete sind klein.
-keep class com.google.mlkit.** { *; }
-keep class com.google.android.gms.internal.mlkit_code_scanner.** { *; }

# Texterkennung (play-services-mlkit-text-recognition, Foto-Import 7.7):
# vorsorglich dieselbe Regel für ihre internen Klassen.
-keep class com.google.android.gms.internal.mlkit_vision_text_common.** { *; }
-keep class com.google.android.gms.internal.mlkit_vision_common.** { *; }
