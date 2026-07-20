from __future__ import annotations

import json
import logging
import re
from typing import Any

import httpx
from fastapi import APIRouter, BackgroundTasks, Depends

from app.agent_factory import build_agent, model_chain
from app.auth import require_bearer
from app.routes.chat import _attachment_image_parts
from app.schemas import AgentTaskRequest, DocumentPayload

log = logging.getLogger("hermes_bridge.agent_task")

router = APIRouter(dependencies=[Depends(require_bearer)])

_JSON_SYSTEM = (
    "Balas HANYA satu objek JSON valid sesuai schema pada instruksi. "
    "Tanpa penjelasan, tanpa markdown, tanpa code-fence."
)

# Dipakai saat percobaan ulang setelah output sebelumnya gagal diparse.
_JSON_RETRY_HINT = (
    "PENTING: respons sebelumnya BUKAN JSON valid. Periksa ulang setiap kurung "
    "buka/tutup dan koma sebelum menjawab. Keluarkan hanya JSON."
)


def _repair_brackets(t: str) -> str:
    """Perbaiki kurung penutup yang HILANG pada JSON yang sudah tidak valid.

    Model kadang melewatkan satu penutup pada struktur bersarang dalam, mis.
    menulis `"items":["a","b"}]` padahal seharusnya `"items":["a","b"]}]`.
    Scanner ini melacak tumpukan kurung (mengabaikan isi string) dan, saat
    menemui penutup yang tidak cocok dengan puncak tumpukan, menyisipkan
    penutup yang benar lebih dulu. Hanya MENAMBAH penutup, tidak pernah
    membuang karakter, dan hanya dipakai sebagai upaya terakhir.
    """
    out: list[str] = []
    stack: list[str] = []
    closer = {"{": "}", "[": "]"}
    in_str = False
    esc = False
    for ch in t:
        if in_str:
            out.append(ch)
            if esc:
                esc = False
            elif ch == "\\":
                esc = True
            elif ch == '"':
                in_str = False
            continue
        if ch == '"':
            in_str = True
            out.append(ch)
            continue
        if ch in "{[":
            stack.append(ch)
            out.append(ch)
            continue
        if ch in "}]":
            # Tutup kontainer dalam yang terlewat sampai penutup ini cocok.
            while stack and closer[stack[-1]] != ch:
                out.append(closer[stack.pop()])
            if stack:
                stack.pop()
            out.append(ch)
            continue
        out.append(ch)
    # Tutup sisa kontainer yang masih terbuka (output terpotong).
    while stack:
        out.append(closer[stack.pop()])
    return "".join(out)


def _extract_json(text: str) -> Any | None:
    """Ambil objek JSON dari output model: buang code-fence, potong dari '{'
    pertama sampai '}' terakhir, lalu — bila masih gagal — coba perbaiki
    kurung penutup yang hilang. Payload deck cukup besar sehingga satu kurung
    meleset tidak boleh menggagalkan seluruh playbook."""
    t = (text or "").strip()
    if t.startswith("```"):
        t = re.sub(r"^```[a-zA-Z]*\n", "", t)
        if t.endswith("```"):
            t = t[: -3]
    t = t.strip()
    try:
        return json.loads(t)
    except Exception:
        pass

    start = t.find("{")
    if start == -1:
        return None

    end = t.rfind("}")
    if end > start:
        body = t[start : end + 1]
        try:
            return json.loads(body)
        except Exception:
            pass
        try:
            obj = json.loads(_repair_brackets(body))
            log.warning("agent-task: JSON model diperbaiki (kurung penutup hilang)")
            return obj
        except Exception:
            pass

    # Upaya terakhir: output terpotong sehingga tidak punya '}' penutup sama
    # sekali — tutup semua kontainer yang masih terbuka. Deck yang tersusun
    # sebagian jauh lebih berguna daripada job gagal total.
    try:
        obj = json.loads(_repair_brackets(t[start:]))
        log.warning("agent-task: JSON model terpotong, dipulihkan sebagian")
        return obj
    except Exception:
        return None


