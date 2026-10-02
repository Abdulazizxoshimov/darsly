"""Dars CRUD + egalik (ownership/IDOR) + validatsiya + RBAC.

Kontrakt (api-contract «Lessons»):
  POST   /lessons        CreateLessonReq → 201 Lesson
  GET    /lessons        → list konverti [Lesson]
  GET    /lessons/:id    → Lesson (begona mentor → 403/404)
  PATCH  /lessons/:id    UpdateLessonReq → Lesson
  DELETE /lessons/:id    → 204
"""
import pytest

from lib.schemas import LESSON, list_of, validate

pytestmark = pytest.mark.destructive  # dars yaratadi → local/ephemeral

NIL_UUID = "00000000-0000-0000-0000-000000000000"


# ── Yaratish ────────────────────────────────────────────────────
def test_create_lesson(as_mentor, factory):
    data = factory.lesson(as_mentor, title="Algebra 5-sinf")
    validate(data, LESSON)
    assert data["title"] == "Algebra 5-sinf"
    assert data["status"] == "scheduled"
    assert data["join_slug"], "join_slug bo'sh bo'lmasligi kerak"


def test_create_lesson_requires_title(as_mentor):
    r = as_mentor.post(
        "/api/v1/lessons",
        json={"duration_min": 30, "is_recording_enabled": False, "is_waiting_room_enabled": False},
    )
    as_mentor.expect(r, 400)


def test_create_lesson_requires_auth(client):
    r = client.post("/api/v1/lessons", json={"title": "x", "duration_min": 30})
    client.expect(r, 401)


def test_student_cannot_create_lesson(as_user):
    # RBAC: dars yaratish mentor huquqi.
    r = as_user.post(
        "/api/v1/lessons",
        json={"title": "x", "duration_min": 30, "is_recording_enabled": False, "is_waiting_room_enabled": False},
    )
    as_user.expect(r, 403)


# ── Ro'yxat ─────────────────────────────────────────────────────
def test_list_lessons(as_mentor, factory):
    factory.lesson(as_mentor)
    r = as_mentor.get("/api/v1/lessons")
    as_mentor.expect(r, 200)
    validate(r.json(), list_of(LESSON))
    assert r.json()["total"] >= 1


def test_list_lessons_requires_auth(client):
    r = client.get("/api/v1/lessons")
    client.expect(r, 401)


# ── O'qish + egalik (IDOR) ──────────────────────────────────────
def test_get_own_lesson(as_mentor, factory):
    data = factory.lesson(as_mentor)
    r = as_mentor.get(f"/api/v1/lessons/{data['id']}")
    as_mentor.expect(r, 200)
    assert as_mentor.data(r)["id"] == data["id"]


def test_get_other_mentors_lesson_forbidden(factory):
    """Begona mentor darsni ololmaydi (join_slug leak bloklangan — CLAUDE.md IDOR)."""
    owner = factory.mentor_client()
    other = factory.mentor_client()
    data = factory.lesson(owner)
    r = other.get(f"/api/v1/lessons/{data['id']}")
    assert r.status_code in (403, 404), f"IDOR: kutilgan 403/404, keldi {r.status_code}"


def test_get_nonexistent_lesson(as_mentor):
    r = as_mentor.get(f"/api/v1/lessons/{NIL_UUID}")
    assert r.status_code == 404, r.status_code


# ── Yangilash + o'chirish ───────────────────────────────────────
def test_update_own_lesson(as_mentor, factory):
    data = factory.lesson(as_mentor)
    r = as_mentor.patch(f"/api/v1/lessons/{data['id']}", json={"title": "Yangilangan sarlavha"})
    as_mentor.expect(r, 200)
    assert as_mentor.data(r)["title"] == "Yangilangan sarlavha"


def test_update_other_mentors_lesson_forbidden(factory):
    owner = factory.mentor_client()
    other = factory.mentor_client()
    data = factory.lesson(owner)
    r = other.patch(f"/api/v1/lessons/{data['id']}", json={"title": "hack"})
    assert r.status_code in (403, 404), r.status_code


def test_delete_own_lesson(as_mentor, factory):
    data = factory.lesson(as_mentor)
    r = as_mentor.delete(f"/api/v1/lessons/{data['id']}")
    as_mentor.expect(r, 204)
    # O'chirilgach topilmasligi kerak.
    assert as_mentor.get(f"/api/v1/lessons/{data['id']}").status_code == 404


def test_delete_other_mentors_lesson_forbidden(factory):
    owner = factory.mentor_client()
    other = factory.mentor_client()
    data = factory.lesson(owner)
    r = other.delete(f"/api/v1/lessons/{data['id']}")
    assert r.status_code in (403, 404), r.status_code
