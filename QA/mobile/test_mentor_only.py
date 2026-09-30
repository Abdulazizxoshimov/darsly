"""Mobil mentor-only guard — backend `/auth/me` role'ni to'g'ri qaytaradimi.

Mobil ilova (AuthRepository.kt) login'dan keyin `GET /auth/me` chaqirib
`role == "admin"` bo'lsa RAD etadi (AdminNotAllowedException). Bu guard backend
`role` claim'iga tayanadi — shu yerda backend har rol uchun to'g'ri `role`
qaytarishini tasdiqlaymiz (guard ishlashi uchun zarur shart).

⚠️ Mobil FAQAT admin'ni rad etadi — student'ni EMAS (client-guard yo'q). Bu QA
bo'shlig'i GAPS.md da hujjatlangan; backend tomondan student mentor-endpointlarida
baribir 403 oladi (backend/test_rbac.py).
"""
import pytest

pytestmark = pytest.mark.destructive


def test_admin_me_role_is_admin(admin):
    """Admin login → /auth/me role=admin (mobil shuni ko'rib login'ni rad etadi)."""
    data = admin.data(admin.get("/api/v1/auth/me"))
    assert data["role"] == "admin"


def test_mentor_me_role_is_mentor(as_mentor):
    """Mentor login → role=mentor (mobil qabul qiladi)."""
    data = as_mentor.data(as_mentor.get("/api/v1/auth/me"))
    assert data["role"] == "mentor"


def test_student_me_role_is_student(as_user):
    """Student login → role=student. Mobil buni client-side RAD ETMAYDI (GAP),
    lekin backend RBAC mentor-endpointlarda 403 beradi (test_rbac.py)."""
    data = as_user.data(as_user.get("/api/v1/auth/me"))
    assert data["role"] == "student"
