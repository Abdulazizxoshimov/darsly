package uz.darsly.mentor

import android.net.Uri
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.enableEdgeToEdge
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import dagger.hilt.android.AndroidEntryPoint
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.VideoCall
import androidx.compose.material.icons.filled.VideoLibrary
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.NavHostController
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import uz.darsly.mentor.data.livekit.LessonSessionHolder
import uz.darsly.mentor.service.LessonService
import uz.darsly.mentor.ui.archive.ArchiveDetailScreen
import uz.darsly.mentor.ui.archive.ArchiveListScreen
import uz.darsly.mentor.ui.auth.PasswordResetScreen
import uz.darsly.mentor.ui.blocklist.BlocklistScreen
import uz.darsly.mentor.data.api.Lesson
import uz.darsly.mentor.ui.lessons.LessonActions
import uz.darsly.mentor.ui.lessons.LessonsScreen
import uz.darsly.mentor.ui.login.LoginScreen
import uz.darsly.mentor.ui.notifications.NotificationsScreen
import uz.darsly.mentor.ui.profile.ProfileScreen
import uz.darsly.mentor.ui.room.RoomScreen
import uz.darsly.mentor.ui.schedule.ScheduleScreen
import androidx.hilt.navigation.compose.hiltViewModel
import uz.darsly.mentor.ui.AppViewModel
import uz.darsly.mentor.ui.theme.DarslyTheme
import uz.darsly.mentor.ui.update.UpdateGate

@AndroidEntryPoint
class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        // Edge-to-edge — kontent status/navigation bar ORTIGA cho'ziladi (Android 15+
        // da majburiy bo'lib bormoqda). Scaffold'lar o'z insetlarini o'zi qo'llaydi,
        // shuning uchun kontent tizim panellari ostiga tushmaydi.
        enableEdgeToEdge()
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
    const val FORGOT_PASSWORD = "forgot-password"

    // Pastki paneldagi to'rt bo'lim.
    const val LESSONS = "lessons"
    const val SCHEDULE = "schedule"
    const val ARCHIVE = "archive"
    const val PROFILE = "profile"

    // Bildirishnomalar pastki panelda EMAS — asosiy ekranlarning o'ng-tepasidagi
    // qo'ng'iroqdan ochiladi (to'liq ekran, "orqaga" bilan qaytadi).
    const val NOTIFICATIONS = "notifications"

    const val ROOM = "room/{lessonId}"
    fun room(id: String) = "room/$id"

    /** Kabinet ichidan ochiladigan qora ro'yxat (argumentsiz — ustozning o'ziniki). */
    const val BLOCKLIST = "blocklist"

    /**
     * Arxiv detali — video (ilova ichida) + chat. Sarlavhada dars NOMI, shuning
     * uchun argument. Nom ichida `/` yoki `?` bo'lishi mumkin → `Uri.encode`
     * majburiy, aks holda marshrut bo'linardi.
     */
    const val ARCHIVE_DETAIL = "archive/{lessonId}?title={title}"
    fun archiveDetail(id: String, title: String) = "archive/$id?title=" + Uri.encode(title)
}

/** Pastki paneldagi bitta bo'lim. */
private data class Tab(
    val route: String,
    val label: String,
    val icon: ImageVector,
)

/**
 * Pastki panel — Zoom naqshi: eng ko'p ishlatiladigan bo'limlar bir bosishda.
 *
 * **Darslar** (kunlik ish), **Jadval** (rejalashtirish), **Arxiv** (o'tgan
 * darslar — video + chat), **Kabinet** (sozlamalar va chiqish). Bildirishnomalar
 * bu yerda YO'Q — ular o'ng-tepadagi qo'ng'iroqda.
 */
private val TABS = listOf(
    Tab(Routes.LESSONS, "Darslar", Icons.Default.VideoCall),
    Tab(Routes.SCHEDULE, "Jadval", Icons.Default.CalendarMonth),
    Tab(Routes.ARCHIVE, "Arxiv", Icons.Default.VideoLibrary),
    Tab(Routes.PROFILE, "Kabinet", Icons.Default.Person),
)

