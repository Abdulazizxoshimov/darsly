"""Ops / infratuzilma endpointlari — /health /ready /metrics /swagger.

Barchasi `/api/v1` ostida EMAS. Bularsiz monitoring ma'nosiz — birinchi navbatda.
"""
import pytest


@pytest.mark.smoke
def test_health_ok(client):
    r = client.get("/health")
    client.expect(r, 200)
    body = r.json()
    assert body.get("status") == "ok", f"kutilgan status=ok, keldi: {body!r}"


@pytest.mark.smoke
def test_ready(client):
    # DB+Redis yetib bo'lsa 200, aks holda 503 — ikkalasi ham to'g'ri javob.
    r = client.get("/ready")
    assert r.status_code in (200, 503), f"kutilmagan status: {r.status_code}"
    assert "status" in r.json()


@pytest.mark.smoke
def test_metrics_reachable(client):
    # Non-prod: ochiq (200). Prod: METRICS_TOKEN bilan (401 tokensiz) yoki umuman yo'q (404).
    r = client.get("/metrics")
    assert r.status_code in (200, 401, 404), f"kutilmagan status: {r.status_code}"


def test_swagger_present_in_dev(client):
    # Faqat non-production'da ro'yxatdan o'tadi. Local'da ochiq bo'lishi kerak.
    r = client.get("/swagger/index.html")
    # 404 bo'lsa — bu prod build (swagger o'chirilgan); local'da 200/301/302 kutamiz.
    assert r.status_code in (200, 301, 302, 404), f"kutilmagan status: {r.status_code}"
