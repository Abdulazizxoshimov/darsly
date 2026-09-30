"""LiveKit webhook — imzo tekshiruvi (KRITIK xavfsizlik testi).

Kontrakt (api-contract + router): POST /webhooks/livekit (ochiq, imzo bilan).
  to'g'ri imzo → 200 · imzosiz / noto'g'ri sekret / buzilgan tana / axlat → 401
Imzo LiveKit webhook JWT (HS256 + sha256(body) claim) — lib/livekit_sign.py.
SFU ishlashini TALAB qilmaydi (faqat sekret bilan tekshiriladi).
"""
import os

from lib.livekit_sign import webhook_token

KEY = os.environ.get("QA_LIVEKIT_WEBHOOK_KEY", "devkey")
SECRET = os.environ.get("QA_LIVEKIT_WEBHOOK_SECRET", "secret_at_least_32_characters_long_000000")

BODY = b'{"event":"egress_ended","egressInfo":{"egressId":"EG_qa_probe","status":"EGRESS_COMPLETE"}}'
CT = {"Content-Type": "application/webhook+json"}


def test_webhook_no_signature_rejected(client):
    r = client.post("/api/v1/webhooks/livekit", content=BODY, headers=CT)
    client.expect(r, 401)


def test_webhook_valid_signature_accepted(client):
    tok = webhook_token(KEY, SECRET, BODY)
    r = client.post("/api/v1/webhooks/livekit", content=BODY, headers={**CT, "Authorization": tok})
    client.expect(r, 200)


def test_webhook_tampered_body_rejected(client):
    """Imzo to'g'ri, lekin tana o'zgartirilgan → 401 (sha256 claim body bilan bog'langan)."""
    tok = webhook_token(KEY, SECRET, BODY)
    r = client.post("/api/v1/webhooks/livekit", content=BODY + b"tampered", headers={**CT, "Authorization": tok})
    client.expect(r, 401)


def test_webhook_wrong_secret_rejected(client):
    tok = webhook_token(KEY, "wrong_secret_wrong_secret_wrong_00000000", BODY)
    r = client.post("/api/v1/webhooks/livekit", content=BODY, headers={**CT, "Authorization": tok})
    client.expect(r, 401)


def test_webhook_garbage_token_rejected(client):
    r = client.post("/api/v1/webhooks/livekit", content=BODY, headers={**CT, "Authorization": "not.a.valid.jwt"})
    client.expect(r, 401)
