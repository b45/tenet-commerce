package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	"github.com/b45/tenet-commerce/backend/internal/entitlement"
	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/b45/tenet-commerce/backend/internal/manager"
	"github.com/b45/tenet-commerce/backend/internal/pos"
	"github.com/b45/tenet-commerce/backend/internal/supplychain"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/database"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	pkgRedis "github.com/b45/tenet-commerce/backend/pkg/redis"
	"github.com/b45/tenet-commerce/backend/pkg/response"
)

// RouterConfig holds all domain handlers and infrastructure dependencies required by the API router
type RouterConfig struct {
	AuthHandler        *internalAuth.Handler
	POSHandler         *pos.Handler
	SupplyChainHandler *supplychain.Handler
	LedgerHandler      *ledger.Handler
	ManagerHandler     *manager.Handler
	EntitlementHandler *entitlement.Handler
	EntitlementService *entitlement.Service
	TenantRepo         *tenant.Repository
	JWTService         *pkgAuth.JWTService
	RedisClient        *pkgRedis.Client
	PostgresDB         *database.PostgresDB
}

// SetupRouter constructs the Gin HTTP engine with global middlewares and modular domain route groups
func SetupRouter(cfg RouterConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("APP_DEBUG") == "true" {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// Disable trusting all proxies by default to resolve security warning
	_ = router.SetTrustedProxies(nil)

	// Mount Global Observability & Recovery Middlewares
	router.Use(logger.RealIPMiddleware())    // 1. Resolve real client IP behind proxies
	router.Use(logger.TraceMiddleware())     // 2. Distributed Tracing (trace_id, span_id)
	router.Use(logger.AccessLogMiddleware()) // 3. Structured JSON Access Logging
	router.Use(logger.RecoveryMiddleware())  // 4. Panic Recovery with stack trace logging
	router.Use(maintenanceMiddleware())

	// Standard JSON 404 and 405 error responses for all undefined endpoints
	router.NoRoute(func(c *gin.Context) {
		response.NotFound(c, "ROUTE_NOT_FOUND", "Endpoint not found: "+c.Request.Method+" "+c.Request.URL.Path)
	})
	router.NoMethod(func(c *gin.Context) {
		response.MethodNotAllowed(c, "METHOD_NOT_ALLOWED", "Method "+c.Request.Method+" not allowed for "+c.Request.URL.Path)
	})

	// Health Check Endpoint (Unauthenticated)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"app":    "tenet-commerce",
		})
	})
	router.GET("/ready", func(c *gin.Context) {
		if maintenanceModeEnabled() {
			c.Header("Retry-After", "300")
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "maintenance"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 750*time.Millisecond)
		defer cancel()
		if cfg.PostgresDB == nil || cfg.PostgresDB.Pool == nil || cfg.PostgresDB.Pool.Ping(ctx) != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		if cfg.RedisClient == nil || cfg.RedisClient.RDB == nil || cfg.RedisClient.RDB.Ping(ctx).Err() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// API v1 Namespace (Central Route Manifest)
	apiV1 := router.Group("/api/v1")
	{
		// =====================================================================
		// 1. PUBLIC ZONE (Unauthenticated Identity Endpoints)
		// =====================================================================
		authPublic := apiV1.Group("/auth")
		cfg.AuthHandler.RegisterPublicRoutes(authPublic)

		// =====================================================================
		// 2. PROTECTED ZONE (JWT Security + Multi-Tenant Schema Isolation)
		// =====================================================================
		protected := apiV1.Group("")
		protected.Use(
			internalAuth.JWTAuthMiddleware(cfg.JWTService, cfg.RedisClient),
			tenant.ContextMiddleware(cfg.PostgresDB, cfg.TenantRepo),
		)

		// Identity & Self-Profile Introspection
		cfg.AuthHandler.RegisterProtectedRoutes(protected.Group("/auth"))

		// Capability Introspection for Client Feature Gating
		if cfg.EntitlementHandler != nil {
			cfg.EntitlementHandler.RegisterRoutes(protected)
		}

		// Core Domain 1: Point of Sale & Checkout Engine (Idempotency & Row Locking)
		cfg.POSHandler.RegisterRoutes(protected.Group("/pos"), cfg.RedisClient)

		// Core Domain 2: Halal Supply Chain & Vendor Compliance Engine
		cfg.SupplyChainHandler.RegisterRoutes(protected.Group("/supply-chain"), cfg.RedisClient)

		// Core Domain 3: Sharia Double-Entry General Ledger (AAOIFI Invariants)
		cfg.LedgerHandler.RegisterRoutes(protected.Group("/ledger"), cfg.RedisClient)

		// Extension Domain: Store Manager Aggregated Analytics & Alerts
		cfg.ManagerHandler.RegisterRoutes(protected.Group("/manager"))
	}

	return router
}

func maintenanceModeEnabled() bool {
	enabled, _ := strconv.ParseBool(strings.TrimSpace(os.Getenv("APP_MAINTENANCE_MODE")))
	return enabled
}

func maintenanceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !maintenanceModeEnabled() || !isMutationMethod(c.Request.Method) || c.Request.URL.Path == "/api/v1/auth/logout" {
			c.Next()
			return
		}

		c.Header("Retry-After", "300")
		message := strings.TrimSpace(os.Getenv("APP_MAINTENANCE_MESSAGE"))
		if message == "" {
			message = "Temporarily unavailable for scheduled maintenance."
		}
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   gin.H{"code": "MAINTENANCE_MODE", "message": message},
		})
	}
}

func isMutationMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}
