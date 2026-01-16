# Phase 3: Implementation Plan

## Overview

총 10개 단계로 구성된 구현 계획입니다.
각 단계는 독립적으로 테스트 가능하며, 순차적으로 진행됩니다.

---

## Step 1: Configuration (config/)

### Files to Create

```
internal/config/
├── config.go      # Config structs
├── loader.go      # Viper loader
└── validator.go   # Config validation

config/
└── app.yaml       # Default config (gitignored for secrets)
    app.example.yaml # Example config (committed)
```

### config.go

```go
package config

import "time"

type Config struct {
    App       AppConfig       `mapstructure:"app"`
    Database  DatabaseConfig  `mapstructure:"database"`
    Redis     RedisConfig     `mapstructure:"redis"`
    NATS      NATSConfig      `mapstructure:"nats"`
    JWT       JWTConfig       `mapstructure:"jwt"`
    CORS      CORSConfig      `mapstructure:"cors"`
    RateLimit RateLimitConfig `mapstructure:"ratelimit"`
    Log       LogConfig       `mapstructure:"log"`
}

type AppConfig struct {
    Name    string `mapstructure:"name"`
    Env     string `mapstructure:"env"`
    Debug   bool   `mapstructure:"debug"`
    Port    int    `mapstructure:"port"`
    Version string `mapstructure:"version"`
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

type RedisConfig struct {
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    Password string `mapstructure:"password"`
    DB       int    `mapstructure:"db"`
}

type NATSConfig struct {
    URL       string `mapstructure:"url"`
    ClusterID string `mapstructure:"cluster_id"`
}

type JWTConfig struct {
    Secret          string        `mapstructure:"secret"`
    AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
    RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
    Issuer          string        `mapstructure:"issuer"`
}

type CORSConfig struct {
    AllowedOrigins []string `mapstructure:"allowed_origins"`
    AllowedMethods []string `mapstructure:"allowed_methods"`
    AllowedHeaders []string `mapstructure:"allowed_headers"`
}

type RateLimitConfig struct {
    Enabled           bool `mapstructure:"enabled"`
    RequestsPerSecond int  `mapstructure:"requests_per_second"`
    Burst             int  `mapstructure:"burst"`
}

type LogConfig struct {
    Level  string `mapstructure:"level"`
    Format string `mapstructure:"format"` // json, console
}
```

### loader.go

```go
package config

import (
    "fmt"
    "strings"

    "github.com/spf13/viper"
)

func Load() (*Config, error) {
    v := viper.New()

    // Config file
    v.SetConfigName("app")
    v.SetConfigType("yaml")
    v.AddConfigPath("./config")
    v.AddConfigPath(".")

    // Environment variables
    v.SetEnvPrefix("KERP")
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    v.AutomaticEnv()

    // Defaults
    setDefaults(v)

    // Read config file (optional)
    if err := v.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("failed to read config: %w", err)
        }
    }

    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    return &cfg, nil
}

func setDefaults(v *viper.Viper) {
    // App
    v.SetDefault("app.name", "kerp-api")
    v.SetDefault("app.env", "development")
    v.SetDefault("app.debug", true)
    v.SetDefault("app.port", 8080)
    v.SetDefault("app.version", "0.2.0")

    // Database
    v.SetDefault("database.host", "localhost")
    v.SetDefault("database.port", 5432)
    v.SetDefault("database.name", "kerp_dev")
    v.SetDefault("database.user", "kerp")
    v.SetDefault("database.sslmode", "disable")
    v.SetDefault("database.max_open_conns", 25)
    v.SetDefault("database.max_idle_conns", 5)
    v.SetDefault("database.conn_max_lifetime", "5m")

    // Redis
    v.SetDefault("redis.host", "localhost")
    v.SetDefault("redis.port", 6379)
    v.SetDefault("redis.db", 0)

    // NATS
    v.SetDefault("nats.url", "nats://localhost:4222")
    v.SetDefault("nats.cluster_id", "kerp-cluster")

    // JWT
    v.SetDefault("jwt.access_token_ttl", "15m")
    v.SetDefault("jwt.refresh_token_ttl", "168h") // 7 days
    v.SetDefault("jwt.issuer", "kerp-api")

    // CORS
    v.SetDefault("cors.allowed_origins", []string{"http://localhost:3000"})
    v.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
    v.SetDefault("cors.allowed_headers", []string{"Authorization", "Content-Type"})

    // Rate limit
    v.SetDefault("ratelimit.enabled", false)
    v.SetDefault("ratelimit.requests_per_second", 100)
    v.SetDefault("ratelimit.burst", 200)

    // Log
    v.SetDefault("log.level", "info")
    v.SetDefault("log.format", "json")
}
```

