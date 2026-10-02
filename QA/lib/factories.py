"""Test ma'lumot fabrikasi — user/lesson yaratadi va o'zidan keyin TOZALAYDI.

Har test uchun bitta `Factory` (conftest `factory` fixture) olinadi; u yaratgan
barcha resurslarni teardown'da o'chiradi (yetim ma'lumot qolmaydi). Prod-monitor
profilida DESTRUKTIV fabrikalar ISHLATILMAYDI (faqat `smoke` testlar).

Kontrakt (docs/api-contract.md):
  POST /lessons  (mentor JWT)  CreateLessonReq  → 201 Lesson
  DELETE /lessons/:id (egasi)                    → 204
  POST /users    (admin JWT)                     → 201 User
  DELETE /users/:id (admin)                      → 204
"""
from __future__ import annotations

from .auth import TestUser, create_user
from .client import Client


class Factory:
    def __init__(self, admin: Client, base_url: str):
        self.admin = admin
        self.base_url = base_url
        self._lessons: list[tuple[Client, str]] = []  # (egasi mijoz, lesson_id)
        self._user_ids: list[str] = []
        self._clients: list[Client] = []  # client_for yaratganlar — cleanup'da yopiladi

    # ── user ─────────────────────────────────────────────────────
    def user(self, role: str = "student") -> TestUser:
        u = create_user(self.admin, role=role)
        if u.user_id:
            self._user_ids.append(u.user_id)
        return u

    def mentor(self) -> TestUser:
        return self.user(role="mentor")

    def client_for(self, u: TestUser) -> Client:
        """Foydalanuvchi tokeni bilan mijoz (teardown'da avto-yopiladi)."""
        c = Client(self.base_url, token=u.access)
        self._clients.append(c)
        return c

    def mentor_client(self) -> Client:
        """Yangi mentor + uning mijozi (IDOR/ownership testlari uchun)."""
        return self.client_for(self.mentor())

    # ── lesson ───────────────────────────────────────────────────
    def lesson(self, owner: Client, **overrides) -> dict:
        """Mentor mijozi bilan dars yaratadi va tozalash uchun ro'yxatga oladi."""
        payload = {
            "title": "QA dars",
            "duration_min": 30,
            "is_recording_enabled": False,
            "is_waiting_room_enabled": False,
        }
        payload.update(overrides)
        resp = owner.post("/api/v1/lessons", json=payload)
        owner.expect(resp, 201)
        data = owner.data(resp)
        self._lessons.append((owner, data["id"]))
        return data

    # ── tozalash ─────────────────────────────────────────────────
    def cleanup(self) -> None:
        # Avval darslar (egasi bilan), keyin userlar (admin bilan). Xato yutiladi —
        # tozalash test natijasini buzmasin (masalan dars allaqachon o'chgan).
        for owner, lesson_id in reversed(self._lessons):
            try:
                owner.delete(f"/api/v1/lessons/{lesson_id}")
            except Exception:
                pass
        for uid in reversed(self._user_ids):
            try:
                self.admin.delete(f"/api/v1/users/{uid}")
            except Exception:
                pass
        for c in self._clients:
            try:
                c.close()
            except Exception:
                pass
        self._lessons.clear()
        self._user_ids.clear()
        self._clients.clear()
