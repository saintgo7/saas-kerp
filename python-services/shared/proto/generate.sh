#!/bin/bash
# Proto Code Generation Script
# K-ERP SaaS - Phase 5: Tax Service

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_DIR="$SCRIPT_DIR"
PYTHON_OUT="$SCRIPT_DIR/../../tax-scraper/src/grpc_gen"
GO_OUT="$SCRIPT_DIR/../../../api/proto"

echo "=== K-ERP Proto Code Generation ==="
echo "Proto directory: $PROTO_DIR"

# Create output directories
mkdir -p "$PYTHON_OUT"

# Generate Python code
echo ""
echo "Generating Python gRPC code..."
python -m grpc_tools.protoc \
    -I"$PROTO_DIR" \
    --python_out="$PYTHON_OUT" \
    --pyi_out="$PYTHON_OUT" \
    --grpc_python_out="$PYTHON_OUT" \
    "$PROTO_DIR/tax.proto" \
    "$PROTO_DIR/common.proto"

# Fix imports in generated Python files
echo "Fixing Python imports..."
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' 's/^import tax_pb2/from . import tax_pb2/' "$PYTHON_OUT/tax_pb2_grpc.py" 2>/dev/null || true
    sed -i '' 's/^import common_pb2/from . import common_pb2/' "$PYTHON_OUT/tax_pb2.py" 2>/dev/null || true
else
    # Linux
    sed -i 's/^import tax_pb2/from . import tax_pb2/' "$PYTHON_OUT/tax_pb2_grpc.py" 2>/dev/null || true
    sed -i 's/^import common_pb2/from . import common_pb2/' "$PYTHON_OUT/tax_pb2.py" 2>/dev/null || true
fi

echo "Python code generated at: $PYTHON_OUT"

# Generate Go code (if protoc-gen-go is available)
if command -v protoc-gen-go &> /dev/null && [ -d "$GO_OUT" ]; then
    echo ""
    echo "Generating Go gRPC code..."
    mkdir -p "$GO_OUT/tax/v1"
    mkdir -p "$GO_OUT/common/v1"

    protoc \
        -I"$PROTO_DIR" \
        --go_out="$GO_OUT" \
        --go_opt=paths=source_relative \
        --go-grpc_out="$GO_OUT" \
        --go-grpc_opt=paths=source_relative \
        "$PROTO_DIR/tax.proto" \
        "$PROTO_DIR/common.proto"

    echo "Go code generated at: $GO_OUT"
else
    echo ""
    echo "Skipping Go code generation (protoc-gen-go not found or api/proto not exists)"
fi

echo ""
echo "=== Code generation complete ==="