### Verification

```bash
# Test config loading
go test ./internal/config/...
```

---

## Step 2: Database Connections (database/)

### Files to Create

```
internal/database/
├── postgres.go    # GORM + pgx
├── redis.go       # Redis client
├── nats.go        # NATS JetStream
└── logger.go      # GORM logger adapter
```

### postgres.go (Key Parts)

```go
package database

import (
    "fmt"

    "github.com/google/uuid"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"

    "github.com/saintgo7/saas-kerp/internal/config"
)

func NewPostgresDB(cfg *config.DatabaseConfig, logger *gorm.Logger) (*gorm.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
    )

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    sqlDB, _ := db.DB()
    sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
    sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
    sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

    return db, nil
}

// ScopedDB returns a DB instance scoped to a company (multi-tenancy)
func ScopedDB(db *gorm.DB, companyID uuid.UUID) *gorm.DB {
    return db.Where("company_id = ?", companyID)
}

// Close closes the database connection
func Close(db *gorm.DB) error {
    sqlDB, err := db.DB()
    if err != nil {
        return err
    }
    return sqlDB.Close()
}
```

---

## Step 3: Error Handling (errors/)

### Files to Create

```
internal/errors/
├── errors.go      # Custom error types
└── codes.go       # Error codes
```

### errors.go

```go
package errors

import (
    "errors"
    "fmt"
)

type AppError struct {
    Code    string
    Message string
    Err     error
}

func (e *AppError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
    return e.Err
}

func New(code, message string) *AppError {
    return &AppError{Code: code, Message: message}
}

func Wrap(code, message string, err error) *AppError {
    return &AppError{Code: code, Message: message, Err: err}
}

// Predefined errors
var (
    ErrUnauthorized      = New(CodeUnauthorized, "Unauthorized")
    ErrForbidden         = New(CodeForbidden, "Forbidden")
    ErrNotFound          = New(CodeNotFound, "Resource not found")
    ErrValidation        = New(CodeValidation, "Validation failed")
    ErrInternal          = New(CodeInternal, "Internal server error")
    ErrEmailExists       = New(CodeEmailExists, "Email already exists")
    ErrInvalidCredentials = New(CodeInvalidCredentials, "Invalid email or password")
    ErrAccountLocked     = New(CodeAccountLocked, "Account is locked")
)

func Is(err error, target *AppError) bool {
    var appErr *AppError
    if errors.As(err, &appErr) {
        return appErr.Code == target.Code
    }
    return false
}
```

### codes.go

```go
package errors

const (
    // Authentication (AUTH_)
    CodeUnauthorized       = "AUTH_001"
    CodeTokenExpired       = "AUTH_002"
    CodeInvalidCredentials = "AUTH_003"
    CodeAccountLocked      = "AUTH_004"
    CodeAccountInactive    = "AUTH_005"
    CodeTokenInvalid       = "AUTH_006"

    // Validation (VAL_)
    CodeValidation   = "VAL_001"
    CodeInvalidInput = "VAL_002"
    CodeMissingField = "VAL_003"

    // Resource (RES_)
    CodeNotFound      = "RES_001"
    CodeAlreadyExists = "RES_002"
    CodeConflict      = "RES_003"
    CodeEmailExists   = "RES_004"

    // Permission (PERM_)
    CodeForbidden        = "PERM_001"
    CodeInsufficientRole = "PERM_002"

    // Server (SRV_)
    CodeInternal        = "SRV_001"
    CodeDatabase        = "SRV_002"
    CodeExternalService = "SRV_003"
)
```

