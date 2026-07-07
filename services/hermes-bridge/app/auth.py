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
    # Compare as bytes: hmac.compare_digest raises TypeError on str inputs
    # containing non-ASCII characters, which would otherwise escape as a 500
    # instead of a clean 401.
    if not hmac.compare_digest(token.encode("utf-8"), settings.api_server_key.encode("utf-8")):
        raise HTTPException(
            status_code=401,
            detail={"error": {"code": "unauthorized", "message": "API key tidak valid"}},
        )
