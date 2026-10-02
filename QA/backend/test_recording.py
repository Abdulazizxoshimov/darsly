"""Recording — list / start / stop / download / get / restore.

Kontrakt (api-contract «Recording»):
  POST /lessons/:id/recording/start → 201 Recording (dars LIVE bo'lishi kerak; scheduled → 400)
  GET  /lessons/:id/recordings      → {data:[Recording]}
  POST /recordings/:id/stop         → 204
  GET  /recordings/:id/download     → RecordingDownload
  GET  /recordings/:id              → Recording (yo'q → 404)
  POST /recordings/:id/restore      → 202
Start ijobiy LIVE dars + LiveKit egress talab qiladi → hozircha negativ (400 "not live")
va auth/RBAC bilan qoplanadi.
"""
import pytest

from lib.schemas import RECORDING, data_array, validate

pytestmark = pytest.mark.destructive

NIL_UUID = "00000000-0000-0000-0000-000000000000"


# ── List ────────────────────────────────────────────────────────
def test_list_recordings_shape(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    r = as_mentor.get(f"/api/v1/lessons/{lesson['id']}/recordings")
    as_mentor.expect(r, 200)
    validate(r.json(), data_array(RECORDING))


def test_list_recordings_requires_auth(client):
    client.expect(client.get(f"/api/v1/lessons/{NIL_UUID}/recordings"), 401)


def test_list_recordings_student_forbidden(as_user):
    as_user.expect(as_user.get(f"/api/v1/lessons/{NIL_UUID}/recordings"), 403)


# ── Start ───────────────────────────────────────────────────────
def test_start_recording_on_scheduled_lesson_rejected(as_mentor, factory):
    """Dars LIVE emas (scheduled) → 400 "lesson is not live"."""
    lesson = factory.lesson(as_mentor, is_recording_enabled=True)
    r = as_mentor.post(f"/api/v1/lessons/{lesson['id']}/recording/start")
    as_mentor.expect(r, 400)


def test_start_recording_student_forbidden(as_user):
    as_user.expect(as_user.post(f"/api/v1/lessons/{NIL_UUID}/recording/start"), 403)


def test_start_recording_other_mentor_forbidden(factory):
    owner = factory.mentor_client()
    other = factory.mentor_client()
    lesson = factory.lesson(owner, is_recording_enabled=True)
    r = other.post(f"/api/v1/lessons/{lesson['id']}/recording/start")
    assert r.status_code in (403, 404), r.status_code


# ── Get / stop / download / restore — mavjud emas → 404 ─────────
def test_get_nonexistent_recording(as_mentor):
    assert as_mentor.get(f"/api/v1/recordings/{NIL_UUID}").status_code == 404


def test_stop_nonexistent_recording(as_mentor):
    assert as_mentor.post(f"/api/v1/recordings/{NIL_UUID}/stop").status_code == 404


def test_download_nonexistent_recording(as_mentor):
    assert as_mentor.get(f"/api/v1/recordings/{NIL_UUID}/download").status_code == 404


def test_recordings_endpoints_require_auth(client):
    assert client.get(f"/api/v1/recordings/{NIL_UUID}").status_code == 401
    assert client.post(f"/api/v1/recordings/{NIL_UUID}/stop").status_code == 401
