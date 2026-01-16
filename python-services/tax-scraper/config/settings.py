"""
Tax Scraper Service Configuration
"""
from functools import lru_cache

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
    )

    # Service
    service_name: str = "tax-scraper"
    service_version: str = "0.1.0"
    environment: str = Field(default="development", alias="ENV")
    debug: bool = False

    # gRPC Server
    grpc_host: str = "0.0.0.0"
    grpc_port: int = 50051
    grpc_max_workers: int = 10
    grpc_reflection_enabled: bool = True

    # Hometax Configuration
    hometax_base_url: str = "https://www.hometax.go.kr"
    hometax_timeout: int = 30
    hometax_retry_count: int = 3
    hometax_retry_delay: float = 1.0

    # Browser (Playwright)
    browser_headless: bool = True
    browser_slow_mo: int = 0
    browser_timeout: int = 30000

    # Redis
    redis_host: str = "localhost"
    redis_port: int = 6379
    redis_db: int = 0
    redis_password: str | None = None

    # Encryption
    seed_key: str = Field(default="", alias="SEED_ENCRYPTION_KEY")

    # Logging
    log_level: str = "INFO"
    log_format: str = "json"

    @property
    def grpc_address(self) -> str:
        """Return gRPC server address."""
        return f"{self.grpc_host}:{self.grpc_port}"

    @property
    def redis_url(self) -> str:
        """Return Redis connection URL."""
        if self.redis_password:
            return f"redis://:{self.redis_password}@{self.redis_host}:{self.redis_port}/{self.redis_db}"
        return f"redis://{self.redis_host}:{self.redis_port}/{self.redis_db}"


@lru_cache
def get_settings() -> Settings:
    """Get cached settings instance."""
    return Settings()