---

## Step 4: Context & Keys (context/)

### Files to Create

```
internal/context/
├── keys.go        # Context key constants
└── helpers.go     # Context helper functions
```

### keys.go

```go
package context

const (
    KeyRequestID = "request_id"
    KeyUserID    = "user_id"
    KeyCompanyID = "company_id"
    KeyEmail     = "email"
    KeyRoles     = "roles"
    KeyLogger    = "logger"
)
```

### helpers.go

```go
package context

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

func GetRequestID(c *gin.Context) string {
    if v, exists := c.Get(KeyRequestID); exists {
        return v.(string)
    }
    return ""
}

func GetUserID(c *gin.Context) uuid.UUID {
    if v, exists := c.Get(KeyUserID); exists {
        return v.(uuid.UUID)
    }
    return uuid.Nil
}

func GetCompanyID(c *gin.Context) uuid.UUID {
    if v, exists := c.Get(KeyCompanyID); exists {
        return v.(uuid.UUID)
    }
    return uuid.Nil
}

func GetRoles(c *gin.Context) []string {
    if v, exists := c.Get(KeyRoles); exists {
        return v.([]string)
    }
    return nil
}

func HasRole(c *gin.Context, role string) bool {
    roles := GetRoles(c)
    for _, r := range roles {
        if r == role {
            return true
        }
    }
    return false
}
```

---

## Step 5: Authentication (auth/)

### Files to Create

```
internal/auth/
├── jwt.go         # JWT service
├── claims.go      # JWT claims
└── password.go    # Password hashing
```

### jwt.go

```go
package auth

import (
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"

    "github.com/saintgo7/saas-kerp/internal/config"
)

type JWTService struct {
    secret          []byte
    accessTokenTTL  time.Duration
    refreshTokenTTL time.Duration
    issuer          string
}

func NewJWTService(cfg *config.JWTConfig) *JWTService {
    return &JWTService{
        secret:          []byte(cfg.Secret),
        accessTokenTTL:  cfg.AccessTokenTTL,
        refreshTokenTTL: cfg.RefreshTokenTTL,
        issuer:          cfg.Issuer,
    }
}

func (s *JWTService) GenerateAccessToken(claims *Claims) (string, error) {
    claims.RegisteredClaims = jwt.RegisteredClaims{
        Issuer:    s.issuer,
        IssuedAt:  jwt.NewNumericDate(time.Now()),
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenTTL)),
        ID:        uuid.New().String(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.secret)
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return s.secret, nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }

    return nil, fmt.Errorf("invalid token")
}

func (s *JWTService) GenerateRefreshToken() string {
    return uuid.New().String()
}

func (s *JWTService) GetRefreshTokenTTL() time.Duration {
    return s.refreshTokenTTL
}
```

### claims.go

```go
package auth

import (
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

type Claims struct {
    jwt.RegisteredClaims
    UserID    uuid.UUID `json:"user_id"`
    CompanyID uuid.UUID `json:"company_id"`
    Email     string    `json:"email"`
    Name      string    `json:"name"`
    Roles     []string  `json:"roles"`
}
```

### password.go

```go
package auth

import (
    "golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
    return string(bytes), err
}

func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

---

## Step 6: Middleware (middleware/)

### Files to Create

```
internal/middleware/
├── requestid.go   # Request ID generation
├── logger.go      # Request logging
├── recovery.go    # Panic recovery
├── cors.go        # CORS handling
├── auth.go        # JWT authentication
├── tenant.go      # Multi-tenant context
├── ratelimit.go   # Rate limiting
└── chain.go       # Middleware chain builder
```

### Key Middleware Examples

**requestid.go**
```go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    appctx "github.com/saintgo7/saas-kerp/internal/context"
)

