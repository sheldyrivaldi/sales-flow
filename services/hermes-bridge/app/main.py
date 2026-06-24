import logging
import os

import uvicorn
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from app.routes import admin, chat, health, responses

log = logging.getLogger("hermes-bridge")

app = FastAPI(title="hermes-bridge", version="0.1.0")


@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception) -> JSONResponse:
    log.error("Unhandled error: %s", exc, exc_info=True)
    return JSONResponse(
        status_code=502,
        content={"error": {"code": "provider_error", "message": "Terjadi kesalahan pada AI provider. Coba lagi nanti."}},
    )


@app.get("/health")
def health_check() -> JSONResponse:
    return JSONResponse({"status": "ok"})


app.include_router(chat.router)
app.include_router(health.router)
app.include_router(responses.router)
app.include_router(admin.router)


if __name__ == "__main__":
    port = int(os.getenv("PORT", "8642"))
    uvicorn.run("app.main:app", host="0.0.0.0", port=port, reload=False)