def _looks_like_provider_error(text: str) -> bool:
    low = (text or "").strip().lower()
    if low.startswith("{") or low.startswith("["):
        return False
    return any(m in low for m in ("api call failed", "connection error", "broken pipe"))


def _post_callback(url: str, secret: str, payload: dict[str, Any]) -> None:
    try:
        headers = {"Content-Type": "application/json"}
        if secret:
            headers["X-Cron-Secret"] = secret
        httpx.post(url, json=payload, headers=headers, timeout=30)
    except Exception:  # pragma: no cover
        log.exception("agent-task: gagal POST callback ke %s", url)


def _run_agent_task(req: AgentTaskRequest) -> None:
    """Susun playbook di background (dengan retry saat koneksi provider flaky),
    lalu POST hasil/error ke callback_url app. app TIDAK menahan koneksi."""
    # Kumpulkan SEMUA lampiran: daftar `documents` (utama) plus bentuk tunggal
    # lama demi kompatibilitas. Satu tugas boleh membawa banyak dokumen —
    # tiap PDF direntang jadi image part per halaman, lalu digabung.
    docs = list(req.documents)
    if req.document_base64:
        docs.append(DocumentPayload(base64=req.document_base64, filename=req.document_filename or "document.pdf"))

    content: Any = req.instruction
    if docs:
        parts: list[dict[str, Any]] = [{"type": "text", "text": req.instruction}]
        for d in docs:
            try:
                parts.extend(_attachment_image_parts(d.base64, d.filename))
            except Exception:
                # Satu lampiran rusak tidak boleh menggagalkan seluruh tugas;
                # sisanya tetap dikirim dan agent diberi tahu mana yang gagal.
                log.warning("agent-task: lampiran %s gagal diproses, dilewati", d.filename)
                parts.append({"type": "text", "text": f"(lampiran {d.filename} tidak bisa dibaca)"})
        content = parts

    obj: Any | None = None
    last_err = ""
    # Tiap percobaan memakai model BERBEDA bila tersedia: saat backend
    # menolak diam-diam sebuah model, mengulanginya hanya membuang 90 detik.
    chain = model_chain()
    attempts = max(3, len(chain))
    for attempt in range(attempts):
        model = chain[attempt] if attempt < len(chain) else chain[-1]
        try:
            agent = build_agent(mode="chat", model_override=model)
            # Setelah output tak-terparse, pertegas permintaan JSON valid.
            system = _JSON_SYSTEM if attempt == 0 else _JSON_SYSTEM + " " + _JSON_RETRY_HINT
            result = agent.run_conversation(
                content,
                system_message=system,
                conversation_history=[],
                task_id="",
            )
            text = (result.get("final_response") or "").strip()
            if not text or _looks_like_provider_error(text):
                last_err = text or "respons kosong"
                log.warning("agent-task attempt %d (model=%s): provider gagal", attempt + 1, model)
                continue
            # Parse DI DALAM loop: output yang bukan JSON valid harus memicu
            # percobaan ulang, bukan langsung menggagalkan job.
            obj = _extract_json(text)
            if obj is not None:
                break
            last_err = "Output AI bukan JSON valid."
            log.warning("agent-task attempt %d (model=%s): output bukan JSON valid (%d char)", attempt + 1, model, len(text))
        except Exception as exc:  # pragma: no cover
            last_err = str(exc)
            log.warning("agent-task attempt %d (model=%s) gagal: %s", attempt + 1, model, last_err)

    if obj is None:
        _post_callback(req.callback_url, req.callback_secret, {"job_id": req.job_id, "error": (last_err or "Output AI bukan JSON valid.")[:400]})
    else:
        _post_callback(req.callback_url, req.callback_secret, {"job_id": req.job_id, "content": obj})


@router.post("/v1/agent-task", status_code=202)
def agent_task(req: AgentTaskRequest, background: BackgroundTasks) -> dict[str, str]:
    """Terima tugas, balas 202 SEKETIKA, kerjakan di background dan lapor balik
    ke app lewat callback_url — app tidak menahan koneksi panjang."""
    background.add_task(_run_agent_task, req)
    return {"status": "accepted"}
