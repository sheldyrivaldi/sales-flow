import os
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


def _fake_agent(response: str = "Halo dari Hermes!"):
    agent = MagicMock()
    agent.run_conversation.return_value = {
        "final_response": response,
        "messages": [],
    }
    return agent


HEADERS = {"Authorization": "Bearer test-key"}


def test_chat_nonstream_success():
    with patch("app.routes.chat.build_agent", return_value=_fake_agent("halo")):
        from app.main import app
        client = TestClient(app)
        r = client.post(
            "/v1/chat/completions",
            json={"model": "default", "messages": [{"role": "user", "content": "halo"}], "stream": False},
            headers=HEADERS,
        )
    assert r.status_code == 200
    body = r.json()
    assert body["choices"][0]["message"]["content"] == "halo"


def test_chat_nonstream_passes_history():
    mock_agent = _fake_agent("ok")
    with patch("app.routes.chat.build_agent", return_value=mock_agent):
        from app.main import app
        client = TestClient(app)
        client.post(
            "/v1/chat/completions",
            json={
                "model": "default",
                "messages": [
                    {"role": "user", "content": "pertanyaan pertama"},
                    {"role": "assistant", "content": "jawaban pertama"},
                    {"role": "user", "content": "pertanyaan kedua"},
                ],
                "stream": False,
            },
            headers=HEADERS,
        )
    call_kwargs = mock_agent.run_conversation.call_args
    assert call_kwargs[0][0] == "pertanyaan kedua"
    assert len(call_kwargs[1]["conversation_history"]) == 2


def test_chat_no_auth():
    from app.main import app
    client = TestClient(app)
    r = client.post(
        "/v1/chat/completions",
        json={"model": "default", "messages": [{"role": "user", "content": "test"}]},
    )
    assert r.status_code == 401


def test_chat_empty_messages():
    from app.main import app
    client = TestClient(app)
    r = client.post(
        "/v1/chat/completions",
        json={"model": "default", "messages": []},
        headers=HEADERS,
    )
    assert r.status_code == 422


def test_chat_with_document_builds_multimodal_user_message():
    """document_base64 → pesan user terakhir menjadi list [text, image parts];
    history tetap string biasa."""
    captured: dict = {}

    def fake_build(**kwargs):
        agent = MagicMock()

        def fake_run(user_message, **run_kwargs):
            captured["user_message"] = user_message
            captured["history"] = run_kwargs.get("conversation_history")
            return {"final_response": "Isi dokumen sudah saya baca.", "messages": []}

        agent.run_conversation.side_effect = fake_run
        return agent

    with patch("app.routes.chat.build_agent", side_effect=fake_build), \
         patch("app.routes.chat.render_pdf_pages_to_data_urls", return_value=["data:image/jpeg;base64,AAA"]):
        from app.main import app
        client = TestClient(app)
        r = client.post(
            "/v1/chat/completions",
            json={
                "model": "default",
                "messages": [
                    {"role": "user", "content": "halo"},
                    {"role": "assistant", "content": "hai"},
                    {"role": "user", "content": "baca dokumen ini"},
                ],
                "stream": False,
                "document_base64": "ZmFrZS1wZGY=",
                "document_filename": "penawaran.pdf",
            },
            headers=HEADERS,
        )

    assert r.status_code == 200
    msg = captured["user_message"]
    assert isinstance(msg, list)
    assert msg[0] == {"type": "text", "text": "baca dokumen ini"}
    assert msg[1] == {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,AAA"}}
    assert captured["history"] == [
        {"role": "user", "content": "halo"},
        {"role": "assistant", "content": "hai"},
    ]


def test_chat_with_image_attachment_passes_through_without_render():
    captured: dict = {}

    def fake_build(**kwargs):
        agent = MagicMock()

        def fake_run(user_message, **run_kwargs):
            captured["user_message"] = user_message
            return {"final_response": "Gambar diterima.", "messages": []}

        agent.run_conversation.side_effect = fake_run
        return agent

    with patch("app.routes.chat.build_agent", side_effect=fake_build):
        from app.main import app
        client = TestClient(app)
        r = client.post(
            "/v1/chat/completions",
            json={
                "model": "default",
                "messages": [{"role": "user", "content": "lihat screenshot ini"}],
                "stream": False,
                "document_base64": "aW1hZ2UtYnl0ZXM=",
                "document_filename": "layar.png",
            },
            headers=HEADERS,
        )

    assert r.status_code == 200
    msg = captured["user_message"]
    assert isinstance(msg, list)
    assert msg[1]["image_url"]["url"].startswith("data:image/png;base64,")
