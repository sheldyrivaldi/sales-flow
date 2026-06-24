from __future__ import annotations

import json
from typing import Any

from app.config import get_settings

# Lazy import agar unit test bisa mock sebelum import riil.
# Import run_agent di sini supaya test bisa monkeypatch app.agent_factory.AIAgent.
try:
    from run_agent import AIAgent  # type: ignore[import-untyped]
except ModuleNotFoundError:  # pragma: no cover
    AIAgent = None  # type: ignore[assignment,misc]


def build_agent(
    *,
    mode: str,
    ephemeral_system_prompt: str | None = None,
) -> Any:
    """Bangun AIAgent baru per-request.

    mode="chat"      → memory ON, toolsets aktif (untuk percakapan user).
    mode="responses" → memory OFF, toolsets kosong (deterministik untuk GenerateJSON).

    Setiap call menghasilkan instance BARU karena AIAgent tidak thread-safe.
    """
    if AIAgent is None:  # pragma: no cover
        raise RuntimeError("hermes-agent library tidak terinstall — jalankan pip install hermes-agent")

    settings = get_settings()
    active = _get_active_provider()

    kwargs: dict[str, Any] = {
        "quiet_mode": True,
    }

    # Model: config aktif (BR-9) → fallback env.
    if active and active.get("model"):
        kwargs["model"] = active["model"]
    else:
        kwargs["model"] = settings.hermes_model

    # API key & base_url: config aktif → fallback env.
    if active and active.get("api_key"):
        kwargs["api_key"] = active["api_key"]
        if active.get("base_url"):
            kwargs["base_url"] = active["base_url"]
    else:
        # Coba openai dulu, lalu openrouter.
        if settings.openai_api_key:
            kwargs["api_key"] = settings.openai_api_key
        elif settings.openrouter_api_key:
            kwargs["api_key"] = settings.openrouter_api_key
            kwargs["base_url"] = "https://openrouter.ai/api/v1"

    if mode == "chat":
        kwargs["enabled_toolsets"] = settings.enabled_toolsets
        kwargs["disabled_toolsets"] = ["terminal"]
        kwargs["skip_memory"] = False
    elif mode == "responses":
        kwargs["enabled_toolsets"] = []
        kwargs["skip_memory"] = True
        if ephemeral_system_prompt:
            kwargs["ephemeral_system_prompt"] = ephemeral_system_prompt

    return AIAgent(**kwargs)


# --- Provider config in-memory (diisi BR-9) ---

_active_provider: dict[str, Any] | None = None


def _get_active_provider() -> dict[str, Any] | None:
    return _active_provider


def set_active_provider(config: dict[str, Any] | None) -> None:
    global _active_provider
    _active_provider = config


def get_active_provider() -> dict[str, Any] | None:
    return _active_provider
