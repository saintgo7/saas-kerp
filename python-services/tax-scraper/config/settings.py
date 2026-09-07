"""
Tax Scraper Service Configuration

Loads configuration from environment variables with sensible defaults.
"""

from functools import lru_cache
from typing import Optional

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    # Service
    service_name: str = "tax-scraper"
    service_version: str = "0.2.0"
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
    hometax_login_timeout: int = 60000  # ms, page load for the login flow

    # Hometax live mode.
    #
    # The Hometax scraping paths (login, invoice search, invoice issuance) have
    # never been validated against the live site: the selectors in
    # `src/hometax/constants.py` are guesses at a WebSquare DOM nobody has
    # checked. Default OFF means those paths refuse with an explicit error
    # instead of returning fabricated results -- which is what they used to do
    # (a hardcoded "NTS-CONFIRM-12345" for every issuance).
    #
    # Turn this on only after verifying the selectors against the real site in a
    # controlled environment.
    hometax_live_mode: bool = Field(default=False, alias="HOMETAX_LIVE_MODE")

    # Session limits for browser contexts (each holds an authenticated session)
    hometax_session_ttl_minutes: int = 60
    hometax_max_sessions: int = 50

    # Browser (Playwright)
    browser_headless: bool = True
    browser_slow_mo: int = 0
    browser_timeout: int = 30000
    playwright_headless: bool = True  # Alias for backward compatibility
    playwright_slow_mo: int = 0  # Alias for backward compatibility
    playwright_timeout: int = 30000  # Alias for backward compatibility

    # Popbill API Configuration
    popbill_link_id: str = Field(default="", alias="POPBILL_LINK_ID")
    popbill_secret_key: str = Field(default="", alias="POPBILL_SECRET_KEY")
    # Defaults to the PRODUCTION endpoint.
    #
    # This used to default to True. Deploying without setting POPBILL_IS_TEST
    # sent every 세금계산서 to popbill-test.linkhub.co.kr: the call succeeded, the
    # UI said "발급 완료", and nothing reached the 국세청. Opting in to the test
    # endpoint has to be deliberate.
    popbill_is_test: bool = Field(default=False, alias="POPBILL_IS_TEST")

    # Redis
    redis_host: str = "localhost"
    redis_port: int = 6379
    redis_db: int = 0
    redis_password: Optional[str] = None

    # Encryption
    seed_key: str = Field(default="", alias="SEED_ENCRYPTION_KEY")

    # Logging
    log_level: str = "INFO"
    log_format: str = "json"

    @property
    def is_production(self) -> bool:
        """Whether this looks like a non-development deployment."""
        return self.environment.lower() not in ("development", "dev", "local", "test")

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
