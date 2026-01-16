# Phase 3: Go Core API Architecture

## Overview

Phase 3는 K-ERP SaaS의 Go Backend Core를 구축합니다.
Clean Architecture 원칙을 따르며, 멀티테넌시와 확장성을 고려한 설계입니다.

---

## Directory Structure

```
internal/
├── config/                 # Configuration management
│   ├── config.go          # Main config struct
│   ├── loader.go          # Viper loader
│   └── validator.go       # Config validation
│
├── middleware/             # HTTP Middleware chain
│   ├── auth.go            # JWT authentication
│   ├── tenant.go          # Multi-tenant context injection
│   ├── logger.go          # Request/Response logging
│   ├── recovery.go        # Panic recovery
│   ├── cors.go            # CORS configuration
│   ├── ratelimit.go       # Rate limiting
│   └── requestid.go       # Request ID generation
│
├── handler/                # HTTP Handlers
│   ├── base.go            # Base handler with helpers
│   ├── response.go        # Standardized responses
│   ├── auth_handler.go    # Login, Logout, Refresh
│   ├── user_handler.go    # User CRUD
│   ├── company_handler.go # Company management
│   └── health_handler.go  # Health checks
│
├── router/                 # Route registration
│   ├── router.go          # Main router setup
│   ├── v1.go              # API v1 routes
│   └── middleware.go      # Middleware chain builder
│
├── database/               # Database connections
│   ├── postgres.go        # PostgreSQL + GORM
│   ├── redis.go           # Redis client
│   └── nats.go            # NATS JetStream
│
├── auth/                   # Authentication utilities
│   ├── jwt.go             # JWT generation/validation
│   ├── password.go        # Password hashing (bcrypt)
│   └── claims.go          # JWT claims struct
│
├── errors/                 # Error definitions
│   ├── errors.go          # Error types
│   └── codes.go           # Error codes
│
├── context/                # Context utilities
│   ├── keys.go            # Context keys
│   └── tenant.go          # Tenant context helpers
│
└── pkg/                    # Shared utilities
    ├── validator/         # Input validation
    ├── pagination/        # Pagination helpers
    └── response/          # Response builders

cmd/
├── api/
│   └── main.go            # API server entry
└── worker/
    └── main.go            # Background worker entry
```

---

## Configuration

### Environment Variables

```yaml
# config/app.yaml (or environment variables)
app:
  name: kerp-api
  env: development  # development, staging, production
  debug: true
  port: 8080

database:
  host: localhost
  port: 5432
  name: kerp_dev
  user: kerp
  password: ${DB_PASSWORD}
  sslmode: disable
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

redis:
  host: localhost
  port: 6379
  password: ${REDIS_PASSWORD}
  db: 0

nats:
  url: nats://localhost:4222
  cluster_id: kerp-cluster

jwt:
  secret: ${JWT_SECRET}
  access_token_ttl: 15m
  refresh_token_ttl: 7d
  issuer: kerp-api

cors:
  allowed_origins:
    - http://localhost:3000
    - https://erp.abada.kr
  allowed_methods:
    - GET
    - POST
    - PUT
    - DELETE
    - OPTIONS
  allowed_headers:
    - Authorization
    - Content-Type
    - X-Company-ID

ratelimit:
  enabled: true
  requests_per_second: 100
  burst: 200
```

### Config Struct

```go
// internal/config/config.go
type Config struct {
    App      AppConfig      `mapstructure:"app"`
    Database DatabaseConfig `mapstructure:"database"`
    Redis    RedisConfig    `mapstructure:"redis"`
    NATS     NATSConfig     `mapstructure:"nats"`
    JWT      JWTConfig      `mapstructure:"jwt"`
    CORS     CORSConfig     `mapstructure:"cors"`
    RateLimit RateLimitConfig `mapstructure:"ratelimit"`
}

type AppConfig struct {
    Name  string `mapstructure:"name"`
    Env   string `mapstructure:"env"`
    Debug bool   `mapstructure:"debug"`
    Port  int    `mapstructure:"port"`
}

type DatabaseConfig struct {
    Host            string        `mapstructure:"host"`
    Port            int           `mapstructure:"port"`
    Name            string        `mapstructure:"name"`
    User            string        `mapstructure:"user"`
    Password        string        `mapstructure:"password"`
    SSLMode         string        `mapstructure:"sslmode"`
    MaxOpenConns    int           `mapstructure:"max_open_conns"`
    MaxIdleConns    int           `mapstructure:"max_idle_conns"`
    ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}
```

