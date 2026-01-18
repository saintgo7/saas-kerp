# Tax Scraper Service Dockerfile
# K-ERP SaaS - Phase 5: Tax/Hometax Integration

FROM python:3.11-slim-bookworm AS base

# Set environment variables
ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PLAYWRIGHT_BROWSERS_PATH=/ms-playwright

WORKDIR /app

# Install system dependencies for Playwright
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Playwright dependencies
    libnss3 \
    libnspr4 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libdbus-1-3 \
    libxkbcommon0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libasound2 \
    libpango-1.0-0 \
    libcairo2 \
    libx11-6 \
    libxext6 \
    fonts-nanum \
    # Build dependencies
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Builder stage
FROM base AS builder

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Install Playwright browsers
RUN playwright install chromium --with-deps

# Generate gRPC code
COPY proto/ ./proto/
COPY scripts/generate_grpc.sh ./scripts/
RUN chmod +x scripts/generate_grpc.sh && \
    mkdir -p src/grpc_gen && \
    python -m grpc_tools.protoc \
        -I./proto \
        --python_out=./src/grpc_gen \
        --pyi_out=./src/grpc_gen \
        --grpc_python_out=./src/grpc_gen \
        ./proto/tax.proto || true

# Production stage
FROM base AS production

# Copy installed packages from builder
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin
COPY --from=builder /ms-playwright /ms-playwright

# Copy application code
COPY . .
COPY --from=builder /app/src/grpc_gen ./src/grpc_gen

# Create non-root user
RUN useradd -m -u 1000 appuser && \
    chown -R appuser:appuser /app /ms-playwright

USER appuser

# Expose gRPC port
EXPOSE 50051

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD python -c "import grpc; channel = grpc.insecure_channel('localhost:50051'); grpc.channel_ready_future(channel).result(timeout=5)"

# Run the service
CMD ["python", "main.py"]
