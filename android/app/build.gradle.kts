import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
    // Bildschirmfotos der Oberfläche in JVM-Tests (Robolectric):
    // ./gradlew recordRoborazziDebug → app/build/outputs/roborazzi/
    alias(libs.plugins.roborazzi)
}

val appVersionName = providers.gradleProperty("appVersionName").get()

// versionCode muss bei jedem Update steigen: 1.2.3 -> 10203.
val appVersionCode = appVersionName.split(".").map { it.toInt() }.let { (major, minor, patch) ->
    major * 10000 + minor * 100 + patch
}

// Signaturschlüssel für Release-Builds. In der GitHub Action kommen die
// Angaben aus Umgebungsvariablen (geheime Werte des Repositorys), lokal
// optional aus android/keystore.properties (nie einchecken, steht in
// .gitignore). Ohne Angaben entsteht nur ein unsigniertes Release-APK.
val signing: Map<String, String> = run {
    val file = rootProject.file("keystore.properties")
    val props = Properties()
    if (file.exists()) file.inputStream().use(props::load)
    fun value(env: String, prop: String): String? =
        System.getenv(env)?.takeIf { it.isNotBlank() } ?: props.getProperty(prop)
    listOfNotNull(
        value("ANDROID_KEYSTORE_PATH", "storeFile")?.let { "storeFile" to it },
        value("ANDROID_KEYSTORE_PASSWORD", "storePassword")?.let { "storePassword" to it },
        value("ANDROID_KEY_ALIAS", "keyAlias")?.let { "keyAlias" to it },
        value("ANDROID_KEY_PASSWORD", "keyPassword")?.let { "keyPassword" to it },
    ).toMap()
}

android {
    namespace = "io.github.thomas1607haja.rezeptplaner"
    // Übersetzt wird gegen das SDK von Android 17, weil die aktuellen
    // AndroidX-Bibliotheken das verlangen. Das Verhalten der App bestimmt
    // targetSdk (Android 16); compileSdk macht nur neuere APIs verfügbar.
    compileSdk = 37

    defaultConfig {
        applicationId = "io.github.thomas1607haja.rezeptplaner"
        minSdk = 29 // Android 10, siehe docs/PLANUNG.md Abschnitt 7
        targetSdk = 36 // Android 16
        versionCode = appVersionCode
        versionName = appVersionName
    }

    signingConfigs {
        if (signing.size == 4) {
            create("release") {
                storeFile = file(signing.getValue("storeFile"))
                storePassword = signing.getValue("storePassword")
                keyAlias = signing.getValue("keyAlias")
                keyPassword = signing.getValue("keyPassword")
            }
        }
    }

    buildTypes {
        debug {
            // Debug-Version neben der installierten Release-Version nutzbar.
            applicationIdSuffix = ".debug"
            versionNameSuffix = "-debug"
        }
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
            signingConfig = signingConfigs.findByName("release")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    testOptions {
        // Robolectric braucht die Ressourcen der App (Texte, Symbole, Farben).
        unitTests.isIncludeAndroidResources = true
        // Robolectric greift für Android 16 auf JDK-Interna zu.
        unitTests.all { it.jvmArgs("--add-opens=java.base/jdk.internal.access=ALL-UNNAMED") }
    }

    lint {
        // targetSdk 36 ist bewusst gewählt (Zielgeräte mit Android 16).
        disable += "OldTargetApi"
    }

    buildFeatures {
        compose = true
        buildConfig = true // BuildConfig.VERSION_NAME für die Anzeige der Version
    }
}

dependencies {
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.lifecycle.runtime.compose)
    implementation(libs.androidx.navigation.compose)
    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.ui.tooling.preview)
    implementation(libs.compose.material3)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    implementation(libs.androidx.datastore.preferences)
    implementation(libs.kotlinx.serialization.json)
    implementation(libs.okhttp)
    // QR-Scanner von Google Play-Diensten: braucht keine Kamera-Berechtigung,
    // das Kamerabild sieht nur der Scanner, die App bekommt nur den Text.
    implementation(libs.play.services.code.scanner)
    // Neuere Basis als die des Scanners, siehe libs.versions.toml.
    implementation(libs.play.services.base)
    implementation(libs.play.services.basement)
    implementation(libs.play.services.tasks)
    debugImplementation(libs.compose.ui.tooling)
    debugImplementation(libs.compose.ui.test.manifest)

    testImplementation(libs.junit)
    testImplementation(libs.okhttp.mockwebserver)
    testImplementation(libs.kotlinx.coroutines.test)
    testImplementation(libs.robolectric)
    testImplementation(libs.roborazzi)
    testImplementation(libs.roborazzi.compose)
    testImplementation(platform(libs.compose.bom))
    testImplementation(libs.compose.ui.test.junit4)
    testImplementation(libs.androidx.test.junit)
}
