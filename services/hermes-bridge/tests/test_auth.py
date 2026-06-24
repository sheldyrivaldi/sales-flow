import os

import pytest
from fastapi import Depends
from fastapi.testclient import TestClient

from app.auth import require_bearer
from app.config import reset_settings
from app.main import app


@pytest.fixture(autouse=True)
def set_api_key(monkeypatch):
    monkeypatch.setenv("API_SERVER_KEY", "test-key-12345")
    reset_settings()
    yield
    reset_settings()


app.get("/test-protected", dependencies=[Depends(require_bearer)])(lambda: {"ok": True})

client = TestClient(app, raise_server_exceptions=True)


def test_no_auth_header():
    r = client.get("/test-protected")
    assert r.status_code == 401


def test_wrong_bearer():
    r = client.get("/test-protected", headers={"Authorization": "Bearer wrong-key"})
    assert r.status_code == 401


def test_correct_bearer():
    r = client.get("/test-protected", headers={"Authorization": "Bearer test-key-12345"})
    assert r.status_code == 200


def test_health_no_auth():
    r = client.get("/health")
    assert r.status_code == 200
    assert r.json() == {"status": "ok"}