@Composable
private fun AppNav(vm: AppViewModel = hiltViewModel()) {
    val nav = rememberNavController()
    val ctx = LocalContext.current

    // M1: ilova qayta ochilganda tokenlar shifrlangan preferens'dan tiklanadi —
    // ustoz login ekranini umuman ko'rmaydi.
    val start = remember { if (vm.isLoggedInNow) Routes.LESSONS else Routes.LOGIN }

    // M2/M3: sessiya HOLATI kuzatiladi (hodisa emas). `StateFlow` bo'lgani uchun
    // bu kollektor kechikib ulansa ham allaqachon sodir bo'lgan logoutni ko'radi —
    // "tokenlar o'chdi, lekin ekran Darslarda qoldi" holati bo'lishi mumkin emas.
    val loggedIn by vm.loggedIn.collectAsStateWithLifecycle()
    LaunchedEffect(loggedIn) {
        val route = nav.currentDestination?.route
        // Parolni tiklash ekrani chiqmagan holatda ochiladi va uning oxirida
        // `AuthRepository.resetPassword` sessiyani tozalaydi. Shartsiz navigatsiya
        // bu oqimni o'z o'rtasida uzib qo'yardi.
        if (!loggedIn && route != Routes.LOGIN && route != Routes.FORGOT_PASSWORD) {
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

    val backStack by nav.currentBackStackEntryAsState()
    val currentRoute = backStack?.destination?.route
    val showBottomBar = TABS.any { it.route == currentRoute }
    val unread by vm.unreadCount.collectAsStateWithLifecycle()

    // Bildirishnomalar ekraniga o'tish — qo'ng'iroqdan (asosiy ekranlarda).
    val openNotifications: () -> Unit = { nav.navigate(Routes.NOTIFICATIONS) }

    Scaffold(
        bottomBar = {
            // Panel FAQAT asosiy bo'limlarda. Xona, arxiv detali va login
            // ekranlarida u ekrandan joy o'g'irlardi va dars paytida chalg'itardi.
            if (showBottomBar) {
                BottomBar(nav = nav, currentRoute = currentRoute)
            }
        },
    ) { padding ->
        NavHost(
            navController = nav,
            startDestination = start,
            modifier = Modifier.fillMaxSize().padding(padding),
            // M9 — ekranlararo yumshoq o'tish: fade + juda kichik siljish (~5%).
            // Keskin "sakrash" o'rniga. Tab almashuvida ham, oldinga navigatsiyada
            // ham tabiiy; to'liq slayd tablar uchun og'ir bo'lardi.
            enterTransition = { fadeIn(tween(220)) + slideInHorizontally(tween(220)) { it / 18 } },
            exitTransition = { fadeOut(tween(160)) },
            popEnterTransition = { fadeIn(tween(220)) },
            popExitTransition = { fadeOut(tween(160)) + slideOutHorizontally(tween(160)) { it / 18 } },
        ) {
            composable(Routes.LOGIN) {
                LoginScreen(
                    onLoggedIn = {
                        nav.navigate(Routes.LESSONS) {
                            popUpTo(Routes.LOGIN) { inclusive = true }
                        }
                    },
                    onForgotPassword = { nav.navigate(Routes.FORGOT_PASSWORD) },
                )
            }

            composable(Routes.FORGOT_PASSWORD) {
                PasswordResetScreen(
                    onBack = { nav.popBackStack() },
                    // Parol o'zgardi → login ekrani. Avtomatik kirish ATAYLAB yo'q:
                    // server barcha sessiyalarni bekor qilgan, va yangi parolni
                    // bir marta kiritish uni eslab qolishga yordam beradi.
                    onDone = {
                        nav.navigate(Routes.LOGIN) {
                            popUpTo(Routes.LOGIN) { inclusive = true }
                        }
                    },
                )
            }

            // YAKUNLANGAN DARS XONANI OCHMAYDI.
            //
            // Avval har qanday dars — jadvaldagi o'tgan dars ham — bosilganda xona
            // ekrani ochilardi va faqat shundan keyin server `lesson is not active`
            // (400) qaytarardi: ustoz uchun bu "dars qaytadan boshlandi" degan
            // yolg'on taassurot edi. Endi yo'nalish darsning holatiga qarab
            // tanlanadi (qoida `LessonActions` da, sof va test ostida).
            val openLesson: (Lesson) -> Unit = { l ->
                when (LessonActions.primary(l.status)) {
                    LessonActions.Primary.START,
                    LessonActions.Primary.RESUME,
                    -> nav.navigate(Routes.room(l.id))
                    // Yakunlangan dars → arxiv (video + chat).
                    LessonActions.Primary.RECORDINGS -> nav.navigate(Routes.archiveDetail(l.id, l.title))
                    LessonActions.Primary.NONE -> Unit // bekor qilingan dars
                }
            }

            composable(Routes.LESSONS) {
                LessonsScreen(
                    onOpenLesson = openLesson,
                    onOpenRecordings = { nav.navigate(Routes.archiveDetail(it.id, it.title)) },
                    unread = unread,
                    onOpenNotifications = openNotifications,
                )
            }

            composable(Routes.SCHEDULE) {
                ScheduleScreen(
                    onOpenLesson = openLesson,
                    unread = unread,
                    onOpenNotifications = openNotifications,
                )
            }

            composable(Routes.ARCHIVE) {
                ArchiveListScreen(
                    onOpen = { nav.navigate(Routes.archiveDetail(it.id, it.title)) },
                    unread = unread,
                    onOpenNotifications = openNotifications,
                )
            }

            composable(Routes.NOTIFICATIONS) {
                NotificationsScreen(
                    // Darsga bog'liq bildirishnoma bosilganda o'sha dars ochiladi.
                    onOpenLesson = { lessonId -> nav.navigate(Routes.room(lessonId)) },
                )
            }

            composable(Routes.PROFILE) {
                ProfileScreen(onOpenBlocklist = { nav.navigate(Routes.BLOCKLIST) })
            }

            // Qora ro'yxat pastki panelda YO'Q: u kamdan-kam ochiladi va Kabinet
            // ichidagi o'z joyida turadi (`showBottomBar` uni ko'rsatmaydi — bu
            // ekran to'liq ekranli, "orqaga" bilan Kabinetga qaytadi).
            composable(Routes.BLOCKLIST) {
                BlocklistScreen(onBack = { nav.popBackStack() })
            }

            composable(Routes.ROOM) { entry ->
                val lessonId = entry.arguments?.getString("lessonId") ?: return@composable
                RoomScreen(lessonId = lessonId, onLeave = { nav.popBackStack() })
            }

            composable(Routes.ARCHIVE_DETAIL) { entry ->
                val lessonId = entry.arguments?.getString("lessonId") ?: return@composable
                ArchiveDetailScreen(
                    lessonId = lessonId,
                    lessonTitle = entry.arguments?.getString("title").orEmpty(),
                    onBack = { nav.popBackStack() },
                )
            }
        }
    }
}

@Composable
private fun BottomBar(nav: NavHostController, currentRoute: String?) {
    NavigationBar {
        TABS.forEach { tab ->
            val selected = currentRoute == tab.route
            NavigationBarItem(
                selected = selected,
                onClick = {
                    if (selected) return@NavigationBarItem
                    nav.navigate(tab.route) {
                        // Bo'limlar orasida almashish stekni O'STIRMAYDI: aks holda
                        // "Orqaga" tugmasi ustozni o'nlab bo'lim orqali qaytarardi.
                        // Bo'lim holati saqlanadi — ro'yxat pozitsiyasi yo'qolmasin.
                        popUpTo(nav.graph.findStartDestination().id) { saveState = true }
                        launchSingleTop = true
                        restoreState = true
                    }
                },
                icon = { Icon(tab.icon, contentDescription = tab.label) },
                label = { Text(tab.label) },
            )
        }
    }
}
