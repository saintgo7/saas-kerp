# Test Generator

Go/Python 테스트 코드를 생성합니다.
Table-driven 테스트, Mock, Integration 테스트를 지원합니다.

## Trigger
`/test <lang> <target>` 또는 `/test`

## Arguments
- `lang`: go, python
- `target`: 테스트 대상 파일/패키지 경로

## Go Tests

### 1. Unit Test (Table-Driven)
```
internal/service/${resource}_service_test.go
```
```go
package service_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/kerp/internal/service"
    "github.com/kerp/internal/domain"
)

// MockRepository
type Mock${Resource}Repository struct {
    mock.Mock
}

func (m *Mock${Resource}Repository) Create(ctx context.Context, entity *domain.${Resource}) error {
    args := m.Called(ctx, entity)
    return args.Error(0)
}

func (m *Mock${Resource}Repository) FindByID(ctx context.Context, companyID, id uuid.UUID) (*domain.${Resource}, error) {
    args := m.Called(ctx, companyID, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*domain.${Resource}), args.Error(1)
}

// Tests
func TestCreate${Resource}(t *testing.T) {
    tests := []struct {
        name    string
        input   *dto.Create${Resource}Request
        setup   func(*Mock${Resource}Repository)
        wantErr bool
    }{
        {
            name: "success",
            input: &dto.Create${Resource}Request{
                // fields
            },
            setup: func(m *Mock${Resource}Repository) {
                m.On("Create", mock.Anything, mock.Anything).Return(nil)
            },
            wantErr: false,
        },
        {
            name: "repository error",
            input: &dto.Create${Resource}Request{},
            setup: func(m *Mock${Resource}Repository) {
                m.On("Create", mock.Anything, mock.Anything).
                    Return(errors.New("db error"))
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := new(Mock${Resource}Repository)
            tt.setup(mockRepo)

            svc := service.New${Resource}Service(mockRepo, zap.NewNop())

            ctx := context.WithValue(context.Background(), "company_id", uuid.New())
            _, err := svc.Create(ctx, tt.input)

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }

            mockRepo.AssertExpectations(t)
        })
    }
}
```

### 2. Handler Test
```go
func TestHandler_Create(t *testing.T) {
    gin.SetMode(gin.TestMode)

    tests := []struct {
        name       string
        body       interface{}
        setup      func(*MockService)
        wantStatus int
    }{
        {
            name: "success",
            body: map[string]interface{}{"field": "value"},
            setup: func(m *MockService) {
                m.On("Create", mock.Anything, mock.Anything).
                    Return(&dto.Response{}, nil)
            },
            wantStatus: http.StatusCreated,
        },
        {
            name:       "invalid body",
            body:       "invalid",
            setup:      func(m *MockService) {},
            wantStatus: http.StatusBadRequest,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockSvc := new(MockService)
            tt.setup(mockSvc)

            h := handler.New${Resource}Handler(mockSvc)
            router := gin.New()
            router.POST("/", h.Create)

            body, _ := json.Marshal(tt.body)
            req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
            req.Header.Set("Content-Type", "application/json")

            w := httptest.NewRecorder()
            router.ServeHTTP(w, req)

            assert.Equal(t, tt.wantStatus, w.Code)
        })
    }
}
```

## Python Tests

### 1. Unit Test (pytest)
```
python-services/${service}/tests/test_${module}.py
```
```python
"""${Module} Tests"""

import pytest
from unittest.mock import Mock, AsyncMock, patch
from src.${module}.handler import ${Module}Handler


@pytest.fixture
def handler():
    return ${Module}Handler()


@pytest.fixture
def mock_cipher():
    with patch('src.crypto.seed.SEEDCipher') as mock:
        yield mock


class TestEncryption:
    def test_encrypt_decrypt(self):
        from src.crypto.seed import SEEDCipher

        key = b'0123456789abcdef'
        iv = b'fedcba9876543210'
        cipher = SEEDCipher(key, iv)

        plaintext = b"Hello, World!"
        encrypted = cipher.encrypt(plaintext)
        decrypted = cipher.decrypt(encrypted)

        assert decrypted == plaintext

    def test_invalid_key_length(self):
        with pytest.raises(ValueError, match="Key must be 16 bytes"):
            SEEDCipher(b"short", b"0" * 16)


class Test${Module}Handler:
    @pytest.mark.asyncio
    async def test_process_success(self, handler, mock_cipher):
        request = Mock()
        request.data = "test"

        result = await handler.process(request)

        assert result is not None

    @pytest.mark.asyncio
    async def test_process_error(self, handler):
        with pytest.raises(Exception):
            await handler.process(None)
```

### 2. Integration Test
```python
"""Integration Tests with gRPC"""

import pytest
import grpc
from src.proto import ${service}_pb2, ${service}_pb2_grpc


@pytest.fixture
async def grpc_channel():
    channel = grpc.aio.insecure_channel('localhost:50051')
    yield channel
    await channel.close()


@pytest.fixture
async def stub(grpc_channel):
    return ${service}_pb2_grpc.${Service}ServiceStub(grpc_channel)


class TestGRPCService:
    @pytest.mark.asyncio
    async def test_health_check(self, stub):
        request = ${service}_pb2.HealthRequest()
        response = await stub.HealthCheck(request)

        assert response.status == ${service}_pb2.HealthResponse.SERVING
```

## Commands
```bash
# Go
go test ./... -v -cover
go test -run TestCreate ./internal/service/...

# Python
pytest tests/ -v --cov=src
pytest tests/test_${module}.py -v
```
