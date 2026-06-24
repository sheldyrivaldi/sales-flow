import os
from unittest.mock import MagicMock, patch

import pytest

from app.config import reset_settings


@pytest.fixture(autouse=True)
def set_env(monkeypatch):
    monkeypatch.setenv("API_SERVER_KEY", "test-key")
    monkeypatch.setenv("HERMES_MODEL", "anthropic/claude-sonnet-4-6")
    monkeypatch.setenv("ENABLED_TOOLSETS", "web,search")
    monkeypatch.setenv("OPENAI_API_KEY", "sk-test")
    reset_settings()
    yield
    reset_settings()


def _make_mock_agent_class():
    instances: list[MagicMock] = []

    class FakeAIAgent:
        def __init__(self, **kwargs):
            self._kwargs = kwargs
            instances.append(self)

        def chat(self, msg):
            return "ok"

        def run_conversation(self, user_message, conversation_history=None, task_id=None):
            return {"final_response": "ok", "messages": []}

    FakeAIAgent.instances = instances
    return FakeAIAgent


def test_build_agent_chat_mode():
    FakeAIAgent = _make_mock_agent_class()
    with patch("app.agent_factory.AIAgent", FakeAIAgent):
        from app import agent_factory
        agent_factory.set_active_provider(None)
        agent = agent_factory.build_agent(mode="chat")

    assert agent._kwargs["skip_memory"] is False
    assert "web" in agent._kwargs["enabled_toolsets"]
    assert "terminal" in agent._kwargs["disabled_toolsets"]
    assert agent._kwargs["model"] == "anthropic/claude-sonnet-4-6"


def test_build_agent_responses_mode():
    FakeAIAgent = _make_mock_agent_class()
    with patch("app.agent_factory.AIAgent", FakeAIAgent):
        from app import agent_factory
        agent_factory.set_active_provider(None)
        agent = agent_factory.build_agent(mode="responses", ephemeral_system_prompt="Balas JSON saja.")

    assert agent._kwargs["skip_memory"] is True
    assert agent._kwargs["enabled_toolsets"] == []
    assert agent._kwargs.get("ephemeral_system_prompt") == "Balas JSON saja."


def test_build_agent_active_provider_override():
    FakeAIAgent = _make_mock_agent_class()
    with patch("app.agent_factory.AIAgent", FakeAIAgent):
        from app import agent_factory
        agent_factory.set_active_provider(
            {"provider": "openrouter", "model": "gpt-4o", "base_url": "https://openrouter.ai/api/v1", "api_key": "or-key"}
        )
        agent = agent_factory.build_agent(mode="chat")
        agent_factory.set_active_provider(None)

    assert agent._kwargs["model"] == "gpt-4o"
    assert agent._kwargs["api_key"] == "or-key"
    assert agent._kwargs["base_url"] == "https://openrouter.ai/api/v1"
