package uz.darsly.mentor.ui.login

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.darsly.mentor.data.api.AdminNotAllowedException

/**
 * Login xatosining foydalanuvchi matni (MEDIUM: LoginViewModel seam).
 *
 * Mobil ilova FAQAT mentorlar uchun — admin kira olmaydi. Bu holat aniq,
 * o'zbekcha sabab ko'rsatishi kerak, generic "xato" emas.
 */
class LoginErrorTest {

    @Test fun adminBlockShowsSpecificReason() {
        // BUG: admin holatida generic "xato" ko'rsatilsa, admin parolni yoki
        // internetni ayblab qayta-qayta urinardi — haqiqiy sabab boshqa.
        val msg = LoginError.messageFor(AdminNotAllowedException())
        assertTrue("web (sayt) haqida eslatilsin", msg.contains("web"))
    }

    @Test fun genericErrorFallsBackToHumanMessage() {
        // Boshqa xato — bo'sh bo'lmagan o'qiladigan matn qaytishi kerak
        // (foydalanuvchiga xom istisno matni ko'rsatilmaydi).
        val msg = LoginError.messageFor(RuntimeException("boom"))
        assertTrue(msg.isNotBlank())
        assertEquals(msg, LoginError.messageFor(RuntimeException("boom")))
    }
}
