import io

import pytest
from PIL import Image

from app.pdf_render import render_pdf_pages_to_data_urls


def _make_pdf(n_pages: int = 1) -> bytes:
    """Builds a real, valid multi-page PDF via Pillow (already a transitive
    hermes-agent dependency) — avoids hand-crafting PDF byte offsets, which
    pdfium (a full spec-compliant engine, unlike the lenient pure-Go text
    extractor this replaces) is not forgiving about."""
    pages = [Image.new("RGB", (200, 300), "white") for _ in range(n_pages)]
    buf = io.BytesIO()
    pages[0].save(buf, format="PDF", save_all=True, append_images=pages[1:])
    return buf.getvalue()


def test_render_pdf_pages_to_data_urls_single_page():
    urls = render_pdf_pages_to_data_urls(_make_pdf(1))
    assert len(urls) == 1
    assert urls[0].startswith("data:image/jpeg;base64,")


def test_render_pdf_pages_to_data_urls_caps_at_max_pages():
    urls = render_pdf_pages_to_data_urls(_make_pdf(5), max_pages=3)
    assert len(urls) == 3


def test_render_pdf_pages_to_data_urls_invalid_bytes_raises_400():
    from fastapi import HTTPException

    with pytest.raises(HTTPException) as exc_info:
        render_pdf_pages_to_data_urls(b"not a pdf at all")
    assert exc_info.value.status_code == 400
