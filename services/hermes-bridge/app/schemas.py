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
    # Optional document attachment for the LAST user message: PDFs are
    # rendered to per-page images and sent as native multimodal vision
    # content; images pass through directly (same pipeline as /v1/responses).
    document_base64: str | None = None
    document_filename: str | None = None


class ResponseFormat(BaseModel):
    type: str = "json_schema"
    json_schema: dict[str, Any] = Field(default_factory=dict)


class DocumentPayload(BaseModel):
    """Satu lampiran (PDF/gambar). Dipakai agar SATU permintaan bisa membawa
    BANYAK dokumen sekaligus — batas satu-dokumen sebelumnya murni buatan
    schema, bukan batas agent-nya."""

    base64: str
    filename: str = "document.pdf"


class ResponsesRequest(BaseModel):
    prompt: str
    response_format: ResponseFormat = Field(default_factory=ResponseFormat)
    # Optional document attachment (EP-13 PDF ingest, vision-based
    # extraction): when set, each page of the PDF is rendered to an image and
    # sent alongside prompt as native multimodal vision input instead of
    # relying on lossy externally-extracted text.
    document_base64: str | None = None
    document_filename: str | None = None
    # documents: BANYAK lampiran sekaligus (mis. beberapa PDF konteks untuk
    # menyusun kuesioner feedback). Tiap dokumen dirender ke gambar per halaman
    # dan digabung jadi satu pesan multimodal. document_base64 tunggal di atas
    # dipertahankan demi kompatibilitas pemanggil lama.
    documents: list[DocumentPayload] = Field(default_factory=list)


class AgentTaskRequest(BaseModel):
    """Fire-and-forget agent task: bridge menyusun playbook di background lalu
    MELAPOR BALIK ke app lewat callback_url (bukan mengandalkan LLM memanggil
    tool). instruction memuat seluruh konteks + perintah "balas HANYA JSON".
    Hasil (atau error) di-POST ke callback_url dengan header X-Cron-Secret."""

    instruction: str
    job_id: str
    callback_url: str
    # documents: daftar lampiran (utama). document_base64/document_filename
    # dipertahankan demi kompatibilitas pemanggil lama.
    documents: list[DocumentPayload] = Field(default_factory=list)
    callback_secret: str = ""
    document_base64: str | None = None
    document_filename: str | None = None


class ProviderConfigRequest(BaseModel):
    provider: str
    model: str
    base_url: str | None = None
    api_key: str
    # None = don't override (build_agent keeps the ENABLED_TOOLSETS env
    # default for chat mode); an explicit list (possibly empty) replaces it.
    enabled_toolsets: list[str] | None = None