func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        c.Set(appctx.KeyRequestID, requestID)
        c.Header("X-Request-ID", requestID)
        c.Next()
    }
}
```

**auth.go**
```go
package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"

    "github.com/saintgo7/saas-kerp/internal/auth"
    appctx "github.com/saintgo7/saas-kerp/internal/context"
    "github.com/saintgo7/saas-kerp/internal/handler/response"
)

func Auth(jwtService *auth.JWTService) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            response.Unauthorized(c, "Missing authorization header")
            c.Abort()
            return
        }

        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            response.Unauthorized(c, "Invalid authorization format")
            c.Abort()
            return
        }

        claims, err := jwtService.ValidateToken(parts[1])
        if err != nil {
            response.Unauthorized(c, "Invalid or expired token")
            c.Abort()
            return
        }

        c.Set(appctx.KeyUserID, claims.UserID)
        c.Set(appctx.KeyCompanyID, claims.CompanyID)
        c.Set(appctx.KeyEmail, claims.Email)
        c.Set(appctx.KeyRoles, claims.Roles)

        c.Next()
    }
}
```

**tenant.go**
```go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    appctx "github.com/saintgo7/saas-kerp/internal/context"
    "github.com/saintgo7/saas-kerp/internal/handler/response"
)

func Tenant() gin.HandlerFunc {
    return func(c *gin.Context) {
        companyID := appctx.GetCompanyID(c)
        if companyID == uuid.Nil {
            response.BadRequest(c, "Company context required")
            c.Abort()
            return
        }
        c.Next()
    }
}
```

---

## Step 7: Response Handlers (handler/response.go)

### Files to Create

```
internal/handler/
├── response/
│   ├── response.go    # Response structs
│   └── helpers.go     # Response helpers
```

### response.go

```go
package response

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"

    appctx "github.com/saintgo7/saas-kerp/internal/context"
)

type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *ErrorBody  `json:"error,omitempty"`
    Meta    Meta        `json:"meta"`
}

type ErrorBody struct {
    Code    string       `json:"code"`
    Message string       `json:"message"`
    Details []FieldError `json:"details,omitempty"`
}

type FieldError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

type Meta struct {
    RequestID  string      `json:"request_id"`
    Timestamp  time.Time   `json:"timestamp"`
    Pagination *Pagination `json:"pagination,omitempty"`
}

type Pagination struct {
    Page       int   `json:"page"`
    PerPage    int   `json:"per_page"`
    Total      int64 `json:"total"`
    TotalPages int   `json:"total_pages"`
}