---

## Middleware Chain

### Order (Top to Bottom)

```
Request
    │
    ▼
┌─────────────────────┐
│   Request ID        │  Generate unique request ID
└─────────────────────┘
    │
    ▼
┌─────────────────────┐
│   Logger            │  Log request start
└─────────────────────┘
    │
    ▼
┌─────────────────────┐
│   Recovery          │  Catch panics
└─────────────────────┘
    │
    ▼
┌─────────────────────┐
│   CORS              │  Handle CORS preflight
└─────────────────────┘
    │
    ▼
┌─────────────────────┐
│   Rate Limit        │  Request throttling
└─────────────────────┘
    │
    ▼
┌─────────────────────┐
│   Auth (JWT)        │  Validate token, set user
└─────────────────────┘
    │
    ▼
┌─────────────────────┐
│   Tenant            │  Extract & validate company_id
└─────────────────────┘
    │
    ▼
┌─────────────────────┐
│   Permission        │  Check RBAC permissions
└─────────────────────┘
    │
    ▼
   Handler
```

### Middleware Interfaces

```go
// internal/middleware/auth.go
func AuthMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractBearerToken(c)
        if token == "" {
            c.AbortWithStatusJSON(401, response.Error("UNAUTHORIZED", "Missing token"))
            return
        }

        claims, err := jwtService.ValidateToken(token)
        if err != nil {
            c.AbortWithStatusJSON(401, response.Error("INVALID_TOKEN", err.Error()))
            return
        }

        // Set user info in context
        c.Set(context.KeyUserID, claims.UserID)
        c.Set(context.KeyCompanyID, claims.CompanyID)
        c.Set(context.KeyRoles, claims.Roles)

        c.Next()
    }
}

// internal/middleware/tenant.go
func TenantMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        companyID := c.GetString(context.KeyCompanyID)
        if companyID == "" {
            c.AbortWithStatusJSON(400, response.Error("INVALID_TENANT", "Company ID required"))
            return
        }

        // Validate UUID format
        if _, err := uuid.Parse(companyID); err != nil {
            c.AbortWithStatusJSON(400, response.Error("INVALID_TENANT", "Invalid company ID"))
            return
        }

        c.Next()
    }
}
```

---

## Authentication Flow

### JWT Token Structure

```go
// internal/auth/claims.go
type Claims struct {
    jwt.RegisteredClaims
    UserID    uuid.UUID `json:"user_id"`
    CompanyID uuid.UUID `json:"company_id"`
    Email     string    `json:"email"`
    Name      string    `json:"name"`
    Roles     []string  `json:"roles"`
}
```

### Login Flow

```
1. POST /api/v1/auth/login
   └─ Body: { email, password }

2. Validate credentials
   └─ Check password hash (bcrypt)
   └─ Check account status (active, not locked)
   └─ Check failed login attempts

3. Generate tokens
   └─ Access Token (15min)
   └─ Refresh Token (7days, stored in DB)

4. Response
   └─ { access_token, refresh_token, expires_in, user }
```

### Refresh Flow

```
1. POST /api/v1/auth/refresh
   └─ Body: { refresh_token }

2. Validate refresh token
   └─ Check token exists in DB
   └─ Check not expired
   └─ Check not revoked

3. Generate new access token
   └─ Optionally rotate refresh token

4. Response
   └─ { access_token, refresh_token, expires_in }
```

---

## Response Format

### Success Response

```json
{
    "success": true,
    "data": { ... },
    "meta": {
        "request_id": "req_abc123",
        "timestamp": "2026-01-17T12:00:00Z"
    }
}
```

### Paginated Response

```json
{
    "success": true,
    "data": [ ... ],
    "meta": {
        "request_id": "req_abc123",
        "timestamp": "2026-01-17T12:00:00Z",
        "pagination": {
            "page": 1,
            "per_page": 20,
            "total": 100,
            "total_pages": 5
        }
    }
}
```

