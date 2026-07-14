from unittest.mock import MagicMock, patch

import pytest
from fastapi.testclient import TestClient

from app.agent_factory import set_active_provider
from app.config import reset_settings


@pytest.fixture(autouse=True)
def set_env(monkeypatch):
    monkeypatch.setenv("API_SERVER_KEY", "test-key")
    monkeypatch.setenv("HERMES_MODEL", "default")
    monkeypatch.setenv("OPENAI_API_KEY", "sk-test")
    reset_settings()
    set_active_provider(None)
    yield
    set_active_provider(None)
    reset_settings()


HEADERS = {"Authorization": "Bearer test-key"}


def test_set_config_no_auth():
    from app.main import app
    client = TestClient(app)
    r = client.post("/admin/config", json={"provider": "openai", "model": "gpt-4o", "api_key": "k"})
    assert r.status_code == 401


def test_set_config_valid():
    from app.main import app
    client = TestClient(app)
    r = client.post(
        "/admin/config",
        json={"provider": "openai", "model": "gpt-4o", "api_key": "sk-real"},
        headers=HEADERS,
    )
    assert r.status_code == 200
    body = r.json()
    assert body["status"] == "ok"
    assert body["provider"] == "openai"


def test_set_config_invalid_provider():
    from app.main import app
    client = TestClient(app)
    r = client.post(
        "/admin/config",
        json={"provider": "anthropic", "model": "claude-3", "api_key": "k"},
        headers=HEADERS,
    )
    assert r.status_code == 422


def test_set_config_openrouter_default_base_url():
    from app.agent_factory import get_active_provider
    from app.main import app
    client = TestClient(app)
    client.post(
        "/admin/config",
        json={"provider": "openrouter", "model": "anthropic/claude-sonnet-4-6", "api_key": "or-k"},
        headers=HEADERS,
    )
    active = get_active_provider()
    assert active is not None
    assert active["base_url"] == "https://openrouter.ai/api/v1"
    assert active["api_key"] == "or-k"


def test_set_config_toolsets_round_trip():
    """enabled_toolsets (EP-18 ST-18.4) round-trips through /admin/config
    POST → GET, and is passed to set_active_provider."""
    from app.agent_factory import get_active_provider
    from app.main import app
    client = TestClient(app)
    client.post(
        "/admin/config",
        json={"provider": "openai", "model": "gpt-4o", "api_key": "k", "enabled_toolsets": ["web", "docs"]},
        headers=HEADERS,
    )

    active = get_active_provider()
    assert active is not None
    assert active["enabled_toolsets"] == ["web", "docs"]

    r = client.get("/admin/config", headers=HEADERS)
    assert r.json()["active"]["enabled_toolsets"] == ["web", "docs"]


def test_set_config_no_toolsets_defaults_to_none():
    from app.agent_factory import get_active_provider
    from app.main import app
    client = TestClient(app)
    client.post(
        "/admin/config",
        json={"provider": "openai", "model": "gpt-4o", "api_key": "k"},
        headers=HEADERS,
    )

    active = get_active_provider()
    assert active is not None
    assert active["enabled_toolsets"] is None


def test_agent_uses_active_config():
    """Setelah set config, build_agent mendapat api_key & model dari config aktif."""
    from app.main import app
    client = TestClient(app)
    client.post(
        "/admin/config",
        json={"provider": "openai", "model": "gpt-4o", "api_key": "override-key"},
        headers=HEADERS,
    )

    captured: dict = {}

    class FakeAgent:
        def __init__(self, **kwargs):
            captured.update(kwargs)

        def run_conversation(self, *args, **kwargs):
            return {"final_response": "ok", "messages": []}

    with patch("app.agent_factory.AIAgent", FakeAgent):
        from app import agent_factory
        agent = agent_factory.build_agent(mode="chat")

    assert captured.get("api_key") == "override-key"
    assert captured.get("model") == "gpt-4o"
