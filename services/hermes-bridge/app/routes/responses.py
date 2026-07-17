from __future__ import annotations

import base64
import json
from typing import Any

from fastapi import APIRouter, Depends, HTTPException

from app.agent_factory import build_agent
from app.auth import require_bearer
from app.pdf_render import render_pdf_pages_to_data_urls
from app.schemas import ResponsesRequest

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

    ephemeral_prompt = "\n\n".join(parts)

    agent = build_agent(mode="responses", ephemeral_system_prompt=ephemeral_prompt)

    content: Any
    if req.document_base64:
        content = _build_document_content(req.prompt, req.document_base64, req.document_filename or "document.pdf")
    else:
        content = req.prompt

    text: str = agent.chat(content)

    return {"output_text": text}
