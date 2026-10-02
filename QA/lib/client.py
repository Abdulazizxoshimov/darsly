"""httpx wrapper — Darsly backend'ining {data}/{code,message} konvertini biladi.

Yo'llar TO'LIQ beriladi (masalan "/api/v1/auth/login" yoki "/health") —
yashirin prefiks yo'q, chunki /health, /ready, /metrics `/api/v1` ostida EMAS.
"""
from __future__ import annotations

import httpx


class ApiError(AssertionError):
    """Kutilmagan status — xato konvertini (code/message) ko'rsatadi."""


class Client:
    def __init__(self, base_url: str, token: str | None = None, timeout: float = 20.0):
        self.base_url = base_url.rstrip("/")
        self.token = token
        self._http = httpx.Client(base_url=self.base_url, timeout=timeout)

    # ── past-daraja ────────────────────────────────────────────
    def _headers(self, extra: dict | None = None) -> dict:
        h = {}
        if self.token:
            h["Authorization"] = f"Bearer {self.token}"
        if extra:
            h.update(extra)
        return h

    def request(self, method: str, path: str, **kw) -> httpx.Response:
        headers = self._headers(kw.pop("headers", None))
        return self._http.request(method, path, headers=headers, **kw)

    def get(self, path, **kw):
        return self.request("GET", path, **kw)

    def post(self, path, **kw):
        return self.request("POST", path, **kw)

    def patch(self, path, **kw):
        return self.request("PATCH", path, **kw)

    def put(self, path, **kw):
        return self.request("PUT", path, **kw)

    def delete(self, path, **kw):
        return self.request("DELETE", path, **kw)

    # ── konvert yordamchilari ──────────────────────────────────
    def set_token(self, token: str | None):
        self.token = token
        return self

    @staticmethod
    def data(resp: httpx.Response):
        """Success javobidan {data} ni ochadi."""
        body = resp.json()
        assert "data" in body, f"'data' maydoni yo'q: {body!r}"
        return body["data"]

    @staticmethod
    def error_code(resp: httpx.Response) -> str:
        """Xato javobidan {code} ni oladi."""
        try:
            return resp.json().get("code", "")
        except Exception:
            return ""

    def expect(self, resp: httpx.Response, status: int) -> httpx.Response:
        """Statusni tekshiradi; mos kelmasa xato konvertini ko'rsatib yiqiladi."""
        if resp.status_code != status:
            body = resp.text[:500]
            raise ApiError(
                f"{resp.request.method} {resp.request.url} → {resp.status_code} "
                f"(kutilgan {status}); javob: {body}"
            )
        return resp

    def close(self):
        self._http.close()
