#!/usr/bin/env python3
"""Tax Scraper Service - Main entry point."""

import asyncio
import sys
from pathlib import Path

# Add project root to path
sys.path.insert(0, str(Path(__file__).parent))

from config.settings import get_settings
from src.server import serve
from src.utils.logger import setup_logger


def main() -> None:
    """Main entry point."""
    setup_logger()
    settings = get_settings()

    print(f"""
    ===============================================
    K-ERP Tax Scraper Service v{settings.service_version}
    ===============================================
    Environment: {settings.environment}
    gRPC Server: {settings.grpc_host}:{settings.grpc_port}
    Headless:    {settings.playwright_headless}
    ===============================================
    """)

    asyncio.run(serve())


if __name__ == "__main__":
    main()
