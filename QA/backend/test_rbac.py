"""RBAC matritsasi — student mentor/admin route'larida 403 (jadval-asosli).

Casbin auth'dan keyin, handler'dan OLDIN ishlaydi → 403 dummy ID bilan ham keladi
(dars mavjudligiga bog'liq emas). Bu — privilege-escalation regressiya to'sig'i.
"""
import pytest

NIL = "00000000-0000-0000-0000-000000000000"

# Mentor huquqini talab qiluvchi route'lar (student → 403).
MENTOR_ONLY = [
    ("POST", "/api/v1/lessons"),
    ("GET", "/api/v1/lessons"),
    ("GET", "/api/v1/blocklist"),
    ("GET", "/api/v1/me/telegram"),
    ("GET", f"/api/v1/lessons/{NIL}/waitingroom"),
    ("POST", f"/api/v1/lessons/{NIL}/end"),
    ("POST", f"/api/v1/lessons/{NIL}/recording/start"),
    ("GET", f"/api/v1/lessons/{NIL}/recordings"),
    ("GET", f"/api/v1/lessons/{NIL}/polls"),
    ("POST", f"/api/v1/lessons/{NIL}/token"),
]

# FAQAT admin route'lari (student VA mentor → 403).
ADMIN_ONLY = [
    ("GET", "/api/v1/users"),
    ("POST", "/api/v1/users"),
    ("DELETE", f"/api/v1/users/{NIL}"),
    ("POST", f"/api/v1/users/{NIL}/deactivate"),
]


@pytest.mark.parametrize("method,path", MENTOR_ONLY + ADMIN_ONLY)
def test_student_forbidden(as_user, method, path):
    r = as_user.request(method, path)
    assert r.status_code == 403, f"{method} {path}: kutilgan 403, keldi {r.status_code}"


@pytest.mark.parametrize("method,path", ADMIN_ONLY)
def test_mentor_forbidden_on_admin_routes(as_mentor, method, path):
    r = as_mentor.request(method, path)
    assert r.status_code == 403, f"{method} {path}: mentor uchun kutilgan 403, keldi {r.status_code}"


@pytest.mark.parametrize("method,path", MENTOR_ONLY)
def test_mentor_not_forbidden_on_mentor_routes(as_mentor, method, path):
    """Mentor o'z route'larida 403 OLMASLIGI kerak (dummy ID → 400/404 bo'lishi mumkin)."""
    r = as_mentor.request(method, path)
    assert r.status_code != 403, f"{method} {path}: mentor 403 OLDI (RBAC juda qattiq)"