func buildMeta(c *gin.Context) Meta {
    return Meta{
        RequestID: appctx.GetRequestID(c),
        Timestamp: time.Now().UTC(),
    }
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

func NoContent(c *gin.Context) {
    c.Status(http.StatusNoContent)
}

func Paginated(c *gin.Context, data interface{}, page, perPage int, total int64) {
    totalPages := int(total) / perPage
    if int(total)%perPage > 0 {
        totalPages++
    }

    meta := buildMeta(c)
    meta.Pagination = &Pagination{
        Page:       page,
        PerPage:    perPage,
        Total:      total,
        TotalPages: totalPages,
    }

    c.JSON(http.StatusOK, Response{
        Success: true,
        Data:    data,
        Meta:    meta,
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

func BadRequest(c *gin.Context, message string) {
    Error(c, http.StatusBadRequest, "VAL_001", message)
}

func Unauthorized(c *gin.Context, message string) {
    Error(c, http.StatusUnauthorized, "AUTH_001", message)
}

func Forbidden(c *gin.Context, message string) {
    Error(c, http.StatusForbidden, "PERM_001", message)
}

func NotFound(c *gin.Context, message string) {
    Error(c, http.StatusNotFound, "RES_001", message)
}

func InternalError(c *gin.Context, message string) {
    Error(c, http.StatusInternalServerError, "SRV_001", message)
}

func ValidationError(c *gin.Context, details []FieldError) {
    c.JSON(http.StatusBadRequest, Response{
        Success: false,
        Error: &ErrorBody{
            Code:    "VAL_001",
            Message: "Validation failed",
            Details: details,
        },
        Meta: buildMeta(c),
    })
}
```

---

## Step 8: Base Handler (handler/base.go)

```go
package handler

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "go.uber.org/zap"
    "gorm.io/gorm"

    appctx "github.com/saintgo7/saas-kerp/internal/context"
    "github.com/saintgo7/saas-kerp/internal/database"
    "github.com/saintgo7/saas-kerp/internal/handler/response"
)

type BaseHandler struct {
    DB     *gorm.DB
    Logger *zap.Logger
}

func (h *BaseHandler) GetUserID(c *gin.Context) uuid.UUID {
    return appctx.GetUserID(c)
}

func (h *BaseHandler) GetCompanyID(c *gin.Context) uuid.UUID {
    return appctx.GetCompanyID(c)
}

func (h *BaseHandler) ScopedDB(c *gin.Context) *gorm.DB {
    return database.ScopedDB(h.DB, h.GetCompanyID(c))
}

func (h *BaseHandler) BindJSON(c *gin.Context, obj interface{}) bool {
    if err := c.ShouldBindJSON(obj); err != nil {
        response.BadRequest(c, err.Error())
        return false
    }
    return true
}

func (h *BaseHandler) ParseUUID(c *gin.Context, param string) (uuid.UUID, bool) {
    id, err := uuid.Parse(c.Param(param))
    if err != nil {
        response.BadRequest(c, "Invalid UUID format")
        return uuid.Nil, false
    }
    return id, true
}
```

---

## Step 9: Router (router/)

### Files to Create

```
internal/router/
├── router.go      # Main router setup
├── v1.go          # API v1 routes
└── health.go      # Health check routes
```

### router.go

```go
package router

import (
    "github.com/gin-gonic/gin"

    "github.com/saintgo7/saas-kerp/internal/auth"
    "github.com/saintgo7/saas-kerp/internal/config"
    "github.com/saintgo7/saas-kerp/internal/handler"
    "github.com/saintgo7/saas-kerp/internal/middleware"
)

type Router struct {
    engine     *gin.Engine
    config     *config.Config
    jwtService *auth.JWTService
    handlers   *handler.Handlers
}

func New(cfg *config.Config, jwtService *auth.JWTService, handlers *handler.Handlers) *Router {
    if cfg.App.Env == "production" {
        gin.SetMode(gin.ReleaseMode)
    }

    engine := gin.New()

    r := &Router{
        engine:     engine,
        config:     cfg,
        jwtService: jwtService,
        handlers:   handlers,
    }

    r.setupMiddleware()
    r.setupRoutes()

    return r
}

func (r *Router) setupMiddleware() {
    r.engine.Use(middleware.RequestID())
    r.engine.Use(middleware.Logger())
    r.engine.Use(middleware.Recovery())
    r.engine.Use(middleware.CORS(&r.config.CORS))

    if r.config.RateLimit.Enabled {
        r.engine.Use(middleware.RateLimit(&r.config.RateLimit))
    }
}

func (r *Router) setupRoutes() {
    // Health checks (no auth)
    r.engine.GET("/health", r.handlers.Health.Check)
    r.engine.GET("/health/ready", r.handlers.Health.Ready)
    r.engine.GET("/health/live", r.handlers.Health.Live)

    // API routes
    api := r.engine.Group("/api")
    RegisterV1Routes(api, r.jwtService, r.handlers)
}

func (r *Router) Engine() *gin.Engine {
    return r.engine
}
```

### v1.go

```go
package router

import (
    "github.com/gin-gonic/gin"

    "github.com/saintgo7/saas-kerp/internal/auth"
    "github.com/saintgo7/saas-kerp/internal/handler"
    "github.com/saintgo7/saas-kerp/internal/middleware"
)

func RegisterV1Routes(api *gin.RouterGroup, jwtService *auth.JWTService, h *handler.Handlers) {
    v1 := api.Group("/v1")

    // Public routes
    public := v1.Group("")
    {
        public.POST("/auth/login", h.Auth.Login)
        public.POST("/auth/register", h.Auth.Register)
        public.POST("/auth/forgot-password", h.Auth.ForgotPassword)
    }

    // Authenticated routes
    authed := v1.Group("")
    authed.Use(middleware.Auth(jwtService))
    {
        // Auth
        authed.POST("/auth/refresh", h.Auth.Refresh)
        authed.POST("/auth/logout", h.Auth.Logout)
        authed.GET("/auth/me", h.Auth.Me)
        authed.PUT("/auth/password", h.Auth.ChangePassword)

        // Tenant-scoped routes
        tenant := authed.Group("")
        tenant.Use(middleware.Tenant())
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

## Step 10: Main Entry Point (cmd/api/main.go)

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "go.uber.org/zap"

    "github.com/saintgo7/saas-kerp/internal/auth"
    "github.com/saintgo7/saas-kerp/internal/config"
    "github.com/saintgo7/saas-kerp/internal/database"
    "github.com/saintgo7/saas-kerp/internal/handler"
    "github.com/saintgo7/saas-kerp/internal/router"
)

func main() {
    // Load config
    cfg, err := config.Load()
    if err != nil {
        fmt.Printf("Failed to load config: %v\n", err)
        os.Exit(1)
    }

    // Initialize logger
    var logger *zap.Logger
    if cfg.App.Debug {
        logger, _ = zap.NewDevelopment()
    } else {
        logger, _ = zap.NewProduction()
    }
    defer logger.Sync()

    // Initialize database
    db, err := database.NewPostgresDB(&cfg.Database, nil)
    if err != nil {
        logger.Fatal("Failed to connect to database", zap.Error(err))
    }
    defer database.Close(db)

    // Initialize Redis
    rdb := database.NewRedisClient(&cfg.Redis)
    defer rdb.Close()

    // Initialize JWT service
    jwtService := auth.NewJWTService(&cfg.JWT)

    // Initialize handlers
    handlers := handler.NewHandlers(db, rdb, logger)

    // Initialize router
    r := router.New(cfg, jwtService, handlers)

    // Create server
    srv := &http.Server{
        Addr:         fmt.Sprintf(":%d", cfg.App.Port),
        Handler:      r.Engine(),
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Start server
    go func() {
        logger.Info("Starting server",
            zap.String("name", cfg.App.Name),
            zap.String("env", cfg.App.Env),
            zap.Int("port", cfg.App.Port),
        )
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("Failed to start server", zap.Error(err))
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        logger.Fatal("Server forced to shutdown", zap.Error(err))
    }

    logger.Info("Server exited")
}
```

---

## Verification Checklist

각 단계 완료 후 확인:

- [ ] Step 1: `go run ./cmd/api` - Config loads without error
- [ ] Step 2: DB connection test passes
- [ ] Step 3: Error types compile
- [ ] Step 4: Context helpers work
- [ ] Step 5: JWT generation/validation works
- [ ] Step 6: Middleware chain functions
- [ ] Step 7: Response helpers work
- [ ] Step 8: Base handler compiles
- [ ] Step 9: Routes register correctly
- [ ] Step 10: Server starts and responds to /health

---

## Dependencies to Add

```bash
go get github.com/golang-jwt/jwt/v5
go get github.com/ulule/limiter/v3
```

Updated go.mod:
```go
require (
    github.com/gin-gonic/gin v1.9.1
    github.com/golang-jwt/jwt/v5 v5.2.0
    github.com/google/uuid v1.6.0
    github.com/jackc/pgx/v5 v5.5.5
    github.com/nats-io/nats.go v1.33.1
    github.com/redis/go-redis/v9 v9.5.1
    github.com/spf13/viper v1.18.2
    github.com/ulule/limiter/v3 v3.11.2
    go.uber.org/zap v1.27.0
    golang.org/x/crypto v0.21.0
    gorm.io/driver/postgres v1.5.7
    gorm.io/gorm v1.25.8
)
```
