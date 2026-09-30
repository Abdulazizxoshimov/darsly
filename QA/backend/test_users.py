"""Users — me / parol / privilege-escalation / admin CRUD (RBAC) / IDOR.

Kontrakt (api-contract «Users» + CLAUDE.md senior-fix):
  GET /users/me           → User
  PUT /users/me           → User  (Role BLOKLANADI — student o'zini mentor qila olmaydi)
  PUT /users/me/password  {current_password,new_password} → 204
  GET/POST/PUT/DELETE /users[...] → FAQAT admin (student/mentor → 403)
  GET /users/:id (begona) → 403 (IDOR)
"""
import pytest

from lib.auth import login
from lib.client import Client
from lib.schemas import USER, validate

pytestmark = pytest.mark.destructive

NIL_UUID = "00000000-0000-0000-0000-000000000000"


# ── /users/me ───────────────────────────────────────────────────
def test_get_me(as_user, user):
    r = as_user.get("/api/v1/users/me")
    as_user.expect(r, 200)
    data = as_user.data(r)
    validate(data, USER)
    assert data["email"] == user.email


def test_get_me_requires_auth(client):
    r = client.get("/api/v1/users/me")
    client.expect(r, 401)


def test_update_me_full_name(as_user):
    r = as_user.put("/api/v1/users/me", json={"full_name": "Yangi Ism"})
    as_user.expect(r, 200)
    assert as_user.data(r)["full_name"] == "Yangi Ism"


def test_update_me_cannot_escalate_role(as_user):
    """Privilege escalation: student o'zini mentor qila olmaydi (Role bloklanadi)."""
    r = as_user.put("/api/v1/users/me", json={"full_name": "Talaba Ism", "role": "mentor"})
    as_user.expect(r, 200)
    assert as_user.data(r)["role"] == "student", "ROL KO'TARILDI — privilege escalation!"


# ── Parol o'zgartirish ──────────────────────────────────────────
def test_change_own_password(factory, backend_url):
    u = factory.user(role="student")
    c = factory.client_for(u)
    r = c.put(
        "/api/v1/users/me/password",
        json={"current_password": u.password, "new_password": "yangiparol12345"},
    )
    c.expect(r, 204)
    # Eski parol endi ishlamaydi, yangisi ishlaydi.
    fresh = Client(backend_url)
    assert fresh.post("/api/v1/auth/login", json={"email": u.email, "password": u.password}).status_code == 401
    login(fresh, u.email, "yangiparol12345")  # 200 bo'lmasa xato tashlaydi
    fresh.close()


def test_change_password_wrong_current(as_user):
    r = as_user.put(
        "/api/v1/users/me/password",
        json={"current_password": "notparol999", "new_password": "yangiparol12345"},
    )
    assert r.status_code in (400, 401), r.status_code


# ── Admin CRUD (RBAC) ───────────────────────────────────────────
def test_admin_can_list_users(admin):
    r = admin.get("/api/v1/users")
    admin.expect(r, 200)
    body = r.json()
    for k in ("data", "total", "page", "limit", "total_pages"):
        assert k in body, f"list konvertida {k} yo'q"


def test_student_cannot_list_users(as_user):
    r = as_user.get("/api/v1/users")
    as_user.expect(r, 403)


def test_mentor_cannot_list_users(as_mentor):
    r = as_mentor.get("/api/v1/users")
    as_mentor.expect(r, 403)


def test_student_cannot_create_user(as_user):
    r = as_user.post(
        "/api/v1/users",
        json={"email": "x@darsly.uz", "password": "parol12345", "full_name": "X Y", "role": "student"},
    )
    as_user.expect(r, 403)


def test_admin_can_get_user_by_id(admin, factory):
    u = factory.user(role="student")
    r = admin.get(f"/api/v1/users/{u.user_id}")
    admin.expect(r, 200)
    assert admin.data(r)["email"] == u.email


# ── IDOR ────────────────────────────────────────────────────────
def test_student_cannot_get_other_user_by_id(factory):
    """Begona foydalanuvchini so'rash → 403 (IDOR, admin bo'lmasa)."""
    victim = factory.user(role="student")
    attacker = factory.user(role="student")
    c = factory.client_for(attacker)
    r = c.get(f"/api/v1/users/{victim.user_id}")
    assert r.status_code in (403, 404), f"IDOR: kutilgan 403/404, keldi {r.status_code}"


def test_student_cannot_delete_user(factory):
    victim = factory.user(role="student")
    attacker = factory.user(role="student")
    c = factory.client_for(attacker)
    r = c.delete(f"/api/v1/users/{victim.user_id}")
    assert r.status_code == 403, r.status_code
