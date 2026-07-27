// Darsly Mentor — Android ilova (R0 spike)
pluginManagement {
    repositories {
        google {
            content {
                includeGroupByRegex("com\\.android.*")
                includeGroupByRegex("com\\.google.*")
                includeGroupByRegex("androidx.*")
            }
        }
        mavenCentral()
        gradlePluginPortal()
    }
}

plugins {
    // Muhitda JDK bo'lmasa (masalan faqat JRE o'rnatilgan bo'lsa) Gradle kerakli
    // JDK 17 toolchain'ini O'ZI yuklab oladi. Shu tufayli qurish mashinaga
    // bog'liq emas: CI va boshqa dasturchi kompyuterida ham hech narsa
    // sozlamasdan ishlaydi (foydalanuvchining global muhitiga tegilmaydi).
    id("org.gradle.toolchains.foojay-resolver-convention") version "0.8.0"
}

dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        mavenCentral()
        // livekit-android 2.27.0 → com.github.davidliu:audioswitch (commit-hash versiya)
        // faqat JitPack'da mavjud. Xavfsizlik uchun FAQAT shu guruhga ruxsat beramiz,
        // qolgan barcha bog'liqlik Maven Central / Google'dan keladi.
        maven("https://jitpack.io") {
            content { includeGroup("com.github.davidliu") }
        }
    }
}

rootProject.name = "DarslyMentor"
include(":app")
