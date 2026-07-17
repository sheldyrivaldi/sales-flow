"""Renders PDF pages to base64 JPEG data URLs for vision-based document
reading (EP-13 Company Profile PDF ingest).

The Go side used to extract plain text locally (github.com/ledongthuc/pdf)
and embed it in the prompt — lossy for tables/layout (a Company Profile RFI
is mostly tables), which is why extraction quality looked incomplete. This
module instead rasterizes each page as an image and hands it to Hermes as
native vision input, so the model reads the document the way a person would.
"""

from __future__ import annotations

import base64
import io

import pypdfium2 as pdfium
from fastapi import HTTPException

# Hard cap on pages rendered — keeps the request payload and per-call image
# cost bounded regardless of how long a source document is. 20 pages covers
# the RFI use case (a company profile / capability deck) comfortably; a
# longer document still gets its first 20 pages read rather than failing
# outright.
MAX_PAGES = 20

# ~144 DPI (pdfium's native render unit is 72 DPI) — legible for body text
# and table contents without producing huge payloads per page.
RENDER_SCALE = 2.0

JPEG_QUALITY = 85


def render_pdf_pages_to_data_urls(pdf_bytes: bytes, max_pages: int = MAX_PAGES) -> list[str]:
    """Renders up to max_pages pages of pdf_bytes to JPEG data URLs.

    Raises HTTPException(400) if pdf_bytes isn't a PDF pdfium can open —
    the caller (routes/responses.py) lets that propagate as a normal FastAPI
    error response, which the Go side already treats as an AI-extraction
    failure (degrades to manual entry, never fails the upload itself).
    """
    try:
        pdf = pdfium.PdfDocument(pdf_bytes)
    except Exception as exc:  # pdfium raises its own exception types
        raise HTTPException(status_code=400, detail=f"dokumen tidak dapat dibuka sebagai PDF: {exc}") from exc

    try:
        n_pages = min(len(pdf), max_pages)
        urls: list[str] = []
        for i in range(n_pages):
            page = pdf[i]
            bitmap = page.render(scale=RENDER_SCALE)
            pil_image = bitmap.to_pil().convert("RGB")
            buf = io.BytesIO()
            pil_image.save(buf, format="JPEG", quality=JPEG_QUALITY)
            b64 = base64.b64encode(buf.getvalue()).decode("ascii")
            urls.append(f"data:image/jpeg;base64,{b64}")
        if not urls:
            raise HTTPException(status_code=400, detail="PDF tidak memiliki halaman")
        return urls
    finally:
        pdf.close()
