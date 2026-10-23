// Android-App des Rezeptplaners. Siehe docs/PLANUNG.md, Abschnitte 7 und 10.
pluginManagement {
    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    // Bibliotheken nur aus diesen Quellen, nicht aus einzelnen Modulen.
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        mavenCentral()
    }
}

rootProject.name = "Rezeptplaner"
include(":app")
