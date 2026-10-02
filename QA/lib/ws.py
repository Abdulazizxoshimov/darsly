"""WebSocket yordamchilari — /ws (mentor) va /ws/waitingroom (guest).

Brauzer WS header yubora olmagani uchun backend tokenni `?token=` / `?request_id=`
query'da kutadi (CLAUDE.md «WebSocket»). Bu yordamchi shuni takrorlaydi.

Ishlatish (async):
    async with ws_connect(ws_url_for(base_url) + f"/api/v1/ws?token={jwt}") as ws:
        msg = await recv_json(ws, timeout=5)
        assert msg["type"] == "notification"
"""
from __future__ import annotations

import json
from contextlib import asynccontextmanager

import websockets


def ws_url_for(base_url: str) -> str:
    """http(s)://host → ws(s)://host. QA_WS_URL bo'lsa u ustun (conftest beradi)."""
    b = base_url.rstrip("/")
    if b.startswith("https://"):
        return "wss://" + b[len("https://"):]
    if b.startswith("http://"):
        return "ws://" + b[len("http://"):]
    return b


def mentor_ws_url(base_url: str, token: str) -> str:
    return f"{ws_url_for(base_url)}/api/v1/ws?token={token}"


def waitingroom_ws_url(base_url: str, request_id: str) -> str:
    return f"{ws_url_for(base_url)}/api/v1/ws/waitingroom?request_id={request_id}"


@asynccontextmanager
async def ws_connect(url: str, **kw):
    """websockets.connect ustidan yupqa qobiq (default timeout bilan)."""
    kw.setdefault("open_timeout", 5)
    async with websockets.connect(url, **kw) as ws:
        yield ws


async def recv_json(ws, timeout: float = 5.0) -> dict:
    """Bitta xabarni JSON sifatida oladi (timeout bilan) — async."""
    import asyncio

    raw = await asyncio.wait_for(ws.recv(), timeout=timeout)
    return json.loads(raw)


# ── Sinxron yordamchilar (websockets.sync.client bilan — pytest-asyncio shart emas) ──


def sync_connect(url: str, open_timeout: float = 5.0):
    """websockets.sync.client.connect qobig'i (test with-bloki uchun)."""
    from websockets.sync.client import connect

    return connect(url, open_timeout=open_timeout)


def wait_for_type(ws, wanted: str, timeout: float = 6.0) -> dict:
    """WS'dan `type == wanted` xabari kelguncha o'qiydi (pending snapshot'larni o'tkazadi).

    Ulanishda avval boshqa xabarlar (snapshot) kelishi mumkin — kerakligini kutamiz.
    Umumiy `timeout` ichida topilmasa AssertionError.
    """
    import time

    deadline = time.monotonic() + timeout
    seen = []
    while time.monotonic() < deadline:
        remaining = max(0.1, deadline - time.monotonic())
        try:
            raw = ws.recv(timeout=remaining)
        except TimeoutError:
            break
        msg = json.loads(raw)
        if msg.get("type") == wanted:
            return msg
        seen.append(msg.get("type"))
    raise AssertionError(f"WS '{wanted}' xabari {timeout}s ichida kelmadi; ko'rilgan turlar: {seen}")
