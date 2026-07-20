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


def _build_document_content(prompt: str, document_base64: str, filename: str) -> list[dict[str, Any]]:
    """Builds an OpenAI-style multimodal user message: the text prompt
    followed by one image part per rendered PDF page. hermes-agent's
    conversation loop passes list-shaped content straight through as the
    turn's message content (confirmed via its image_url/input_image
    handling elsewhere in run_agent.py), so this works with `agent.chat()`
    exactly like a plain string prompt does."""
    try:
        pdf_bytes = base64.b64decode(document_base64, validate=True)
    except Exception as exc:
        raise HTTPException(status_code=400, detail=f"document_base64 tidak valid: {exc}") from exc

    page_urls = render_pdf_pages_to_data_urls(pdf_bytes)

    parts: list[dict[str, Any]] = [{"type": "text", "text": prompt}]
    for url in page_urls:
        parts.append({"type": "image_url", "image_url": {"url": url}})
    return parts


@router.post("/v1/responses")
def responses(req: ResponsesRequest) -> dict[str, Any]:
    schema = req.response_format.json_schema.get("schema", {})
    schema_text = json.dumps(schema, ensure_ascii=False, indent=2) if schema else ""

    parts = ["Balas HANYA JSON valid sesuai schema berikut, tanpa penjelasan, tanpa markdown, tanpa code-fence."]
    if schema_text:
        parts.append(f"Schema:\n{schema_text}")

    system_prompt = "\n\n".join(parts)

    content: Any
    if req.document_base64:
        content = _build_document_content(req.prompt, req.document_base64, req.document_filename or "document.pdf")
    else:
        content = req.prompt

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
