import hmac

from fastapi import Header, HTTPException

from app.config import get_settings


def require_bearer(authorization: str | None = Header(default=None)) -> None:
    settings = get_settings()
    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(
            status_code=401,
            detail={"error": {"code": "unauthorized", "message": "Authorization header wajib berformat 'Bearer <key>'"}},
        )
    token = authorization[len("Bearer "):]
    if not hmac.compare_digest(token, settings.api_server_key):
        raise HTTPException(
            status_code=401,
            detail={"error": {"code": "unauthorized", "message": "API key tidak valid"}},
        )
