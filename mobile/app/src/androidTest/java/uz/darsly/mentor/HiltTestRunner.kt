package uz.darsly.mentor

import android.app.Application
import android.content.Context
import androidx.test.runner.AndroidJUnitRunner
import dagger.hilt.android.testing.HiltTestApplication

// Instrumented testlar DarslyApp o'rniga HiltTestApplication bilan ishga tushadi
// (standart Hilt test naqshi). build.gradle.kts `testInstrumentationRunner`da ko'rsatilgan.
class HiltTestRunner : AndroidJUnitRunner() {
    override fun newApplication(cl: ClassLoader?, name: String?, context: Context?): Application =
        super.newApplication(cl, HiltTestApplication::class.java.name, context)
}
