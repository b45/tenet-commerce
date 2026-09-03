package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
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
			internalAuth.JWTAuthMiddleware(cfg.JWTService),
			tenant.ContextMiddleware(cfg.PostgresDB, cfg.TenantRepo),
		)

		// Identity & Self-Profile Introspection
		cfg.AuthHandler.RegisterProtectedRoutes(protected.Group("/auth"))

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
