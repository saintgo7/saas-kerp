# Python Insurance EDI Service Dockerfile
# Multi-stage build for gRPC service

# =============================================================================
# Development Stage
# =============================================================================
FROM python:3.11-slim-bookworm AS development

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
COPY python-services/insurance-edi/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy source code
COPY python-services/insurance-edi/ .
COPY python-services/shared/ ./shared/

# Expose ports
EXPOSE 50052 9090

# Development command with hot-reload
CMD ["python", "-m", "watchfiles", "main.main", "."]

# =============================================================================
# Builder Stage
# =============================================================================
FROM python:3.11-slim-bookworm AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
COPY python-services/insurance-edi/requirements.txt .
RUN pip install --no-cache-dir --user -r requirements.txt

# =============================================================================
# Production Stage
# =============================================================================
FROM python:3.11-slim-bookworm AS production

WORKDIR /app

# Create non-root user
RUN groupadd -r kerp && useradd -r -g kerp kerp

# Install runtime dependencies only
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy Python packages from builder
COPY --from=builder /root/.local /root/.local
ENV PATH=/root/.local/bin:$PATH

# Copy source code
COPY --chown=kerp:kerp python-services/insurance-edi/ .
COPY --chown=kerp:kerp python-services/shared/ ./shared/

# Expose ports
EXPOSE 50052 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:9090/health || exit 1

# Run as non-root user
USER kerp

# Production command
CMD ["python", "main.py"]