### Error Response

```json
{
    "success": false,
    "error": {
        "code": "VALIDATION_ERROR",
        "message": "Invalid input data",
        "details": [
            { "field": "email", "message": "Invalid email format" }
        ]
    },
    "meta": {
        "request_id": "req_abc123",
        "timestamp": "2026-01-17T12:00:00Z"
    }
}
```

### Response Builder

```go
// internal/handler/response.go
type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *ErrorBody  `json:"error,omitempty"`
    Meta    Meta        `json:"meta"`
}

type ErrorBody struct {
    Code    string        `json:"code"`
    Message string        `json:"message"`
    Details []FieldError  `json:"details,omitempty"`
}

type Meta struct {
    RequestID  string      `json:"request_id"`
    Timestamp  time.Time   `json:"timestamp"`
    Pagination *Pagination `json:"pagination,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Response{
        Success: true,
        Data:    data,
        Meta:    buildMeta(c),
    })
}

func Created(c *gin.Context, data interface{}) {
    c.JSON(http.StatusCreated, Response{
        Success: true,
        Data:    data,
        Meta:    buildMeta(c),
    })
}

func Error(c *gin.Context, status int, code, message string) {
    c.JSON(status, Response{
        Success: false,
        Error: &ErrorBody{
            Code:    code,
            Message: message,
        },
        Meta: buildMeta(c),
    })
}
```

---

## Error Codes

```go
// internal/errors/codes.go
const (
    // Auth errors (1xxx)
    ErrUnauthorized     = "AUTH_1001"  // Missing or invalid token
    ErrTokenExpired     = "AUTH_1002"  // Token expired
    ErrInvalidCredentials = "AUTH_1003" // Wrong email/password
    ErrAccountLocked    = "AUTH_1004"  // Account locked
    ErrAccountInactive  = "AUTH_1005"  // Account inactive

    // Validation errors (2xxx)
    ErrValidation       = "VAL_2001"   // General validation error
    ErrInvalidInput     = "VAL_2002"   // Invalid input format
    ErrMissingField     = "VAL_2003"   // Required field missing

    // Resource errors (3xxx)
    ErrNotFound         = "RES_3001"   // Resource not found
    ErrAlreadyExists    = "RES_3002"   // Resource already exists
    ErrConflict         = "RES_3003"   // Resource conflict

    // Permission errors (4xxx)
    ErrForbidden        = "PERM_4001"  // Access denied
    ErrInsufficientRole = "PERM_4002"  // Role not sufficient

    // Server errors (5xxx)
    ErrInternal         = "SRV_5001"   // Internal server error
    ErrDatabase         = "SRV_5002"   // Database error
    ErrExternalService  = "SRV_5003"   // External service error
)
```

---

## Router Organization

### API Routes Structure

```go
// internal/router/v1.go
func RegisterV1Routes(r *gin.RouterGroup, h *handler.Handlers, mw *middleware.Middlewares) {
    v1 := r.Group("/v1")

    // Public routes (no auth required)
    public := v1.Group("")
    {
        public.POST("/auth/login", h.Auth.Login)
        public.POST("/auth/register", h.Auth.Register)
        public.POST("/auth/forgot-password", h.Auth.ForgotPassword)
    }

    // Authenticated routes
    auth := v1.Group("")
    auth.Use(mw.Auth)
    {
        // Auth management
        auth.POST("/auth/refresh", h.Auth.Refresh)
        auth.POST("/auth/logout", h.Auth.Logout)
        auth.GET("/auth/me", h.Auth.Me)
        auth.PUT("/auth/password", h.Auth.ChangePassword)

        // Tenant-scoped routes
        tenant := auth.Group("")
        tenant.Use(mw.Tenant)
        {
            // Users
            users := tenant.Group("/users")
            {
                users.GET("", h.User.List)
                users.POST("", h.User.Create)
                users.GET("/:id", h.User.Get)
                users.PUT("/:id", h.User.Update)
                users.DELETE("/:id", h.User.Delete)
            }

            // Roles
            roles := tenant.Group("/roles")
            {
                roles.GET("", h.Role.List)
                roles.POST("", h.Role.Create)
                roles.GET("/:id", h.Role.Get)
                roles.PUT("/:id", h.Role.Update)
                roles.DELETE("/:id", h.Role.Delete)
            }

            // Company
            company := tenant.Group("/company")
            {
                company.GET("", h.Company.Get)
                company.PUT("", h.Company.Update)
            }

            // Partners
            partners := tenant.Group("/partners")
            {
                partners.GET("", h.Partner.List)
                partners.POST("", h.Partner.Create)
                partners.GET("/:id", h.Partner.Get)
                partners.PUT("/:id", h.Partner.Update)
                partners.DELETE("/:id", h.Partner.Delete)
            }
        }
    }
}
```

---

## Database Connection

### GORM Setup

```go
// internal/database/postgres.go
func NewPostgresDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
    )

    db, err := gorm.Open(postgres.New(postgres.Config{
        DSN:                  dsn,
        PreferSimpleProtocol: true,
    }), &gorm.Config{
        Logger: newGormLogger(),
        NamingStrategy: schema.NamingStrategy{
            SingularTable: true,
        },
    })
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    sqlDB, err := db.DB()
    if err != nil {
        return nil, fmt.Errorf("failed to get sql.DB: %w", err)
    }

    sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
    sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
    sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

    return db, nil
}

