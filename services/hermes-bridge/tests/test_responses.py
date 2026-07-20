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
    # /v1/responses now uses run_conversation (jalur yang sama dgn /v1/chat)
    # yang mengembalikan dict {"final_response": ...}, bukan agent.chat().
    agent.run_conversation.return_value = {"final_response": response}
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


def test_responses_no_auth():
    from app.main import app
    client = TestClient(app)
    r = client.post(
        "/v1/responses",
        json={"prompt": "test", "response_format": {"type": "json_schema", "json_schema": {"schema": {}}}},
    )
    assert r.status_code == 401


def test_responses_with_document_sends_multimodal_content():
    """When document_base64 is set, agent.chat must receive a list — text
    prompt first, then one image_url part per rendered page — not the plain
    string path (EP-13 vision-based PDF ingest)."""
    captured: dict = {}

    def fake_build(**kwargs):
        agent = MagicMock()

        def fake_run(content, **_):
            captured["content"] = content
            return {"final_response": '{"company_name": "PT Contoh"}'}

        agent.run_conversation.side_effect = fake_run
        return agent

    with patch("app.routes.responses.build_agent", side_effect=fake_build), \
         patch("app.routes.responses.render_pdf_pages_to_data_urls", return_value=["data:image/jpeg;base64,AAA", "data:image/jpeg;base64,BBB"]):
        from app.main import app
        client = TestClient(app)
        r = client.post(
            "/v1/responses",
            json={
                "prompt": "Ekstrak profil perusahaan",
                "response_format": {"type": "json_schema", "json_schema": {"schema": {}}},
                "document_base64": "ZmFrZS1wZGYtYnl0ZXM=",
                "document_filename": "profile.pdf",
            },
            headers=HEADERS,
        )

    assert r.status_code == 200
    assert r.json()["output_text"] == '{"company_name": "PT Contoh"}'

    content = captured["content"]
    assert isinstance(content, list)
    assert content[0] == {"type": "text", "text": "Ekstrak profil perusahaan"}
    assert content[1] == {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,AAA"}}
    assert content[2] == {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,BBB"}}


def test_responses_without_document_sends_plain_string():
    """Backward-compat: no document_base64 → content is still the bare
    prompt string, exactly as before this feature existed."""
    captured: dict = {}

    def fake_build(**kwargs):
        agent = MagicMock()

        def fake_run(content, **_):
            captured["content"] = content
            return {"final_response": '{"ok": true}'}

        agent.run_conversation.side_effect = fake_run
        return agent

    with patch("app.routes.responses.build_agent", side_effect=fake_build):
        from app.main import app
        client = TestClient(app)
        client.post(
            "/v1/responses",
            json={"prompt": "test tanpa dokumen", "response_format": {"type": "json_schema", "json_schema": {"schema": {}}}},
            headers=HEADERS,
        )

    assert captured["content"] == "test tanpa dokumen"


def test_responses_invalid_document_base64_returns_400():
    with patch("app.routes.responses.build_agent", return_value=_fake_agent('{"ok": true}')):
        from app.main import app
        client = TestClient(app)
        r = client.post(
            "/v1/responses",
            json={
                "prompt": "test",
                "response_format": {"type": "json_schema", "json_schema": {"schema": {}}},
                "document_base64": "not-valid-base64!!!",
            },
            headers=HEADERS,
        )
    assert r.status_code == 400
