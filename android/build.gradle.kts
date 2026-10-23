// Plugins werden hier nur bekannt gemacht und im Modul app angewendet.
// Kotlin bringt das Android-Gradle-Plugin ab Version 9 selbst mit.
plugins {
    alias(libs.plugins.android.application) apply false
    alias(libs.plugins.kotlin.compose) apply false
    alias(libs.plugins.kotlin.serialization) apply false
    alias(libs.plugins.roborazzi) apply false
}
