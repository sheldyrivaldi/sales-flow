import pytest
from fastapi.testclient import TestClient

from app.config import reset_settings


@pytest.fixture(autouse=True)
def set_env(monkeypatch):
    monkeypatch.setenv("API_SERVER_KEY", "test-key")
    monkeypatch.setenv("HERMES_MODEL", "anthropic/claude-sonnet-4-6")
    reset_settings()
    yield
    reset_settings()


def test_capabilities_returns_version_and_models():
    from app.main import app
    client = TestClient(app)
    r = client.get("/v1/capabilities")
    assert r.status_code == 200
    body = r.json()
    assert body["version"]  # non-kosong
    assert len(body["models"]) > 0
    assert "chat" in body["features"]


def test_capabilities_model_matches_env():
    from app.main import app
    client = TestClient(app)
    r = client.get("/v1/capabilities")
    body = r.json()
    assert "anthropic/claude-sonnet-4-6" in body["models"]


def test_provider_error_returns_502():
    """Saat provider/library melempar exception, worker tetap hidup & balas 502."""
    from unittest.mock import patch
    from app.main import app
    client = TestClient(app, raise_server_exceptions=False)
    with patch("app.routes.chat.build_agent", side_effect=RuntimeError("provider mati")):
        r = client.post(
            "/v1/chat/completions",
            json={"model": "default", "messages": [{"role": "user", "content": "test"}]},
            headers={"Authorization": "Bearer test-key"},
        )
    assert r.status_code == 502
    assert "error" in r.json()
    # Worker tetap hidup — request berikutnya ok
    r2 = client.get("/health")
    assert r2.status_code == 200
