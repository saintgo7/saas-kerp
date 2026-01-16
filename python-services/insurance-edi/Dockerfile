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

# Copy proto files and generate gRPC code
COPY python-services/insurance-edi/proto/ ./proto/
RUN mkdir -p generated && \
    python -m grpc_tools.protoc \
        -Iproto \
        --python_out=generated \
        --grpc_python_out=generated \
        --pyi_out=generated \
        proto/insurance.proto && \
    sed -i 's/import insurance_pb2/from . import insurance_pb2/g' generated/insurance_pb2_grpc.py && \
    touch generated/__init__.py

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

# Copy Python packages from builder to site-packages (accessible to all users)
COPY --from=builder /root/.local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages

# Copy source code
COPY --chown=kerp:kerp python-services/insurance-edi/ .
COPY --chown=kerp:kerp python-services/shared/ ./shared/

# Copy generated proto files from builder
COPY --from=builder --chown=kerp:kerp /app/generated ./generated/

# Expose ports
EXPOSE 50052 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import grpc; ch=grpc.insecure_channel('localhost:50052'); grpc.channel_ready_future(ch).result(timeout=5)" || exit 1

# Run as non-root user
USER kerp

# Production command
CMD ["python", "main.py"]