// Scoped DB for multi-tenancy
func ScopedDB(db *gorm.DB, companyID uuid.UUID) *gorm.DB {
    return db.Where("company_id = ?", companyID)
}
```

---

## Handler Pattern

### Base Handler

```go
// internal/handler/base.go
type BaseHandler struct {
    db     *gorm.DB
    redis  *redis.Client
    logger *zap.Logger
}

func (h *BaseHandler) GetUserID(c *gin.Context) uuid.UUID {
    id, _ := c.Get(context.KeyUserID)
    return id.(uuid.UUID)
}

func (h *BaseHandler) GetCompanyID(c *gin.Context) uuid.UUID {
    id, _ := c.Get(context.KeyCompanyID)
    return id.(uuid.UUID)
}

func (h *BaseHandler) ScopedDB(c *gin.Context) *gorm.DB {
    return database.ScopedDB(h.db, h.GetCompanyID(c))
}

func (h *BaseHandler) BindJSON(c *gin.Context, obj interface{}) error {
    if err := c.ShouldBindJSON(obj); err != nil {
        response.ValidationError(c, err)
        return err
    }
    return nil
}
```

### Example Handler

```go
// internal/handler/user_handler.go
type UserHandler struct {
    BaseHandler
    userService service.UserService
}

func (h *UserHandler) List(c *gin.Context) {
    companyID := h.GetCompanyID(c)

    // Parse pagination
    page, perPage := pagination.FromQuery(c)

    users, total, err := h.userService.List(c.Request.Context(), companyID, page, perPage)
    if err != nil {
        response.Error(c, 500, errors.ErrInternal, err.Error())
        return
    }

    response.Paginated(c, users, page, perPage, total)
}

