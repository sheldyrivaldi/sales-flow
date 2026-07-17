from __future__ import annotations

import base64
import json
from typing import Any

from fastapi import APIRouter, Depends, Header, HTTPException
from fastapi.responses import StreamingResponse

from app.agent_factory import build_agent
from app.auth import require_bearer
from app.pdf_render import render_pdf_pages_to_data_urls
from app.schemas import ChatCompletionRequest

router = APIRouter(dependencies=[Depends(require_bearer)])

# Ekstensi gambar yang dikirim langsung sebagai satu image part (tanpa render).
_IMAGE_MIME = {
    ".png": "image/png",
    ".jpg": "image/jpeg",
    ".jpeg": "image/jpeg",
    ".webp": "image/webp",
    ".gif": "image/gif",
}


def _attachment_image_parts(document_base64: str, filename: str) -> list[dict[str, Any]]:
    """Ubah lampiran (PDF/gambar) menjadi daftar image part OpenAI-style.

    PDF dirender per halaman via pdfium (pipeline yang sama dengan
    /v1/responses); gambar diteruskan apa adanya sebagai data URL.
    """
    lower = (filename or "").lower()
    for ext, mime in _IMAGE_MIME.items():
        if lower.endswith(ext):
            return [{"type": "image_url", "image_url": {"url": f"data:{mime};base64,{document_base64}"}}]

    try:
        pdf_bytes = base64.b64decode(document_base64, validate=True)
    except Exception as exc:
        raise HTTPException(status_code=400, detail=f"document_base64 tidak valid: {exc}") from exc

    urls = render_pdf_pages_to_data_urls(pdf_bytes)
    return [{"type": "image_url", "image_url": {"url": u}} for u in urls]


def _split_chunks(text: str) -> list[str]:
    """Pecah teks jadi potongan per kalimat agar stream terasa mengalir."""
    import re
    sentences = re.split(r"(?<=[.!?])\s+", text.strip())
    result = []
    for s in sentences:
        if s:
            result.append(s + " ")
    if not result:
        result = [text]
    return result


def _chat_stream_generator(text: str):
    for chunk in _split_chunks(text):
        data = {"choices": [{"delta": {"content": chunk}}]}
        yield f"data: {json.dumps(data, ensure_ascii=False)}\n\n"
    yield "data: [DONE]\n\n"


@router.post("/v1/chat/completions")
def chat_completions(
    req: ChatCompletionRequest,
    x_hermes_session_id: str | None = Header(default=None, alias="X-Hermes-Session-Id"),
) -> Any:
    # Pisah history + user_message.
    if not req.messages:
        from fastapi import HTTPException
        raise HTTPException(status_code=422, detail="messages tidak boleh kosong")

    # Pesan system pertama (guardrail dari Go) diteruskan sebagai
    # system_message run_conversation — bukan sebagai history biasa — supaya
    # benar-benar berlaku sebagai instruksi sistem di setiap giliran.
    messages = list(req.messages)
    system_message: str | None = None
    if messages and messages[0].role == "system":
        system_message = messages[0].content
        messages = messages[1:]

    if not messages:
        from fastapi import HTTPException as _HTTPException
        raise _HTTPException(status_code=422, detail="messages tidak boleh kosong setelah system message")

    history = [{"role": m.role, "content": m.content} for m in messages[:-1]]
    user_message: Any = messages[-1].content
    task_id = x_hermes_session_id or ""

    # Lampiran dokumen → pesan user terakhir menjadi konten multimodal
    # (teks + image parts). hermes-agent meneruskan konten berbentuk list ke
    # provider vision-capable apa adanya (lihat penanganan image_url/
    # input_image di run_agent.py).
    if req.document_base64:
        parts: list[dict[str, Any]] = [{"type": "text", "text": user_message}]
        parts.extend(_attachment_image_parts(req.document_base64, req.document_filename or "document.pdf"))
        user_message = parts

    agent = build_agent(mode="chat")
    result = agent.run_conversation(
        user_message,
        system_message=system_message,
        conversation_history=history,
        task_id=task_id,
    )
    # result.get(..., "") tidak cukup: kunci "final_response" bisa ADA dengan
    # nilai None (mis. semua retry provider gagal), bukan cuma hilang — .get()
    # hanya memakai default saat kunci benar-benar tidak ada.
    final_response: str = result.get("final_response") or ""

    if not final_response:
        from fastapi import HTTPException
        raise HTTPException(
            status_code=502,
            detail="Hermes agent gagal menghasilkan respons — cek log hermes-bridge untuk detail provider.",
        )

    if req.stream:
        return StreamingResponse(
            _chat_stream_generator(final_response),
            media_type="text/event-stream",
            headers={
                "Cache-Control": "no-cache",
                "X-Accel-Buffering": "no",
            },
        )

    return {
        "choices": [
            {
                "message": {
                    "content": final_response,
                    "tool_calls": [],
                }
            }
        ]
    }
