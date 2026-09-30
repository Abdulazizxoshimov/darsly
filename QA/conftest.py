"""Global pytest fixture'lar — barcha QA testlari uchun.

Profil `QA_BASE_URL` env bilan tanlanadi (default: local http://localhost:8087).
Backend ishlamayotgan bo'lsa — barcha testlar aniq sabab bilan SKIP qilinadi.
"""
from __future__ import annotations

import os
import sys
from pathlib import Path

import httpx
import pytest

# QA/ ni import yo'liga qo'shamiz (lib.* ishlashi uchun)
sys.path.insert(0, str(Path(__file__).parent))

try:
    from dotenv import load_dotenv

    load_dotenv(Path(__file__).parent / ".env")
except Exception:
    pass

from lib.auth import admin_client, create_user  # noqa: E402
from lib.client import Client  # noqa: E402
from lib.factories import Factory  # noqa: E402
from lib.ws import ws_url_for  # noqa: E402


def base_url() -> str:
    return os.environ.get("QA_BASE_URL", "http://localhost:8087").rstrip("/")


def profile() -> str:
    """Faol profil: local | ephemeral | prod-monitor.

    QA_PROFILE aniq berilsa u; aks holda URL'dan taxmin (https + sslip/jonly → prod).
    """
    p = os.environ.get("QA_PROFILE", "").strip().lower()
    if p:
        return p
    url = base_url()
    if url.startswith("https://") and ("localhost" not in url and "127.0.0.1" not in url):
        return "prod-monitor"
    return "local"


def _is_prod_monitor() -> bool:
    return profile() == "prod-monitor"


def pytest_collection_modifyitems(config, items):
    """Prod-monitor profilida DESTRUKTIV testlarni skip qiladi.

    Prod'da faqat `smoke` (o'qish yoki o'z-ustidan tozalanadigan) ishlaydi —
    README §4 «Prod'ga destruktiv test yo'q» qoidasi shu yerda majburlanadi.
    """
    if not _is_prod_monitor():
        return
    skip_destructive = pytest.mark.skip(reason="prod-monitor profilida destruktiv test o'tkazib yuboriladi")
    for item in items:
        if "destructive" in item.keywords and "smoke" not in item.keywords:
            item.add_marker(skip_destructive)


@pytest.fixture(scope="session")
def backend_url() -> str:
    return base_url()


@pytest.fixture(scope="session")
def qa_profile() -> str:
    return profile()


@pytest.fixture(scope="session")
def ws_url(backend_url) -> str:
    """WebSocket bazasi — QA_WS_URL bo'lsa u, aks holda URL'dan almashtiriladi."""
    return os.environ.get("QA_WS_URL", "").rstrip("/") or ws_url_for(backend_url)


@pytest.fixture(scope="session", autouse=True)
def _require_backend(backend_url):
    """Backend yetib bo'lmasa — hamma testni tushunarli sabab bilan skip qiladi."""
    try:
        r = httpx.get(f"{backend_url}/health", timeout=3.0)
        if r.status_code != 200:
            pytest.skip(f"backend /health {r.status_code} qaytardi ({backend_url})", allow_module_level=False)
    except Exception as e:
        pytest.skip(
            f"backend yetib bo'lmadi: {backend_url} ({e}). "
            f"Ishga tushiring: cd backend && docker compose -f docker-compose.dev.yml up -d && go run ./cmd",
        )


@pytest.fixture
def client(backend_url) -> Client:
    """Har test uchun yangi (tokensiz) mijoz."""
    c = Client(backend_url)
    yield c
    c.close()


@pytest.fixture(scope="session")
def admin(backend_url) -> Client:
    """Seed admin bilan kirilgan mijoz (SESSION-scoped).

    Butun sessiyada BIR MARTA login qiladi: aks holda har test seed-admin bilan
    qayta kirib `RATE_LIMIT_LOGIN_IP_EMAIL` (10/min) chegarasiga urardi (429) va
    «bitta akkaunt = bitta sessiya» qoidasi eski admin tokenini bekor qilardi.
    """
    c = admin_client(backend_url)
    yield c
    c.close()


@pytest.fixture
def user(factory):
    """Admin API orqali yaratilgan student (factory orqali → teardown'da tozalanadi)."""
    return factory.user(role="student")


@pytest.fixture
def mentor(factory):
    """Admin API orqali yaratilgan mentor (factory orqali → teardown'da tozalanadi)."""
    return factory.user(role="mentor")


@pytest.fixture
def as_user(backend_url, user) -> Client:
    """user (student) tokeni bilan avtorizatsiyalangan mijoz."""
    c = Client(backend_url, token=user.access)
    yield c
    c.close()


@pytest.fixture
def as_mentor(backend_url, mentor) -> Client:
    """mentor tokeni bilan avtorizatsiyalangan mijoz."""
    c = Client(backend_url, token=mentor.access)
    yield c
    c.close()


@pytest.fixture(scope="session")
def livekit_available(admin, backend_url) -> bool:
    """LiveKit ishlaydimi — host token oldirib bir marta aniqlanadi (session-cache).

    Host token LiveKit SFU'ga borishni talab qiladi; SFU yo'q bo'lsa 500. Shu
    yordamchi bir marta probe qiladi va room-token testlari `requires_livekit`
    orqali LiveKit yo'q bo'lsa TOZA skip bo'ladi (xato emas).
    """
    from lib.client import Client

    m = create_user(admin, role="mentor")
    mc = Client(backend_url, token=m.access)
    try:
        lr = mc.post(
            "/api/v1/lessons",
            json={"title": "LK probe", "duration_min": 30,
                  "is_recording_enabled": False, "is_waiting_room_enabled": False},
        )
        if lr.status_code != 201:
            return False
        lid = lr.json()["data"]["id"]
        ok = mc.post(f"/api/v1/lessons/{lid}/token").status_code == 200
        mc.delete(f"/api/v1/lessons/{lid}")
        return ok
    except Exception:
        return False
    finally:
        try:
            admin.delete(f"/api/v1/users/{m.user_id}")
        except Exception:
            pass
        mc.close()


@pytest.fixture
def requires_livekit(livekit_available):
    """LiveKit yo'q bo'lsa testni skip qiladi (room-token/egress kerak bo'lganda)."""
    if not livekit_available:
        pytest.skip("LiveKit ishlamayapti — room-token/egress testi o'tkazib yuboriladi")


@pytest.fixture
def factory(admin, backend_url):
    """Test ma'lumot fabrikasi — yaratganini teardown'da tozalaydi."""
    f = Factory(admin, backend_url)
    yield f
    f.cleanup()


@pytest.fixture
def lesson(factory, as_mentor):
    """as_mentor egaligidagi yangi dars (tozalanadi)."""
    return factory.lesson(as_mentor)
