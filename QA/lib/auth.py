"""Auth yordamchilari — test foydalanuvchi yaratish / kirish.

Bu muhitda ochiq registratsiya odatda YOPIQ (ALLOW_OPEN_REGISTRATION=false):
foydalanuvchilar admin API orqali (POST /users) yaratiladi. Shuning uchun
test foydalanuvchini seed admin bilan yaratamiz — register endpointiga bog'liq emasmiz.

Kontrakt:
  POST /auth/login           {email, password}                    → 200 {data:{access_token, refresh_token}}
  POST /users (admin)        {email, password, full_name, role}    → 201 {data: User}
  POST /auth/register        {full_name, email, password}          → 201 (ochiq bo'lsa) / 403 (yopiq)
  GET  /app-config           → {data:{allow_open_registration}}
"""
from __future__ import annotations

import os
import uuid
from dataclasses import dataclass

from .client import Client


@dataclass
class TestUser:
    email: str
    password: str
    full_name: str
    role: str = "student"
    user_id: str = ""
    access: str = ""
    refresh: str = ""


def unique_email(prefix: str = "qa") -> str:
    return f"{prefix}+{uuid.uuid4().hex[:12]}@darsly.uz"


def open_registration_enabled(client: Client) -> bool:
    r = client.get("/api/v1/app-config")
    if r.status_code != 200:
        return False
    return bool(client.data(r).get("allow_open_registration", False))


def seed_admin_creds() -> tuple[str, str]:
    return (
        os.environ.get("QA_SEED_ADMIN_EMAIL", "admin@darsly.uz"),
        os.environ.get("QA_SEED_ADMIN_PASSWORD", "Admin12345"),
    )


def login(client: Client, email: str, password: str) -> dict:
    resp = client.post("/api/v1/auth/login", json={"email": email, "password": password})
    client.expect(resp, 200)
    return client.data(resp)


def admin_client(base_url: str) -> Client:
    """Seed admin bilan kirilgan mijoz."""
    c = Client(base_url)
    email, pw = seed_admin_creds()
    data = login(c, email, pw)
    c.set_token(data["access_token"])
    return c


def create_user(admin: Client, role: str = "student", password: str = "parol12345") -> TestUser:
    """Admin API orqali foydalanuvchi yaratadi va kirib tokenlarni oladi."""
    email = unique_email()
    resp = admin.post(
        "/api/v1/users",
        json={"email": email, "password": password, "full_name": "QA User", "role": role},
    )
    admin.expect(resp, 201)
    data = admin.data(resp)

    # Yangi mijoz orqali kiramiz (admin tokeni bilan aralashmasin).
    login_client = Client(admin.base_url)
    tokens = login(login_client, email, password)
    login_client.close()

    return TestUser(
        email=email,
        password=password,
        full_name="QA User",
        role=role,
        user_id=data.get("id", ""),
        access=tokens["access_token"],
        refresh=tokens["refresh_token"],
    )
