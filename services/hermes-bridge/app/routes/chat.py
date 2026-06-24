from __future__ import annotations

import json
from typing import Any

from fastapi import APIRouter, Depends, Header
from fastapi.responses import StreamingResponse

from app.agent_factory import build_agent
from app.auth import require_bearer
from app.schemas import ChatCompletionRequest

router = APIRouter(dependencies=[Depends(require_bearer)])


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

    history = [{"role": m.role, "content": m.content} for m in req.messages[:-1]]
    user_message = req.messages[-1].content
    task_id = x_hermes_session_id or ""

    agent = build_agent(mode="chat")
    result = agent.run_conversation(
        user_message,
        conversation_history=history,
        task_id=task_id,
    )
    final_response: str = result.get("final_response", "")

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
