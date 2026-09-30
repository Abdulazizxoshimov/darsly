"""Telegram bog'lanishi — status / link / unlink (mentor).

Kontrakt (api-contract «Telegram»):
  GET    /me/telegram        → TelegramLinkStatus {enabled, linked, ...}
  POST   /me/telegram/link   → 201 TelegramLink (integratsiya o'chiq bo'lsa 400)
  DELETE /me/telegram        → 204
Muhitga moslashuvchan: TELEGRAM_BOT_TOKEN yo'q bo'lsa enabled=False, link → 400.
"""
from lib.schemas import TELEGRAM_LINK_STATUS, validate


def test_telegram_status_shape(as_mentor):
    r = as_mentor.get("/api/v1/me/telegram")
    as_mentor.expect(r, 200)
    data = as_mentor.data(r)
    validate(data, TELEGRAM_LINK_STATUS)
    assert isinstance(data["enabled"], bool)
    assert isinstance(data["linked"], bool)


def test_telegram_requires_auth(client):
    client.expect(client.get("/api/v1/me/telegram"), 401)


def test_telegram_link_adapts_to_integration_flag(as_mentor):
    enabled = as_mentor.data(as_mentor.get("/api/v1/me/telegram"))["enabled"]
    r = as_mentor.post("/api/v1/me/telegram/link")
    if enabled:
        as_mentor.expect(r, 201)
        assert as_mentor.data(r)["code"]
    else:
        as_mentor.expect(r, 400)  # "integration disabled"
