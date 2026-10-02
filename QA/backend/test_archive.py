"""Dars arxivi (№20) — video + chat + materiallar bitta javobda.

Kontrakt (api-contract «Dars arxivi»):
  GET /lessons/:id/archive → LessonArchive {lesson, recording|null, chat[], materials[]}
  Cache-Control: no-store; begona mentor → 403; yo'q dars → 404; student → 403.
LiveKit KERAK EMAS (yozuvsiz darsda recording=null).
"""
import pytest

from lib.schemas import LESSON_ARCHIVE, validate

pytestmark = pytest.mark.destructive

NIL_UUID = "00000000-0000-0000-0000-000000000000"


def test_archive_own_lesson(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    r = as_mentor.get(f"/api/v1/lessons/{lesson['id']}/archive")
    as_mentor.expect(r, 200)
    validate(as_mentor.data(r), LESSON_ARCHIVE)
    assert r.headers.get("Cache-Control") == "no-store", "arxiv no-store bilan kelishi kerak"


def test_archive_other_mentor_forbidden(factory):
    owner = factory.mentor_client()
    other = factory.mentor_client()
    lesson = factory.lesson(owner)
    r = other.get(f"/api/v1/lessons/{lesson['id']}/archive")
    assert r.status_code in (403, 404), r.status_code


def test_archive_nonexistent_lesson(as_mentor):
    r = as_mentor.get(f"/api/v1/lessons/{NIL_UUID}/archive")
    assert r.status_code == 404, r.status_code


def test_archive_requires_auth(client):
    client.expect(client.get(f"/api/v1/lessons/{NIL_UUID}/archive"), 401)


def test_student_cannot_access_archive(as_user, as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    r = as_user.get(f"/api/v1/lessons/{lesson['id']}/archive")
    assert r.status_code == 403, r.status_code
