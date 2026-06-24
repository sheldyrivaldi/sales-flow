from __future__ import annotations

import json
from typing import Any

from fastapi import APIRouter, Depends

from app.agent_factory import build_agent
from app.auth import require_bearer
from app.schemas import ResponsesRequest

router = APIRouter(dependencies=[Depends(require_bearer)])


@router.post("/v1/responses")
def responses(req: ResponsesRequest) -> dict[str, Any]:
    schema = req.response_format.json_schema.get("schema", {})
    schema_text = json.dumps(schema, ensure_ascii=False, indent=2) if schema else ""

    parts = ["Balas HANYA JSON valid sesuai schema berikut, tanpa penjelasan, tanpa markdown, tanpa code-fence."]
    if schema_text:
        parts.append(f"Schema:\n{schema_text}")

    ephemeral_prompt = "\n\n".join(parts)

    agent = build_agent(mode="responses", ephemeral_system_prompt=ephemeral_prompt)
    text: str = agent.chat(req.prompt)

    return {"output_text": text}
