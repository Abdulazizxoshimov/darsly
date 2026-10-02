"""Notifications — list / unread-count / read / read-all (mentor + student).

Kontrakt (api-contract «Notifications»):
  GET  /notifications?unread=&page=&limit=  → list [Notification]
  GET  /notifications/unread-count          → {count}
  POST /notifications/:id/read              → 204
  POST /notifications/read-all              → 204
"""
import pytest

from lib.schemas import NOTIFICATION, UNREAD_COUNT, list_of, validate

NIL_UUID = "00000000-0000-0000-0000-000000000000"


def test_list_notifications_shape(as_user):
    r = as_user.get("/api/v1/notifications")
    as_user.expect(r, 200)
    validate(r.json(), list_of(NOTIFICATION))


def test_list_notifications_unread_filter(as_user):
    r = as_user.get("/api/v1/notifications", params={"unread": "true"})
    as_user.expect(r, 200)
    validate(r.json(), list_of(NOTIFICATION))


def test_unread_count_shape(as_user):
    r = as_user.get("/api/v1/notifications/unread-count")
    as_user.expect(r, 200)
    data = as_user.data(r)
    validate(data, UNREAD_COUNT)
    assert data["count"] >= 0


def test_notifications_require_auth(client):
    assert client.get("/api/v1/notifications").status_code == 401
    assert client.get("/api/v1/notifications/unread-count").status_code == 401


def test_read_all_is_idempotent(as_user):
    # Bildirishnoma bo'lmasa ham 204 (tugma har doim ishlashi kerak).
    r = as_user.post("/api/v1/notifications/read-all")
    as_user.expect(r, 204)


def test_read_nonexistent_notification(as_user):
    r = as_user.post(f"/api/v1/notifications/{NIL_UUID}/read")
    # Boshqa foydalanuvchi / mavjud emas → 404 (yoki 204 idempotent bo'lsa ham xavfsiz).
    assert r.status_code in (204, 404), r.status_code
