"""App-config — mobil klient ishga tushishida min_version/force_update oladi (ochiq).

Kontrakt (api-contract): GET /app-config → {android:{min_version,latest_version,
apk_url,force_update,release_notes}} (+ top-level allow_open_registration).
"""
import pytest

from lib.schemas import validate, APP_CONFIG

ANDROID_FIELDS = ("min_version", "latest_version", "apk_url", "force_update", "release_notes")


@pytest.mark.smoke
def test_app_config_shape(client):
    r = client.get("/api/v1/app-config")
    client.expect(r, 200)
    data = client.data(r)
    validate(data, APP_CONFIG)
    android = data["android"]
    for k in ANDROID_FIELDS:
        assert k in android, f"android.{k} javobda yo'q: {android!r}"
    assert isinstance(android["force_update"], bool)


@pytest.mark.smoke
def test_app_config_no_auth_required(client):
    # Auth'dan OLDIN chaqiriladi — token bo'lmasa ham 200.
    r = client.get("/api/v1/app-config")
    assert r.status_code == 200


@pytest.mark.smoke
def test_app_config_exposes_open_registration_flag(client):
    # Klient/QA registratsiya yopiq-ochiqligini shu bayroqdan biladi.
    data = client.data(client.get("/api/v1/app-config"))
    assert "allow_open_registration" in data
    assert isinstance(data["allow_open_registration"], bool)
