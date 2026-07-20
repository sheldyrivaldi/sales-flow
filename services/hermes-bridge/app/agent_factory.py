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


def primary_model() -> str:
    """Model utama: config aktif (BR-9) → fallback env."""
    active = _get_active_provider()
    if active and active.get("model"):
        return str(active["model"])
    return get_settings().hermes_model


def model_chain() -> list[str]:
    """Urutan model yang dicoba: utama lalu cadangan (tanpa duplikat).

    Backend Codex kadang MENOLAK DIAM-DIAM model tertentu — request
    menggantung 90 detik lalu broken pipe, tanpa error yang bisa dibedakan.
    Mengulang model yang sama tidak menolong, jadi percobaan berikutnya harus
    memakai model lain.
    """
    chain = [primary_model()]
    for m in get_settings().hermes_model_fallbacks:
        if m not in chain:
            chain.append(m)
    return chain


def build_agent(
    *,
    mode: str,
    ephemeral_system_prompt: str | None = None,
    model_override: str | None = None,
) -> Any:
    """Bangun AIAgent baru per-request.

    mode="chat"      → memory ON, toolsets aktif (untuk percakapan user).
    mode="responses" → memory OFF, toolsets kosong (deterministik untuk GenerateJSON).

    model_override memaksa model tertentu (dipakai saat mencoba cadangan).

    Setiap call menghasilkan instance BARU karena AIAgent tidak thread-safe.
    """
    if AIAgent is None:  # pragma: no cover
        raise RuntimeError("hermes-agent library tidak terinstall — jalankan pip install hermes-agent")

    settings = get_settings()
    active = _get_active_provider()

    kwargs: dict[str, Any] = {
        "quiet_mode": True,
    }

    kwargs["model"] = model_override or primary_model()

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
        # Active config's enabled_toolsets (EP-18 ST-18.4, AI Provider Config
        # UI) overrides the ENABLED_TOOLSETS env default when explicitly
        # set — None means "not configured", so the env default still
        # applies; an explicit (possibly empty) list always wins.
        if active and active.get("enabled_toolsets") is not None:
            kwargs["enabled_toolsets"] = active["enabled_toolsets"]
        else:
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