func (h *UserHandler) Create(c *gin.Context) {
    var req dto.CreateUserRequest
    if err := h.BindJSON(c, &req); err != nil {
        return
    }

    companyID := h.GetCompanyID(c)
    createdBy := h.GetUserID(c)

    user, err := h.userService.Create(c.Request.Context(), companyID, createdBy, &req)
    if err != nil {
        // Handle specific errors
        switch {
        case errors.Is(err, service.ErrEmailExists):
            response.Error(c, 409, errors.ErrAlreadyExists, "Email already exists")
        default:
            response.Error(c, 500, errors.ErrInternal, err.Error())
        }
        return
    }

    response.Created(c, user)
}
```

---

## Phase 3 vs Phase 4 Boundary

### Phase 3 Scope (This Phase)

```
- cmd/api/main.go           # Entry point
- internal/config/          # Configuration
- internal/middleware/      # All middleware
- internal/handler/         # HTTP handlers (interface layer)
- internal/router/          # Route registration
- internal/database/        # DB connections
- internal/auth/            # JWT, password utilities
- internal/errors/          # Error definitions
- internal/context/         # Context helpers
- internal/pkg/             # Shared utilities
```

### Phase 4 Scope (Next Phase)

```
- internal/domain/          # Domain models (entities)
- internal/dto/             # Data transfer objects
- internal/service/         # Business logic
- internal/repository/      # Data access layer
```

### Interface Contract

Phase 3 handlers will use interfaces defined for Phase 4:

```go
// internal/service/interfaces.go (Phase 3 defines, Phase 4 implements)
type UserService interface {
    List(ctx context.Context, companyID uuid.UUID, page, perPage int) ([]dto.UserResponse, int64, error)
    Get(ctx context.Context, companyID, userID uuid.UUID) (*dto.UserResponse, error)
    Create(ctx context.Context, companyID, createdBy uuid.UUID, req *dto.CreateUserRequest) (*dto.UserResponse, error)
    Update(ctx context.Context, companyID, userID uuid.UUID, req *dto.UpdateUserRequest) (*dto.UserResponse, error)
    Delete(ctx context.Context, companyID, userID uuid.UUID) error
}
```

---

## Dependency Injection

### Application Container

```go
// internal/app/app.go
type App struct {
    Config    *config.Config
    DB        *gorm.DB
    Redis     *redis.Client
    NATS      *nats.Conn
    Logger    *zap.Logger
    JWTService *auth.JWTService

    // Services (Phase 4 will implement)
    Services *Services

    // Handlers
    Handlers *Handlers

    // Middleware
    Middleware *Middleware
}

func NewApp(cfg *config.Config) (*App, error) {
    // Initialize logger
    logger, _ := zap.NewProduction()

    // Initialize database
    db, err := database.NewPostgresDB(&cfg.Database)
    if err != nil {
        return nil, err
    }

    // Initialize Redis
    rdb := database.NewRedisClient(&cfg.Redis)

    // Initialize NATS
    nc, err := database.NewNATSConnection(&cfg.NATS)
    if err != nil {
        return nil, err
    }

    // Initialize JWT service
    jwtService := auth.NewJWTService(&cfg.JWT)

    app := &App{
        Config:     cfg,
        DB:         db,
        Redis:      rdb,
        NATS:       nc,
        Logger:     logger,
        JWTService: jwtService,
    }

    // Initialize handlers
    app.Handlers = NewHandlers(app)

    // Initialize middleware
    app.Middleware = NewMiddleware(app)

    return app, nil
}
```

---

## Implementation Order

1. **config/** - Configuration loading (Viper)
2. **database/** - PostgreSQL, Redis, NATS connections
3. **auth/** - JWT service, password hashing
4. **errors/** - Error types and codes
5. **context/** - Context keys and helpers
6. **middleware/** - Auth, Tenant, Logger, etc.
7. **handler/response.go** - Response builders
8. **handler/base.go** - Base handler
9. **router/** - Route registration
10. **cmd/api/main.go** - Wire everything together

---

## Testing Strategy

```go
// Phase 3 tests focus on:
// 1. Middleware unit tests
// 2. Handler integration tests (with mock services)
// 3. Router tests (route matching)

// internal/middleware/auth_test.go
func TestAuthMiddleware_ValidToken(t *testing.T) {
    // Setup
    jwtService := auth.NewJWTService(&config.JWTConfig{Secret: "test"})
    token, _ := jwtService.GenerateAccessToken(claims)

    // Create request
    req := httptest.NewRequest("GET", "/test", nil)
    req.Header.Set("Authorization", "Bearer "+token)

    // Execute
    w := httptest.NewRecorder()
    router := gin.New()
    router.Use(middleware.AuthMiddleware(jwtService))
    router.GET("/test", func(c *gin.Context) {
        c.JSON(200, gin.H{"ok": true})
    })
    router.ServeHTTP(w, req)

    // Assert
    assert.Equal(t, 200, w.Code)
}
```

---

## Summary

Phase 3 구현 완료 시:

- [x] Clean Architecture 기반 구조
- [x] 환경별 설정 관리
- [x] JWT 기반 인증
- [x] 멀티테넌시 미들웨어
- [x] 표준화된 응답 포맷
- [x] 에러 코드 체계
- [x] PostgreSQL + Redis + NATS 연결
- [ ] Phase 4 서비스 인터페이스 정의
