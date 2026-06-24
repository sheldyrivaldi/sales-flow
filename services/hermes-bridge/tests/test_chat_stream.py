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
    agent.run_conversation.return_value = {
        "final_response": response,
        "messages": [],
    }
    return agent


def test_stream_returns_sse():
    text = "Halo! Ini jawaban panjang dari Hermes. Semoga berguna."
    with patch("app.routes.chat.build_agent", return_value=_fake_agent(text)):
        from app.main import app
        client = TestClient(app)
        r = client.post(
            "/v1/chat/completions",
            json={"model": "default", "messages": [{"role": "user", "content": "test"}], "stream": True},
            headers=HEADERS,
        )
    assert r.status_code == 200
    assert "text/event-stream" in r.headers.get("content-type", "")
    lines = [l for l in r.text.splitlines() if l.startswith("data: ")]
    assert len(lines) >= 2  # ≥1 data chunk + [DONE]
    last = lines[-1]
    assert last == "data: [DONE]"
    # Semua baris kecuali [DONE] harus valid JSON dengan choices.delta.content
    for line in lines[:-1]:
        payload = json.loads(line[len("data: "):])
        assert "choices" in payload
        assert payload["choices"][0]["delta"]["content"]


def test_stream_single_word_text():
    """Teks pendek tanpa kalimat tetap menghasilkan ≥1 chunk + [DONE]."""
    with patch("app.routes.chat.build_agent", return_value=_fake_agent("ok")):
        from app.main import app
        client = TestClient(app)
        r = client.post(
            "/v1/chat/completions",
            json={"model": "default", "messages": [{"role": "user", "content": "test"}], "stream": True},
            headers=HEADERS,
        )
    lines = [l for l in r.text.splitlines() if l.startswith("data: ")]
    assert any(l == "data: [DONE]" for l in lines)
    assert len(lines) >= 2
