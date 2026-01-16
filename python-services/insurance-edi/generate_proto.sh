#!/bin/bash
# Generate Python gRPC code from proto files

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_DIR="${SCRIPT_DIR}/proto"
OUTPUT_DIR="${SCRIPT_DIR}/generated"

echo "Generating gRPC code from proto files..."

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Generate Python code
python -m grpc_tools.protoc \
    -I"${PROTO_DIR}" \
    --python_out="${OUTPUT_DIR}" \
    --grpc_python_out="${OUTPUT_DIR}" \
    --pyi_out="${OUTPUT_DIR}" \
    "${PROTO_DIR}/insurance.proto"

# Fix imports in generated files
# grpc_tools generates imports like 'import insurance_pb2'
# but we need 'from . import insurance_pb2' for package imports
sed -i.bak 's/import insurance_pb2/from . import insurance_pb2/g' "${OUTPUT_DIR}/insurance_pb2_grpc.py" 2>/dev/null || \
    sed -i '' 's/import insurance_pb2/from . import insurance_pb2/g' "${OUTPUT_DIR}/insurance_pb2_grpc.py"

rm -f "${OUTPUT_DIR}/"*.bak

echo "Proto generation complete!"
echo "Generated files:"
ls -la "${OUTPUT_DIR}"
