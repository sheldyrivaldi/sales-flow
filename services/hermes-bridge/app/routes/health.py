from __future__ import annotations

import importlib.metadata

from fastapi import APIRouter

from app.config import get_settings

router = APIRouter()


@router.get("/v1/capabilities")
def capabilities() -> dict:
    settings = get_settings()
    try:
        version = importlib.metadata.version("hermes-agent")
    except importlib.metadata.PackageNotFoundError:
        version = "dev"

    return {
        "version": version,
        "models": [settings.hermes_model],
        "features": ["chat", "memory", "tools"],
    }
