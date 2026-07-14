from unittest.mock import patch

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


def test_reset_memory_no_auth():
    from app.main import app
    client = TestClient(app)
    r = client.post("/admin/reset-memory")
    assert r.status_code == 401


def test_reset_memory_library_not_installed_returns_friendly_error():
    """In this environment hermes-agent (and hermes_constants within it) is
    genuinely not installed — app.routes.admin.get_hermes_home is None at
    import time, exactly like app.agent_factory.AIAgent — so this exercises
    the real degrade path, not a mocked one."""
    from app.main import app
    client = TestClient(app)
    r = client.post("/admin/reset-memory", headers=HEADERS)
    assert r.status_code == 502
    assert r.json()["detail"]["error"]["code"] == "provider_error"


def test_reset_memory_deletes_existing_files(tmp_path):
    memories_dir = tmp_path / "memories"
    memories_dir.mkdir()
    (memories_dir / "MEMORY.md").write_text("some agent notes")
    (memories_dir / "USER.md").write_text("some user profile")
    (tmp_path / "memory_store.db").write_bytes(b"sqlite-fake-content")
    (tmp_path / "memory_store.db-wal").write_bytes(b"wal")

    from app.main import app
    client = TestClient(app)

    with patch("app.routes.admin.get_hermes_home", return_value=tmp_path):
        r = client.post("/admin/reset-memory", headers=HEADERS)

    assert r.status_code == 200
    body = r.json()
    assert body["status"] == "ok"
    assert len(body["deleted"]) == 4

    assert not (memories_dir / "MEMORY.md").exists()
    assert not (memories_dir / "USER.md").exists()
    assert not (tmp_path / "memory_store.db").exists()
    assert not (tmp_path / "memory_store.db-wal").exists()


def test_reset_memory_no_files_present_still_succeeds(tmp_path):
    from app.main import app
    client = TestClient(app)

    with patch("app.routes.admin.get_hermes_home", return_value=tmp_path):
        r = client.post("/admin/reset-memory", headers=HEADERS)

    assert r.status_code == 200
    body = r.json()
    assert body["status"] == "ok"
    assert body["deleted"] == []
