import json
from unittest.mock import MagicMock, patch

import pytest
from fastapi.testclient import TestClient

from app.config import reset_settings


@pytest.fixture(autouse=True)
def set_env(monkeypatch):
    monkeypatch.setenv("API_SERVER_KEY", "test-key")
    monkeypatch.setenv("HERMES_MODEL", "default")
    monkeypatch.setenv("OPENAI_API_KEY", "sk-test")
    reset_settings()
    yield
    reset_settings()


HEADERS = {"Authorization": "Bearer test-key"}


def _fake_agent(response: str):
    agent = MagicMock()
    agent.chat.return_value = response
    return agent


def test_responses_returns_output_text():
    with patch("app.routes.responses.build_agent", return_value=_fake_agent('{"ok": true}')):
        from app.main import app
        client = TestClient(app)
        r = client.post(
            "/v1/responses",
            json={
                "prompt": 'Balas JSON persis ini: {"ok": true}',
                "response_format": {
                    "type": "json_schema",
                    "json_schema": {"schema": {"type": "object", "properties": {"ok": {"type": "boolean"}}}},
                },
            },
            headers=HEADERS,
        )
    assert r.status_code == 200
    body = r.json()
    assert body["output_text"] == '{"ok": true}'


def test_responses_mode_uses_no_toolsets():
    """Factory dipanggil dgn mode='responses' (toolsets off, skip_memory=True)."""
    captured: dict = {}

    def fake_build(**kwargs):
        captured.update(kwargs)
        return _fake_agent('{"x": 1}')

    with patch("app.routes.responses.build_agent", side_effect=fake_build):
        from app.main import app
        client = TestClient(app)
        client.post(
            "/v1/responses",
            json={"prompt": "test", "response_format": {"type": "json_schema", "json_schema": {"schema": {}}}},
            headers=HEADERS,
        )
    assert captured.get("mode") == "responses"
    assert captured.get("ephemeral_system_prompt")


def test_responses_no_auth():
    from app.main import app
    client = TestClient(app)
    r = client.post(
        "/v1/responses",
        json={"prompt": "test", "response_format": {"type": "json_schema", "json_schema": {"schema": {}}}},
    )
    assert r.status_code == 401
