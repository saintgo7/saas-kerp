#!/bin/bash
# Generate Python gRPC code from proto files

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

PROTO_DIR="$PROJECT_ROOT/proto"
OUTPUT_DIR="$PROJECT_ROOT/src/grpc_gen"

echo "Generating gRPC code..."
echo "Proto dir: $PROTO_DIR"
echo "Output dir: $OUTPUT_DIR"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Generate Python code from proto files
python -m grpc_tools.protoc \
    -I"$PROTO_DIR" \
    --python_out="$OUTPUT_DIR" \
    --pyi_out="$OUTPUT_DIR" \
    --grpc_python_out="$OUTPUT_DIR" \
    "$PROTO_DIR/tax.proto"

# Fix imports in generated files
# The generated code uses absolute imports which need to be relative
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' 's/^import tax_pb2/from . import tax_pb2/' "$OUTPUT_DIR/tax_pb2_grpc.py"
else
    # Linux
    sed -i 's/^import tax_pb2/from . import tax_pb2/' "$OUTPUT_DIR/tax_pb2_grpc.py"
fi

# Create __init__.py
cat > "$OUTPUT_DIR/__init__.py" << 'EOF'
"""Generated gRPC code for tax service."""

from .tax_pb2 import *
from .tax_pb2_grpc import *

__all__ = [
    "tax_pb2",
    "tax_pb2_grpc",
]
EOF

echo "gRPC code generation complete!"
echo "Generated files:"
ls -la "$OUTPUT_DIR"
