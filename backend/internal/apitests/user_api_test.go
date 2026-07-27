package apitests_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// ─── USER: success ─────────────────────────────────────────────────────────────

func TestUserGetMe(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	access, _ := mustRegister(t, cl, "Sardor", "getme@darsly.uz", "parol12345")
	code, body := cl.get("/api/v1/users/me", access)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "getme@darsly.uz", gjson(body, "data", "email"))
	require.Equal(t, "student", gjson(body, "data", "role"))
}

func TestUserUpdateProfile(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	access, _ := mustRegister(t, cl, "Eski Ism", "upd@darsly.uz", "parol12345")
	code, body := cl.do(http.MethodPut, "/api/v1/users/me", access, map[string]any{
		"full_name": "Yangi Ism", "language": "en",
	})
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "Yangi Ism", gjson(body, "data", "full_name"))
	require.Equal(t, "en", gjson(body, "data", "language"))
}

func TestUserChangePassword(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	access, _ := mustRegister(t, cl, "Parolchi", "chpw@darsly.uz", "eskiparol12345")

	// Joriy parol bilan → 204.
	code, _ := cl.do(http.MethodPut, "/api/v1/users/me/password", access, map[string]string{
		"current_password": "eskiparol12345", "new_password": "yangiparol12345",
	})
	require.Equal(t, http.StatusNoContent, code)

	// Yangi parol bilan login ishlaydi.
	code, _ = cl.post("/api/v1/auth/login", "", map[string]string{"email": "chpw@darsly.uz", "password": "yangiparol12345"})
	require.Equal(t, http.StatusOK, code, "yangi parol bilan login ishlashi kerak")

	// Eski parol endi ishlamaydi.
	code, _ = cl.post("/api/v1/auth/login", "", map[string]string{"email": "chpw@darsly.uz", "password": "eskiparol12345"})
	require.Equal(t, http.StatusUnauthorized, code, "eski parol bekor bo'lishi kerak")
}

// ─── USER: bad ─────────────────────────────────────────────────────────────────

// TestUserGetOtherForbidden — IDOR: student boshqa foydalanuvchi profilini o'qiy olmaydi.
func TestUserGetOtherForbidden(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	// B foydalanuvchi — uning id'sini olamiz.
	accessB, _ := mustRegister(t, cl, "Bekzod", "victim@darsly.uz", "parol12345")
	_, bodyB := cl.get("/api/v1/users/me", accessB)
	idB := gjson(bodyB, "data", "id")
	require.NotEmpty(t, idB)

	// A foydalanuvchi (student) B'ning profilini so'raydi → 403.
	accessA, _ := mustRegister(t, cl, "Akmal", "attacker@darsly.uz", "parol12345")
	code, _ := cl.get("/api/v1/users/"+idB, accessA)
	require.Equal(t, http.StatusForbidden, code, "student boshqa foydalanuvchi profilini ko'ra olmaydi (IDOR)")
}

// TestUserSelfRoleEscalationBlocked — student PUT /users/me orqali o'zini mentor qila olmaydi.
func TestUserSelfRoleEscalationBlocked(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	access, _ := mustRegister(t, cl, "Oddiy", "escalate@darsly.uz", "parol12345")

	// Role o'zgartirishga urinish — so'rov o'tadi, lekin rol e'tiborsiz qoldiriladi.
	code, body := cl.do(http.MethodPut, "/api/v1/users/me", access, map[string]any{
		"full_name": "Oddiy", "role": "mentor",
	})
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "student", gjson(body, "data", "role"), "student o'zini mentor qila olmasligi kerak")

	// Qayta o'qiganda ham student.
	code, body = cl.get("/api/v1/users/me", access)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "student", gjson(body, "data", "role"))
}

func TestUserChangePasswordWrongCurrent(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	access, _ := mustRegister(t, cl, "Xato", "wrongcur@darsly.uz", "parol12345")
	code, _ := cl.do(http.MethodPut, "/api/v1/users/me/password", access, map[string]string{
		"current_password": "notthecurrent", "new_password": "yangiparol12345",
	})
	require.Equal(t, http.StatusBadRequest, code, "noto'g'ri joriy parol → 400")
}

// ─── H-2: user-management endi faqat admin'da (mentor global admin EMAS) ─────────

// TestMentorCannotManageUsers — mentor endi butun user bazasini boshqara OLMAYDI:
// ro'yxat/o'chirish/deaktivatsiya/parol-reset hammasi 403.
func TestMentorCannotManageUsers(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	// Qurbon (student) — id'sini olamiz.
	registerStudent(t, cl, "Qurbon", "victim-h2@darsly.uz")
	victimID := userID(t, pg, "victim-h2@darsly.uz")

	// Mentor — user-management route'lariga urinadi.
	mentorTok := registerMentor(t, cl, pg, "Ustoz", "mentor-h2@darsly.uz")

	code, _ := cl.get("/api/v1/users", mentorTok)
	require.Equal(t, http.StatusForbidden, code, "mentor foydalanuvchilar ro'yxatini ololmaydi")

	code, _ = cl.do(http.MethodDelete, "/api/v1/users/"+victimID, mentorTok, nil)
	require.Equal(t, http.StatusForbidden, code, "mentor user o'chira olmaydi")

	code, _ = cl.post("/api/v1/users/"+victimID+"/deactivate", mentorTok, nil)
	require.Equal(t, http.StatusForbidden, code, "mentor user deaktivatsiya qila olmaydi")

	code, _ = cl.do(http.MethodPut, "/api/v1/users/"+victimID+"/password", mentorTok, map[string]string{
		"new_password": "hijacked12345",
	})
	require.Equal(t, http.StatusForbidden, code, "mentor boshqa foydalanuvchi parolini reset qila olmaydi")
}

// TestAdminCanManageUsers — admin roli user-management qila oladi (deactivate/activate/delete).
func TestAdminCanManageUsers(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	registerStudent(t, cl, "Nishon", "target-h2@darsly.uz")
	targetID := userID(t, pg, "target-h2@darsly.uz")

	adminTok := registerAdmin(t, cl, pg, "Admin", "admin-h2@darsly.uz")

	code, _ := cl.get("/api/v1/users", adminTok)
	require.Equal(t, http.StatusOK, code, "admin foydalanuvchilar ro'yxatini oladi")

	code, _ = cl.post("/api/v1/users/"+targetID+"/deactivate", adminTok, nil)
	require.Equal(t, http.StatusNoContent, code, "admin user deaktivatsiya qiladi")

	code, _ = cl.post("/api/v1/users/"+targetID+"/activate", adminTok, nil)
	require.Equal(t, http.StatusNoContent, code, "admin user aktivatsiya qiladi")

	code, _ = cl.do(http.MethodPut, "/api/v1/users/"+targetID+"/password", adminTok, map[string]string{
		"new_password": "resetbyadmin12345",
	})
	require.Equal(t, http.StatusNoContent, code, "admin boshqa foydalanuvchi parolini reset qiladi")

	// Reset'dan so'ng yangi parol bilan login ishlaydi.
	code, _ = cl.post("/api/v1/auth/login", "", map[string]string{"email": "target-h2@darsly.uz", "password": "resetbyadmin12345"})
	require.Equal(t, http.StatusOK, code, "admin reset qilgan parol bilan login ishlashi kerak")
}
