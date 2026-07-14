from __future__ import annotations

from urllib.parse import urlparse

from fastapi import APIRouter, Depends, HTTPException

from app.agent_factory import get_active_provider, set_active_provider
from app.auth import require_bearer
from app.schemas import ProviderConfigRequest

# Lazy/defensive import mirrors app.agent_factory's AIAgent import: the
# hermes-agent library (and hermes_constants within it) may not be installed
# in every environment this bridge runs test suites in. get_hermes_home is
# None (not raised at import time) so /admin/reset-memory can return a
# friendly error instead of crashing module import for the whole app.
try:
    from hermes_constants import get_hermes_home  # type: ignore[import-untyped]
except ModuleNotFoundError:  # pragma: no cover
    get_hermes_home = None  # type: ignore[assignment]

router = APIRouter(dependencies=[Depends(require_bearer)])

_PROVIDER_DEFAULTS: dict[str, str] = {
    "openai": "https://api.openai.com/v1",
    "openrouter": "https://openrouter.ai/api/v1",
}

# Host suffix each provider's base_url is allowed to point at. Prevents a
# caller with the (shared) admin bearer key from redirecting agent traffic —
# including full conversation content — to an arbitrary or internal host.
_ALLOWED_HOST_SUFFIXES: dict[str, str] = {
    "openai": "openai.com",
    "openrouter": "openrouter.ai",
}


def _validate_base_url(provider: str, base_url: str) -> None:
    parsed = urlparse(base_url)
    host = parsed.hostname or ""
    suffix = _ALLOWED_HOST_SUFFIXES[provider]
    if parsed.scheme != "https" or not (host == suffix or host.endswith("." + suffix)):
        raise HTTPException(
            status_code=422,
            detail={
                "error": {
                    "code": "invalid_base_url",
                    "message": f"base_url harus https dan berada di domain {suffix}",
                }
            },
        )


@router.post("/admin/config")
def set_config(req: ProviderConfigRequest) -> dict:
    if req.provider not in _PROVIDER_DEFAULTS:
        raise HTTPException(
            status_code=422,
            detail={"error": {"code": "invalid_provider", "message": f"provider harus salah satu dari: {list(_PROVIDER_DEFAULTS)}"}},
        )

    if req.base_url:
        _validate_base_url(req.provider, req.base_url)
    base_url = req.base_url or _PROVIDER_DEFAULTS[req.provider]

    set_active_provider({
        "provider": req.provider,
        "model": req.model,
        "base_url": base_url,
        "api_key": req.api_key,
        "enabled_toolsets": req.enabled_toolsets,
    })

    return {"status": "ok", "provider": req.provider, "model": req.model}


@router.get("/admin/config")
def get_config() -> dict:
    active = get_active_provider()
    if not active:
        return {"active": None}
    return {
        "active": {
            "provider": active.get("provider"),
            "model": active.get("model"),
            "base_url": active.get("base_url"),
            "enabled_toolsets": active.get("enabled_toolsets"),
        }
    }


@router.post("/admin/reset-memory")
def reset_memory() -> dict:
    """Clears Hermes workspace memory (EP-16 TK-16.3.1).

    Investigated directly against the pinned hermes-agent source
    (v2026.6.19, see pyproject.toml): there is no in-process "clear memory"
    method on AIAgent itself — `hermes memory reset --yes --target all`
    (hermes_cli/main.py:cmd_memory) simply deletes
    get_hermes_home()/memories/{MEMORY.md,USER.md} on disk, so that's
    replicated here in-process rather than shelling out to the CLI. This
    workspace's configured memory.provider (deploy/hermes/config.yaml.example)
    is "holographic" — a SQLite fact store at get_hermes_home()/
    memory_store.db (plugins/memory/holographic/store.py) with no explicit
    "clear all" API either; deleting the file is that store's own bootstrap
    path (recreated empty on next use), so it's deleted too, along with any
    SQLite WAL/SHM sidecar files.

    Known gap: if plugins.hermes-memory-store.db_path is ever overridden
    away from the default in config.yaml, this hardcoded path would miss it.
    Not verified end-to-end against a live AIAgent process in this
    environment (hermes-agent isn't installed here — see TK-16.4.2).
    """
    if get_hermes_home is None:  # pragma: no cover
        raise HTTPException(
            status_code=502,
            detail={"error": {"code": "provider_error", "message": "hermes-agent library tidak terinstall — jalankan pip install hermes-agent"}},
        )

    home = get_hermes_home()
    deleted: list[str] = []

    memories_dir = home / "memories"
    for filename in ("MEMORY.md", "USER.md"):
        path = memories_dir / filename
        if path.exists():
            path.unlink()
            deleted.append(str(path))

    for suffix in ("", "-wal", "-shm"):
        db_path = home / f"memory_store.db{suffix}"
        if db_path.exists():
            db_path.unlink()
            deleted.append(str(db_path))

    return {"status": "ok", "deleted": deleted}
