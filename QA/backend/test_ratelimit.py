"""Rate-limit — login IP+email chegarasi (429).

⚠️ Bu testlar DESTRUKTIV + SLOW: real 429 chiqarish uchun endpoint'ni bir necha
marta uradi. Prod-monitor profilida AVTO-SKIP (limitni buzmaslik uchun — conftest).
To'g'ri parol ishlatiladi (lockout EMAS — faqat rate-limit); hisob throwaway.
"""
import pytest

from lib.client import Client

pytestmark = [pytest.mark.slow, pytest.mark.destructive]


def test_login_rate_limited_per_ip_email(factory, backend_url):
    u = factory.user(role="student")
    c = Client(backend_url)
    codes = []
    try:
        for _ in range(15):  # IP+email 10/min chegarasidan oshadi
            codes.append(
                c.post("/api/v1/auth/login", json={"email": u.email, "password": u.password}).status_code
            )
    finally:
        c.close()
    assert 429 in codes, f"429 (login IP+email rate-limit) kutildi, keldi: {codes}"
    # To'g'ri parol — lockout emas; dastlabki urinishlar muvaffaqiyatli (200) bo'lishi kerak.
    assert 200 in codes, f"kamida bitta 200 kutildi (to'g'ri parol), keldi: {codes}"
