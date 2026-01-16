# Go REST API Scaffold

Go API 엔드포인트를 Clean Architecture 패턴으로 빠르게 생성합니다.

## Trigger
`/api <module> <resource>` 또는 `/api`

## Arguments
- `module`: 모듈명 (account, invoice, payroll, purchase, sales, inventory)
- `resource`: 리소스명 (kebab-case)

## Process

### 1. Domain Model 생성
```
internal/domain/${resource}.go
```
```go
package domain

type ${Resource} struct {
    ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
    CompanyID uuid.UUID `json:"company_id" gorm:"type:uuid;not null;index"`
    // ... fields
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
```

### 2. DTO 생성
```
internal/dto/request/${resource}_request.go
internal/dto/response/${resource}_response.go
```

### 3. Repository 생성
```
internal/repository/${resource}_repo.go
```
```go
type ${Resource}Repository interface {
    Create(ctx context.Context, entity *domain.${Resource}) error
    FindByID(ctx context.Context, companyID, id uuid.UUID) (*domain.${Resource}, error)
    FindAll(ctx context.Context, companyID uuid.UUID, filter *Filter) ([]*domain.${Resource}, error)
    Update(ctx context.Context, entity *domain.${Resource}) error
    Delete(ctx context.Context, companyID, id uuid.UUID) error
}
```

### 4. Service 생성
```
internal/service/${resource}_service.go
```
```go
type ${Resource}Service struct {
    repo   repository.${Resource}Repository
    logger *zap.Logger
}

func (s *${Resource}Service) Create(ctx context.Context, req *dto.Create${Resource}Request) (*dto.${Resource}Response, error) {
    companyID := ctx.Value("company_id").(uuid.UUID)
    entity := req.ToEntity(companyID)

    if err := s.repo.Create(ctx, entity); err != nil {
        return nil, fmt.Errorf("create ${resource}: %w", err)
    }

    return dto.${Resource}ResponseFrom(entity), nil
}
```

### 5. Handler 생성
```
internal/handler/${resource}_handler.go
```
```go
func (h *${Resource}Handler) Create(c *gin.Context) {
    var req dto.Create${Resource}Request
    if err := c.ShouldBindJSON(&req); err != nil {
        h.respondError(c, http.StatusBadRequest, err)
        return
    }

    resp, err := h.service.Create(c.Request.Context(), &req)
    if err != nil {
        h.respondError(c, http.StatusInternalServerError, err)
        return
    }

    c.JSON(http.StatusCreated, resp)
}
```

### 6. Router 등록
```
internal/handler/router.go
```
```go
// ${resource} routes
${resource}Group := v1.Group("/${resource}s")
${resource}Group.Use(middleware.Auth(), middleware.Tenant())
{
    ${resource}Group.POST("", ${resource}Handler.Create)
    ${resource}Group.GET("", ${resource}Handler.List)
    ${resource}Group.GET("/:id", ${resource}Handler.GetByID)
    ${resource}Group.PUT("/:id", ${resource}Handler.Update)
    ${resource}Handler.DELETE("/:id", ${resource}Handler.Delete)
}
```

## Output Structure
```
internal/
├── domain/${resource}.go
├── dto/
│   ├── request/${resource}_request.go
│   └── response/${resource}_response.go
├── repository/${resource}_repo.go
├── service/${resource}_service.go
└── handler/${resource}_handler.go
```

## Rules
- 모든 테이블에 `company_id` 필수
- Context에서 tenant 정보 추출
- 에러는 `fmt.Errorf("context: %w", err)` 형식
- UUID는 `github.com/google/uuid` 사용
