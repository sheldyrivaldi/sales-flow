from __future__ import annotations

import base64
import json
import logging
from typing import Any

from fastapi import APIRouter, Depends, HTTPException

from app.agent_factory import build_agent, model_chain
from app.auth import require_bearer
from app.pdf_render import render_pdf_pages_to_data_urls
from app.routes.agent_task import _looks_like_provider_error
from app.schemas import ResponsesRequest

log = logging.getLogger("hermes_bridge.responses")

router = APIRouter(dependencies=[Depends(require_bearer)])


def _image_mime(data: bytes) -> str | None:
    """Detects a directly-embeddable image by magic bytes. Images can't be
    opened by pdfium (a PDF renderer), so they must be sent to the model as-is
    rather than routed through render_pdf_pages_to_data_urls."""
    if data[:3] == b"\xff\xd8\xff":
        return "image/jpeg"
    if data[:8] == b"\x89PNG\r\n\x1a\n":
        return "image/png"
    if data[:6] in (b"GIF87a", b"GIF89a"):
        return "image/gif"
    if data[:4] == b"RIFF" and data[8:12] == b"WEBP":
        return "image/webp"
    return None


def _doc_to_image_urls(document_base64: str, filename: str) -> list[str]:
    """Turns one attachment into image data-URLs the model can read: images
    pass through directly, PDFs are rendered page-by-page. NEVER raises — an
    attachment that can't be decoded/rendered is skipped (returns []) so a
    single bad file (mis. gambar rusak, PDF terenkripsi) doesn't sink the whole
    request; the caller falls back to prompt-only when nothing renders."""
    try:
        data = base64.b64decode(document_base64, validate=True)
    except Exception as exc:
        log.warning("responses: lampiran %s gagal decode base64: %s", filename, exc)
        return []
    mime = _image_mime(data)
    if mime is not None:
        b64 = base64.b64encode(data).decode("ascii")
        return [f"data:{mime};base64,{b64}"]
    try:
        return render_pdf_pages_to_data_urls(data)
    except Exception as exc:
        # HTTPException (PDF tak bisa dibuka) atau error render lain — lewati.
        log.warning("responses: lampiran %s gagal dirender (dilewati): %s", filename, exc)
        return []


def _build_document_content(prompt: str, docs: list[tuple[str, str]]) -> list[dict[str, Any]] | None:
    """Builds an OpenAI-style multimodal user message: the text prompt followed
    by one image part per rendered page/image, across ALL attached documents.
    hermes-agent's conversation loop passes list-shaped content straight through
    as the turn's message content, so this works with `agent.chat()` exactly
    like a plain string prompt does.

    Returns None when NO attachment produced any image (all skipped) so the
    caller can send prompt-only text instead of an image-less list."""
    image_parts: list[dict[str, Any]] = []
    for document_base64, filename in docs:
        for url in _doc_to_image_urls(document_base64, filename):
            image_parts.append({"type": "image_url", "image_url": {"url": url}})
    if not image_parts:
        return None
    return [{"type": "text", "text": prompt}, *image_parts]


@router.post("/v1/responses")
def responses(req: ResponsesRequest) -> dict[str, Any]:
    schema = req.response_format.json_schema.get("schema", {})
    schema_text = json.dumps(schema, ensure_ascii=False, indent=2) if schema else ""

    parts = ["Balas HANYA JSON valid sesuai schema berikut, tanpa penjelasan, tanpa markdown, tanpa code-fence."]
    if schema_text:
        parts.append(f"Schema:\n{schema_text}")

    system_prompt = "\n\n".join(parts)

    # Kumpulkan semua lampiran: bentuk jamak `documents` (utama) + bentuk
    # tunggal lama `document_base64` demi kompatibilitas.
    docs: list[tuple[str, str]] = [(d.base64, d.filename) for d in req.documents if d.base64]
    if req.document_base64:
        docs.append((req.document_base64, req.document_filename or "document.pdf"))

    content: Any = req.prompt
    if docs:
        built = _build_document_content(req.prompt, docs)
        if built is not None:
            content = built
        # built is None → semua lampiran gagal dibaca; lanjut prompt-only
        # (melampirkan berkas tak boleh membuat generasi lebih buruk).

    # Gunakan run_conversation (jalur yang SAMA dengan /v1/chat) alih-alih
    # agent.chat(): pada generasi panjang, agent.chat() non-stream rentan
    # "broken pipe" ke provider dan MENGEMBALIKAN string error sebagai hasil
    # ("API call failed after 3 retries...") — bukan JSON — sehingga sisi Go
    # gagal parse. run_conversation lebih tahan panggilan lama dan bila semua
    # retry provider gagal, final_response kosong → kita balas 502 (bukan
    # 200 berisi teks error yang menyamar sebagai output).
    #
    # Bila model utama sedang ditolak diam-diam oleh backend, coba model
    # cadangan berikutnya alih-alih langsung menyerah.
    text = ""
    for model in model_chain():
        agent = build_agent(mode="responses", model_override=model)
        try:
            result = agent.run_conversation(
                content,
                system_message=system_prompt,
                conversation_history=[],
                task_id="",
            )
        except Exception as exc:
            log.warning("responses: model %s gagal: %s", model, exc)
            continue
        candidate = (result.get("final_response") or "").strip()
        if candidate and not _looks_like_provider_error(candidate):
            text = candidate
            break
        log.warning("responses: model %s tidak menghasilkan output valid", model)

    if not text:
        raise HTTPException(
            status_code=502,
            detail="Agent AI gagal menghasilkan output — cek log hermes-bridge untuk detail provider.",
        )

    return {"output_text": text}
