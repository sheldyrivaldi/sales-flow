import os


class Settings:
    api_server_key: str
    hermes_model: str
    port: int
    enabled_toolsets: list[str]
    openai_api_key: str
    openrouter_api_key: str

    def __init__(self) -> None:
        key = os.getenv("API_SERVER_KEY", "")
        if not key:
            raise RuntimeError("API_SERVER_KEY env var wajib diisi")
        self.api_server_key = key
        self.hermes_model = os.getenv("HERMES_MODEL", "default")
        self.port = int(os.getenv("PORT", "8642"))
        toolsets_raw = os.getenv("ENABLED_TOOLSETS", "web")
        self.enabled_toolsets = [t.strip() for t in toolsets_raw.split(",") if t.strip()]
        self.openai_api_key = os.getenv("OPENAI_API_KEY", "")
        self.openrouter_api_key = os.getenv("OPENROUTER_API_KEY", "")


_settings: Settings | None = None


def get_settings() -> Settings:
    global _settings
    if _settings is None:
        _settings = Settings()
    return _settings


def reset_settings() -> None:
    """Reset cached settings (dipakai di tests)."""
    global _settings
    _settings = None
