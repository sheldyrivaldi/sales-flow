from __future__ import annotations

from typing import Any

from pydantic import BaseModel, Field


class ToolCallFunction(BaseModel):
    name: str
    arguments: Any = None


class ToolCall(BaseModel):
    id: str = ""
    type: str = "function"
    function: ToolCallFunction = Field(default_factory=lambda: ToolCallFunction(name=""))


class ChatMessage(BaseModel):
    role: str
    content: str = ""
    tool_calls: list[ToolCall] | None = None


class ChatCompletionRequest(BaseModel):
    model: str = "default"
    messages: list[ChatMessage]
    stream: bool = False


class ResponseFormat(BaseModel):
    type: str = "json_schema"
    json_schema: dict[str, Any] = Field(default_factory=dict)


class ResponsesRequest(BaseModel):
    prompt: str
    response_format: ResponseFormat = Field(default_factory=ResponseFormat)


class ProviderConfigRequest(BaseModel):
    provider: str
    model: str
    base_url: str | None = None
    api_key: str
    # None = don't override (build_agent keeps the ENABLED_TOOLSETS env
    # default for chat mode); an explicit list (possibly empty) replaces it.
    enabled_toolsets: list[str] | None = None
