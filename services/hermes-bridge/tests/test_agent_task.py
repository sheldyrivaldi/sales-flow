"""Ketahanan parsing output agent-task.

Payload playbook sekarang memuat "deck" (belasan KB), sehingga satu kurung
penutup yang meleset tidak boleh menggagalkan seluruh job. Dua lapis
pengaman diuji di sini: perbaikan kurung, dan retry saat output tak-terparse.
"""

import json
from unittest.mock import MagicMock, patch

import pytest

from app.routes.agent_task import _extract_json, _repair_brackets, _run_agent_task
from app.schemas import AgentTaskRequest


def test_extract_json_plain():
    assert _extract_json('{"a": 1}') == {"a": 1}


def test_extract_json_strips_code_fence():
    assert _extract_json('```json\n{"a": 1}\n```') == {"a": 1}


def test_repair_missing_array_closer():
    """Kasus nyata dari model: `items` ditutup '}' tanpa ']' lebih dulu."""
    broken = '{"columns":[{"title":"A","items":["x","y"}]}'
    with pytest.raises(json.JSONDecodeError):
        json.loads(broken)
    obj = _extract_json(broken)
    assert obj == {"columns": [{"title": "A", "items": ["x", "y"]}]}


def test_repair_ignores_brackets_inside_strings():
    """Kurung di dalam string tidak boleh dianggap struktur."""
    src = '{"note":"pakai [a] dan {b} di teks","ok":true}'
    assert _repair_brackets(src) == src
    assert _extract_json(src)["note"] == "pakai [a] dan {b} di teks"


def test_repair_closes_truncated_tail():
    obj = _extract_json('{"deck":[{"layout":"cover","heading":"X"')
    assert obj == {"deck": [{"layout": "cover", "heading": "X"}]}


def test_valid_json_is_left_untouched():
    src = '{"a":[1,2],"b":{"c":3}}'
    assert _repair_brackets(src) == src


def _req() -> AgentTaskRequest:
    return AgentTaskRequest(
        instruction="buat playbook",
        job_id="job-1",
        callback_url="http://app/internal/playbook-jobs/job-1/complete",
        callback_secret="s3cret",
    )


@patch("app.routes.agent_task._post_callback")
@patch("app.routes.agent_task.build_agent")
def test_retries_when_output_is_not_json(build_agent, post_callback):
    """Output tak-terparse harus memicu percobaan ulang, bukan langsung gagal."""
    agent = MagicMock()
    agent.run_conversation.side_effect = [
        {"final_response": "maaf, ini bukan JSON sama sekali"},
        {"final_response": '{"title":"Playbook","deck":[]}'},
    ]
    build_agent.return_value = agent

    _run_agent_task(_req())

    assert agent.run_conversation.call_count == 2
    payload = post_callback.call_args[0][2]
    assert payload["content"]["title"] == "Playbook"
    assert "error" not in payload


@patch("app.routes.agent_task._post_callback")
@patch("app.routes.agent_task.build_agent")
def test_reports_error_after_all_attempts_fail(build_agent, post_callback):
    agent = MagicMock()
    agent.run_conversation.return_value = {"final_response": "bukan json"}
    build_agent.return_value = agent

    _run_agent_task(_req())

    assert agent.run_conversation.call_count == 3
    payload = post_callback.call_args[0][2]
    assert "error" in payload


@patch("app.routes.agent_task.model_chain", return_value=["mati", "hidup"])
@patch("app.routes.agent_task._post_callback")
@patch("app.routes.agent_task.build_agent")
def test_falls_back_to_next_model_when_provider_rejects(build_agent, post_callback, _chain):
    """Backend menolak diam-diam model utama → percobaan kedua HARUS memakai
    model lain, bukan mengulang model yang sama."""
    agent = MagicMock()
    agent.run_conversation.side_effect = [
        {"final_response": "API call failed after 3 retries: Connection error."},
        {"final_response": '{"title":"Playbook","deck":[]}'},
    ]
    build_agent.return_value = agent

    _run_agent_task(_req())

    models_tried = [c.kwargs["model_override"] for c in build_agent.call_args_list]
    assert models_tried[:2] == ["mati", "hidup"]
    payload = post_callback.call_args[0][2]
    assert payload["content"]["title"] == "Playbook"


def test_model_chain_dedupes_and_orders(monkeypatch):
    from app import agent_factory
    from app.config import reset_settings

    monkeypatch.setenv("API_SERVER_KEY", "k")
    monkeypatch.setenv("HERMES_MODEL", "utama")
    monkeypatch.setenv("HERMES_MODEL_FALLBACKS", "cadangan, utama ,lain")
    reset_settings()
    agent_factory.set_active_provider(None)
    try:
        assert agent_factory.model_chain() == ["utama", "cadangan", "lain"]
    finally:
        reset_settings()
