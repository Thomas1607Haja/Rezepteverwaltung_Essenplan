import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
    // Bildschirmfotos der Oberfläche in JVM-Tests (Robolectric):
    // ./gradlew recordRoborazziGithubDebug → app/build/outputs/roborazzi/
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

// Adresse der Seite „Unterstützen“ (Buy Me a Coffee). Sie steht nur in
// support-url.txt im Hauptordner des Repositorys; die Weboberfläche liest
// dieselbe Datei. fileContents meldet die Datei dem Configuration Cache,
// eine Änderung wirkt also beim nächsten Build.
val supportUrl = providers.fileContents(rootProject.layout.projectDirectory.file("../support-url.txt"))
    .asText.get().trim()

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
            buildConfigField("boolean", "ALLOW_LOCAL_HTTP", "true")
        }
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
            signingConfig = signingConfigs.findByName("release")
            buildConfigField("boolean", "ALLOW_LOCAL_HTTP", "false")
        }
        // Wie release (mit R8), aber mit Debug-Schlüssel und http zum
        // Testserver im Emulator: zum Prüfen des verkleinerten Builds vor
        // jedem Release (./gradlew assembleGithubQa). Wird nie veröffentlicht.
        create("qa") {
            initWith(getByName("release"))
            applicationIdSuffix = ".qa"
            versionNameSuffix = "-qa"
            signingConfig = signingConfigs.getByName("debug")
            matchingFallbacks += "release"
            buildConfigField("boolean", "ALLOW_LOCAL_HTTP", "true")
        }
    }

    // Zwei Vertriebswege. Gebaut und veröffentlicht wird vorerst nur
    // „github“ (GitHub-Releases, Obtainium). „play“ ist für einen späteren
    // Play-Store-Eintrag vorbereitet und enthält keinen Spendenlink, weil
    // der gegen die Play-Richtlinien verstoßen kann. Beide behalten die
    // gleiche applicationId.
    flavorDimensions += "store"
    productFlavors {
        create("github") {
            dimension = "store"
            buildConfigField("String", "SUPPORT_URL", "\"$supportUrl\"")
        }
        create("play") {
            dimension = "store"
            buildConfigField("String", "SUPPORT_URL", "\"\"")
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

    androidResources {
        // Sprachauswahl pro App ab Android 13 (Einstellungen → Apps →
        // Rezeptplaner → Sprache): Die Liste der Sprachen entsteht aus den
        // Ordnern values-*; die Grundsprache steht in res/resources.properties.
        generateLocaleConfig = true
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
    implementation(libs.androidx.work.runtime)
    implementation(libs.coil.compose)
    // QR-Scanner von Google Play-Diensten: braucht keine Kamera-Berechtigung,
    // das Kamerabild sieht nur der Scanner, die App bekommt nur den Text.
    implementation(libs.play.services.code.scanner)
    // Texterkennung für den Foto-Import (7.7), ebenfalls über die Play-Dienste:
    // Das Modell lädt das Handy selbst, die APK bleibt klein. Erkannt wird
    // auf dem Gerät, die Fotos verlassen das Handy dafür nicht.
    implementation(libs.play.services.text.recognition)
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
