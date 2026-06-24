from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException

from app.agent_factory import get_active_provider, set_active_provider
from app.auth import require_bearer
from app.schemas import ProviderConfigRequest

router = APIRouter(dependencies=[Depends(require_bearer)])

_PROVIDER_DEFAULTS: dict[str, str] = {
    "openai": "https://api.openai.com/v1",
    "openrouter": "https://openrouter.ai/api/v1",
}


@router.post("/admin/config")
def set_config(req: ProviderConfigRequest) -> dict:
    if req.provider not in _PROVIDER_DEFAULTS:
        raise HTTPException(
            status_code=422,
            detail={"error": {"code": "invalid_provider", "message": f"provider harus salah satu dari: {list(_PROVIDER_DEFAULTS)}"}},
        )

    base_url = req.base_url or _PROVIDER_DEFAULTS[req.provider]

    set_active_provider({
        "provider": req.provider,
        "model": req.model,
        "base_url": base_url,
        "api_key": req.api_key,
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
        }
    }
