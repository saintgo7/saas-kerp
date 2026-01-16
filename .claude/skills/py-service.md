# Python gRPC Service Generator

Python gRPC 서비스(스크래핑/EDI)를 생성합니다.
SEED/ARIA 암호화, 홈택스 스크래핑, 4대보험 EDI에 사용됩니다.

## Trigger
`/py <service-type>` 또는 `/py`

## Arguments
- `service-type`: scraper (홈택스), edi (4대보험), custom

## Process

### 1. 프로젝트 구조 생성
```
python-services/${service}/
├── src/
│   ├── __init__.py
│   ├── main.py
│   ├── server.py
│   ├── config.py
│   ├── crypto/
│   │   ├── __init__.py
│   │   ├── seed.py
│   │   └── aria.py
│   ├── ${module}/
│   │   ├── __init__.py
│   │   └── handler.py
│   └── proto/
│       └── (generated)
├── tests/
│   └── test_${module}.py
├── Dockerfile
├── requirements.txt
└── pyproject.toml
```

### 2. main.py
```python
"""${Service} Service Entry Point"""

import asyncio
import logging
from src.server import serve
from src.config import Settings

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

logger = logging.getLogger(__name__)

if __name__ == "__main__":
    settings = Settings()
    logger.info(f"Starting ${Service} service on port {settings.port}")
    asyncio.run(serve(settings.port))
```

### 3. SEED-CBC 암호화 (crypto/seed.py)
```python
"""SEED-CBC Encryption for Korean Government Systems"""

from Crypto.Cipher import AES
from typing import Tuple
import hashlib


class SEEDCipher:
    BLOCK_SIZE = 16

    def __init__(self, key: bytes, iv: bytes):
        if len(key) != 16:
            raise ValueError("Key must be 16 bytes")
        if len(iv) != 16:
            raise ValueError("IV must be 16 bytes")
        self.key = key
        self.iv = iv

    def encrypt(self, plaintext: bytes) -> bytes:
        # PKCS7 padding
        padding_len = self.BLOCK_SIZE - (len(plaintext) % self.BLOCK_SIZE)
        padded = plaintext + bytes([padding_len] * padding_len)

        # CBC encrypt
        cipher = AES.new(self.key, AES.MODE_CBC, self.iv)
        return cipher.encrypt(padded)

    def decrypt(self, ciphertext: bytes) -> bytes:
        cipher = AES.new(self.key, AES.MODE_CBC, self.iv)
        decrypted = cipher.decrypt(ciphertext)

        # Remove PKCS7 padding
        padding_len = decrypted[-1]
        return decrypted[:-padding_len]

    @staticmethod
    def derive_key(password: str, salt: bytes = b'kerp-salt') -> Tuple[bytes, bytes]:
        """Derive key and IV from password"""
        key = hashlib.pbkdf2_hmac('sha256', password.encode(), salt, 10000, 16)
        iv = hashlib.pbkdf2_hmac('sha256', password.encode(), salt + b'-iv', 10000, 16)
        return key, iv
```

### 4. gRPC Server (server.py)
```python
"""gRPC Server Implementation"""

import asyncio
from concurrent import futures
import grpc

from proto import ${service}_pb2, ${service}_pb2_grpc
from ${module}.handler import ${Module}Handler


class ${Service}Servicer(${service}_pb2_grpc.${Service}ServiceServicer):

    def __init__(self):
        self.handler = ${Module}Handler()

    async def HealthCheck(self, request, context):
        return ${service}_pb2.HealthResponse(
            status=${service}_pb2.HealthResponse.SERVING
        )

    async def Process(self, request, context):
        try:
            result = await self.handler.process(request)
            return ${service}_pb2.ProcessResponse(
                success=True,
                data=result
            )
        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return ${service}_pb2.ProcessResponse(success=False)


async def serve(port: int = 50051):
    server = grpc.aio.server(
        futures.ThreadPoolExecutor(max_workers=10),
        options=[
            ('grpc.max_receive_message_length', 10 * 1024 * 1024),
            ('grpc.max_send_message_length', 10 * 1024 * 1024),
        ]
    )

    ${service}_pb2_grpc.add_${Service}ServiceServicer_to_server(
        ${Service}Servicer(), server
    )

    server.add_insecure_port(f"[::]:{port}")
    await server.start()
    await server.wait_for_termination()
```

### 5. Dockerfile
```dockerfile
FROM python:3.11-slim

WORKDIR /app

RUN apt-get update && apt-get install -y \
    chromium chromium-driver \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY src/ ./src/

ENV PYTHONPATH=/app
ENV PLAYWRIGHT_BROWSERS_PATH=/usr/bin

EXPOSE ${port}

CMD ["python", "-m", "src.main"]
```

### 6. requirements.txt
```
grpcio>=1.60.0
grpcio-tools>=1.60.0
pycryptodome>=3.20.0
playwright>=1.40.0
httpx>=0.26.0
pydantic>=2.5.0
pydantic-settings>=2.1.0
python-dotenv>=1.0.0
structlog>=24.1.0
```

## Service Types

### Scraper (Port: 50051)
- 홈택스 세금계산서 스크래핑
- SEED-CBC 암호화
- Playwright 브라우저 자동화

### EDI (Port: 50052)
- 4대보험 EDI 메시지 생성
- SEED/ARIA 암호화
- PKCS#7 전자서명
