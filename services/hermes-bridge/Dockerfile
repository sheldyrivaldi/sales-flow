FROM python:3.11-slim

# git diperlukan untuk menginstall hermes-agent dari git+https://...
RUN apt-get update && apt-get install -y --no-install-recommends git && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY pyproject.toml ./
COPY app ./app

RUN pip install --no-cache-dir .

EXPOSE 8642

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8642"]
