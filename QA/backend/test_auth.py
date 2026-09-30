"""Auth oqimi — register / login / refresh / logout / me.

Test foydalanuvchilar admin API orqali yaratiladi (`user`/`admin` fixture, conftest.py),
chunki bu muhitda ochiq registratsiya YOPIQ. Register endpointining o'zi esa
`allow_open_registration` bayrog'iga qarab moslashuvchan tekshiriladi.

Kontrakt (13-bo'lim, README): TokenPair {access_token, refresh_token}; xato {code, message}.
"""
import pytest

from lib.auth import login, open_registration_enabled, unique_email

pytestmark = pytest.mark.destructive  # foydalanuvchi yaratadi → local/ephemeral


# ── Register (bayroqqa moslashuvchan) ──────────────────────────
def test_register_respects_open_registration_flag(client):
    """Ochiq registratsiya yopiq bo'lsa 403, yoqiq bo'lsa 201 + TokenPair."""
    payload = {"full_name": "QA Reg", "email": unique_email(), "password": "parol12345"}
    r = client.post("/api/v1/auth/register", json=payload)

    if open_registration_enabled(client):
        client.expect(r, 201)
        data = client.data(r)
        assert data["access_token"] and data["refresh_token"]
    else:
        client.expect(r, 403)
        assert client.error_code(r)  # xato kodi bo'sh emas


# ── Login ──────────────────────────────────────────────────────
def test_login_success(user, client):
    data = login(client, user.email, user.password)
    assert data["access_token"] and data["refresh_token"]


def test_login_wrong_password(user, client):
    r = client.post("/api/v1/auth/login", json={"email": user.email, "password": "notparol999"})
    client.expect(r, 401)


def test_login_nonexistent_user(client):
    # Enumeration himoyasi: yo'q user ham 401 (mavjud user xato paroli bilan bir xil).
    r = client.post("/api/v1/auth/login", json={"email": unique_email(), "password": "parol12345"})
    client.expect(r, 401)


# ── Refresh ────────────────────────────────────────────────────
def test_refresh_rotates_tokens(user, client):
    r = client.post("/api/v1/auth/refresh", json={"refresh_token": user.refresh})
    client.expect(r, 200)
    data = client.data(r)
    assert data["access_token"] and data["refresh_token"]


def test_refresh_invalid_token(client):
    r = client.post("/api/v1/auth/refresh", json={"refresh_token": "chala.token.qiymat"})
    client.expect(r, 401)


# ── /auth/me ───────────────────────────────────────────────────
def test_me_requires_auth(client):
    r = client.get("/api/v1/auth/me")
    client.expect(r, 401)


def test_me_returns_current_user(as_user, user):
    r = as_user.get("/api/v1/auth/me")
    as_user.expect(r, 200)
    data = as_user.data(r)
    assert data["email"] == user.email
    assert "id" in data


# ── Logout ─────────────────────────────────────────────────────
def test_logout_then_refresh_revoked(backend_url, user):
    from lib.client import Client

    c = Client(backend_url, token=user.access)
    r = c.post("/api/v1/auth/logout", json={"refresh_token": user.refresh})
    c.expect(r, 204)

    # Sessiya o'chgach eski refresh ishlamasligi kerak.
    r2 = c.post("/api/v1/auth/refresh", json={"refresh_token": user.refresh})
    c.expect(r2, 401)
    c.close()


# ── Forgot / reset (enumeration yo'q) ──────────────────────────
def test_forgot_password_always_204(client):
    # Mavjud bo'lmagan email ham 204 — user borligini oshkor qilmaydi.
    r = client.post("/api/v1/auth/forgot-password", json={"email": unique_email()})
    client.expect(r, 204)


def test_reset_password_invalid_token(client):
    r = client.post(
        "/api/v1/auth/reset-password",
        json={"token": "yaroqsiz-token", "new_password": "yangiparol123"},
    )
    assert r.status_code in (400, 401), f"kutilmagan status: {r.status_code}"
