# gRPC Generator

Go-Python 간 gRPC 통신을 위한 Proto 정의 및 코드를 생성합니다.

## Trigger
`/grpc <service-name>` 또는 `/grpc`

## Arguments
- `service-name`: 서비스명 (scraper, insurance, etc.)

## Process

### 1. Proto 파일 생성
```
api/proto/${service}.proto
```
```protobuf
syntax = "proto3";

package kerp.${service};

option go_package = "github.com/kerp/api/proto/${service}";

// Health Check
message HealthRequest {}
message HealthResponse {
    enum Status {
        UNKNOWN = 0;
        SERVING = 1;
        NOT_SERVING = 2;
    }
    Status status = 1;
}

// Service Definition
service ${Service}Service {
    rpc HealthCheck(HealthRequest) returns (HealthResponse);
    // Add your RPCs here
}
```

### 2. Go Client 생성
```
internal/grpc/${service}_client.go
```
```go
package grpc

import (
    "context"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    pb "github.com/kerp/api/proto/${service}"
)

type ${Service}Client struct {
    conn   *grpc.ClientConn
    client pb.${Service}ServiceClient
    config *ClientConfig
}

type ClientConfig struct {
    Address string
    Timeout time.Duration
}

func New${Service}Client(cfg *ClientConfig) (*${Service}Client, error) {
    conn, err := grpc.Dial(cfg.Address,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return nil, fmt.Errorf("dial ${service}: %w", err)
    }

    return &${Service}Client{
        conn:   conn,
        client: pb.New${Service}ServiceClient(conn),
        config: cfg,
    }, nil
}

func (c *${Service}Client) HealthCheck(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    resp, err := c.client.HealthCheck(ctx, &pb.HealthRequest{})
    if err != nil {
        return err
    }

    if resp.Status != pb.HealthResponse_SERVING {
        return fmt.Errorf("service not serving")
    }
    return nil
}

func (c *${Service}Client) Close() error {
    return c.conn.Close()
}
```

### 3. Python Server 템플릿 생성
```
python-services/${service}/src/server.py
```
```python
"""${Service} gRPC Server"""

import asyncio
from concurrent import futures
import grpc

from proto import ${service}_pb2, ${service}_pb2_grpc


class ${Service}Servicer(${service}_pb2_grpc.${Service}ServiceServicer):
    """${Service} gRPC Service Implementation"""

    async def HealthCheck(self, request, context):
        return ${service}_pb2.HealthResponse(
            status=${service}_pb2.HealthResponse.SERVING
        )


async def serve(port: int = 50051):
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))
    ${service}_pb2_grpc.add_${Service}ServiceServicer_to_server(
        ${Service}Servicer(), server
    )

    listen_addr = f"[::]:{port}"
    server.add_insecure_port(listen_addr)

    print(f"Starting ${Service} gRPC server on {listen_addr}")
    await server.start()
    await server.wait_for_termination()


if __name__ == "__main__":
    asyncio.run(serve())
```

### 4. Proto 컴파일 명령
```bash
# Go
protoc --go_out=. --go-grpc_out=. api/proto/${service}.proto

# Python
python -m grpc_tools.protoc \
    -I api/proto \
    --python_out=python-services/${service}/src/proto \
    --grpc_python_out=python-services/${service}/src/proto \
    api/proto/${service}.proto
```

## Output Structure
```
api/proto/${service}.proto
internal/grpc/${service}_client.go
python-services/${service}/
├── src/
│   ├── server.py
│   └── proto/
│       ├── ${service}_pb2.py
│       └── ${service}_pb2_grpc.py
└── Dockerfile
```

## Port Convention
| Service | Port |
|---------|------|
| Tax Scraper | 50051 |
| Insurance EDI | 50052 |
| Custom | 50053+ |
