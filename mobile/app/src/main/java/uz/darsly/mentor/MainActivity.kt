package uz.darsly.mentor

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import uz.darsly.mentor.data.api.Session
import uz.darsly.mentor.data.livekit.LessonSessionHolder
import uz.darsly.mentor.service.LessonService
import uz.darsly.mentor.ui.lessons.LessonsScreen
import uz.darsly.mentor.ui.login.LoginScreen
import uz.darsly.mentor.ui.room.RoomScreen
import uz.darsly.mentor.ui.theme.DarslyTheme
import uz.darsly.mentor.ui.update.UpdateGate

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            DarslyTheme {
                Box(Modifier.fillMaxSize()) {
                    AppNav()
                    // M42: bloklovchi/yumshoq yangilanish dialogi — hamma narsa ustida.
                    UpdateGate()
                }
            }
        }
    }
}

private object Routes {
    const val LOGIN = "login"
    const val LESSONS = "lessons"
    const val ROOM = "room/{lessonId}"
    fun room(id: String) = "room/$id"
}

@Composable
private fun AppNav() {
    val nav = rememberNavController()
    val ctx = LocalContext.current

    // M1: ilova qayta ochilganda tokenlar shifrlangan preferens'dan tiklanadi —
    // ustoz login ekranini umuman ko'rmaydi.
    val start = remember { if (Session.isLoggedIn) Routes.LESSONS else Routes.LOGIN }

    // M2/M3: sessiya HOLATI kuzatiladi (hodisa emas). `StateFlow` bo'lgani uchun
    // bu kollektor kechikib ulansa ham allaqachon sodir bo'lgan logoutni ko'radi —
    // "tokenlar o'chdi, lekin ekran Darslarda qoldi" holati bo'lishi mumkin emas.
    val loggedIn by Session.loggedIn.collectAsStateWithLifecycle()
    LaunchedEffect(loggedIn) {
        if (!loggedIn && nav.currentDestination?.route != Routes.LOGIN) {
            // Dars o'rtasida bo'lsak — LiveKit sessiyasi va foreground servis
            // osilib qolmasin (aks holda ustoz "ko'rinmas xona"da qoladi).
            LessonSessionHolder.stop()
            LessonService.stop(ctx)
            nav.navigate(Routes.LOGIN) {
                popUpTo(nav.graph.id) { inclusive = true }
                launchSingleTop = true
            }
        }
    }

    Scaffold { padding ->
        NavHost(
            navController = nav,
            startDestination = start,
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
            composable(Routes.LOGIN) {
                LoginScreen(
                    onLoggedIn = {
                        nav.navigate(Routes.LESSONS) {
                            popUpTo(Routes.LOGIN) { inclusive = true }
                        }
                    },
                )
            }
            composable(Routes.LESSONS) {
                LessonsScreen(onOpenLesson = { nav.navigate(Routes.room(it.id)) })
            }
            composable(Routes.ROOM) { entry ->
                val lessonId = entry.arguments?.getString("lessonId") ?: return@composable
                RoomScreen(lessonId = lessonId, onLeave = { nav.popBackStack() })
            }
        }
    }
}
