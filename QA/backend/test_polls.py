"""So'rovnomalar — create / list / close / publish / vote / results.

Kontrakt (api-contract «Polls» + «Poll: ikki rejim»):
  POST /lessons/:id/polls {question,options[2..10],results_visibility?} → 201 Poll
  GET  /lessons/:id/polls                          → {data:[Poll]}
  POST /lessons/:id/polls/:pollID/publish          → PollResults · mentor_only → 400
  POST /polls/:id/close                            → PollResults
  POST /polls/:id/vote  {token,option_index}       → 204 (guest room-token)
  GET  /polls/:id/results?token=                   → PollResults · e'lon qilinmagan → 403
Vote/results guest room-token → LiveKit guard. Qolgani host JWT (guardsiz).
"""
import pytest

from lib import rooms
from lib.schemas import POLL, POLL_RESULTS, data_array, validate

pytestmark = pytest.mark.destructive

NIL_UUID = "00000000-0000-0000-0000-000000000000"


def _poll(as_mentor, factory, **overrides):
    lesson = factory.lesson(as_mentor)
    payload = {"question": "2+2 nechchi?", "options": ["3", "4", "5"]}
    payload.update(overrides)
    data = as_mentor.data(as_mentor.post(f"/api/v1/lessons/{lesson['id']}/polls", json=payload))
    return lesson, data


# ── Create ──────────────────────────────────────────────────────
def test_create_poll(as_mentor, factory):
    _, poll = _poll(as_mentor, factory)
    validate(poll, POLL)
    assert poll["question"] == "2+2 nechchi?"
    assert poll["results_visibility"] == "mentor_only"  # default


def test_create_poll_too_few_options(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    r = as_mentor.post(f"/api/v1/lessons/{lesson['id']}/polls", json={"question": "x?", "options": ["faqat bitta"]})
    as_mentor.expect(r, 400)


def test_create_poll_too_many_options(as_mentor, factory):
    lesson = factory.lesson(as_mentor)
    r = as_mentor.post(
        f"/api/v1/lessons/{lesson['id']}/polls",
        json={"question": "x?", "options": [str(i) for i in range(11)]},
    )
    as_mentor.expect(r, 400)


def test_create_poll_student_forbidden(as_user):
    as_user.expect(as_user.post(f"/api/v1/lessons/{NIL_UUID}/polls", json={"question": "x?", "options": ["a", "b"]}), 403)


def test_create_poll_requires_auth(client):
    client.expect(client.post(f"/api/v1/lessons/{NIL_UUID}/polls", json={"question": "x?", "options": ["a", "b"]}), 401)


# ── List ────────────────────────────────────────────────────────
def test_list_polls(as_mentor, factory):
    lesson, _ = _poll(as_mentor, factory)
    r = as_mentor.get(f"/api/v1/lessons/{lesson['id']}/polls")
    as_mentor.expect(r, 200)
    validate(r.json(), data_array(POLL))
    assert len(as_mentor.data(r)) >= 1


# ── Close ───────────────────────────────────────────────────────
def test_close_poll(as_mentor, factory):
    _, poll = _poll(as_mentor, factory)
    r = as_mentor.post(f"/api/v1/polls/{poll['id']}/close")
    as_mentor.expect(r, 200)
    validate(as_mentor.data(r), POLL_RESULTS)


# ── Publish ─────────────────────────────────────────────────────
def test_publish_mentor_only_rejected(as_mentor, factory):
    """mentor_only so'rovnoma natijasini e'lon qilib BO'LMAYDI → 400."""
    lesson, poll = _poll(as_mentor, factory)  # default mentor_only
    r = as_mentor.post(f"/api/v1/lessons/{lesson['id']}/polls/{poll['id']}/publish")
    as_mentor.expect(r, 400)


def test_publish_public_poll(as_mentor, factory):
    lesson, poll = _poll(as_mentor, factory, results_visibility="public")
    r = as_mentor.post(f"/api/v1/lessons/{lesson['id']}/polls/{poll['id']}/publish")
    as_mentor.expect(r, 200)
    validate(as_mentor.data(r), POLL_RESULTS)


# ── Vote / results (guest room-token) — LiveKit kerak ───────────
def test_vote_requires_token(client):
    client.expect(client.post(f"/api/v1/polls/{NIL_UUID}/vote", json={"option_index": 0}), 401)


def test_results_requires_token(client):
    client.expect(client.get(f"/api/v1/polls/{NIL_UUID}/results"), 401)


def test_vote_and_results(requires_livekit, as_mentor, factory, client):
    lesson, poll = _poll(as_mentor, factory, results_visibility="public")
    tok = rooms.host_token(as_mentor, lesson["id"])["token"]
    # Host token bilan ovoz berish
    client.expect(client.post(f"/api/v1/polls/{poll['id']}/vote", json={"token": tok, "option_index": 1}), 204)
    # Host doim natijani ko'radi
    r = client.get(f"/api/v1/polls/{poll['id']}/results", params={"token": tok})
    client.expect(r, 200)
    validate(client.data(r), POLL_RESULTS)
