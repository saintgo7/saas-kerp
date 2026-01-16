"""
Insurance EDI Service Configuration
"""
from pydantic_settings import BaseSettings
from pydantic import Field
from typing import Optional


class EDIServerConfig(BaseSettings):
    """EDI server connection settings for each insurance provider."""

    # National Pension Service (NPS) - 국민연금공단
    nps_host: str = Field(default="edi.nps.or.kr", env="NPS_EDI_HOST")
    nps_port: int = Field(default=9100, env="NPS_EDI_PORT")
    nps_timeout: int = Field(default=30, env="NPS_EDI_TIMEOUT")

    # National Health Insurance Service (NHIS) - 건강보험공단
    nhis_host: str = Field(default="edi.nhis.or.kr", env="NHIS_EDI_HOST")
    nhis_port: int = Field(default=9100, env="NHIS_EDI_PORT")
    nhis_timeout: int = Field(default=30, env="NHIS_EDI_TIMEOUT")

    # Employment Insurance (EI) - 고용산재보험
    ei_host: str = Field(default="edi.comwel.or.kr", env="EI_EDI_HOST")
    ei_port: int = Field(default=9100, env="EI_EDI_PORT")
    ei_timeout: int = Field(default=30, env="EI_EDI_TIMEOUT")


class CryptoConfig(BaseSettings):
    """Cryptography settings."""

    # ARIA encryption key (128-bit)
    aria_key: Optional[str] = Field(default=None, env="ARIA_ENCRYPTION_KEY")

    # Certificate paths
    cert_path: str = Field(default="/app/certs", env="CERT_PATH")
    private_key_path: Optional[str] = Field(default=None, env="PRIVATE_KEY_PATH")
    public_key_path: Optional[str] = Field(default=None, env="PUBLIC_KEY_PATH")


class Settings(BaseSettings):
    """Main application settings."""

    # Service info
    service_name: str = "insurance-edi"
    service_version: str = "0.1.0"

    # gRPC server
    grpc_host: str = Field(default="0.0.0.0", env="GRPC_HOST")
    grpc_port: int = Field(default=50052, env="GRPC_PORT")
    grpc_max_workers: int = Field(default=10, env="GRPC_MAX_WORKERS")

    # Environment
    environment: str = Field(default="development", env="ENVIRONMENT")
    debug: bool = Field(default=False, env="DEBUG")

    # Logging
    log_level: str = Field(default="INFO", env="LOG_LEVEL")
    log_format: str = Field(default="json", env="LOG_FORMAT")

    # EDI servers
    edi: EDIServerConfig = EDIServerConfig()

    # Crypto
    crypto: CryptoConfig = CryptoConfig()

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"
        env_nested_delimiter = "__"


# Global settings instance
settings = Settings()
