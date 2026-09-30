"""LiveKit webhook imzolovchisi — IJOBIY webhook testini LiveKit'siz qilish uchun.

LiveKit webhook `Authorization: <JWT>` header yuboradi. JWT (HS256) claim'lari:
  iss    = API key
  exp/nbf
  sha256 = STANDARD base64(sha256(body))  ← body bilan bog'lash (tampering himoyasi)
Backend `webhook.ReceiveWebhookEvent` shu imzoni sekret bilan tekshiradi — SFU
ishlashini talab qilmaydi. Shuning uchun «to'g'ri imzo → 200» testini shu yerdan
imzolab qilamiz (sekret QA .env / GitHub Secret'da).
"""
from __future__ import annotations

import base64
import hashlib
import hmac
import json
import time


def _b64url(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode()


def webhook_token(api_key: str, api_secret: str, body: bytes) -> str:
    """LiveKit webhook uchun yaroqli imzo tokeni (body bilan bog'langan)."""
    header = {"alg": "HS256", "typ": "JWT"}
    now = int(time.time())
    digest = base64.b64encode(hashlib.sha256(body).digest()).decode()  # standard b64
    claims = {"iss": api_key, "exp": now + 600, "nbf": now - 10, "sha256": digest}
    seg = (
        _b64url(json.dumps(header, separators=(",", ":")).encode())
        + "."
        + _b64url(json.dumps(claims, separators=(",", ":")).encode())
    )
    sig = hmac.new(api_secret.encode(), seg.encode(), hashlib.sha256).digest()
    return seg + "." + _b64url(sig)
