package main

import (
	"context"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/database"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
)

func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8081"
	}

	// 1. Initialize Database Connection Pool
	ctx := context.Background()
	db, err := database.NewPostgresDB(ctx)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 2. Initialize Services & Repositories
	jwtService := pkgAuth.NewJWTService()
	tenantRepo := tenant.NewRepository(db)
	authRepo := internalAuth.NewRepository(db)
	authHandler := internalAuth.NewHandler(authRepo, jwtService)

	// 3. Setup Gin Engine with Structured Observability Stack
	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("APP_DEBUG") == "true" {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// Mount Global Observability Middlewares
	router.Use(logger.RealIPMiddleware())    // 1. Resolve real client IP behind proxies
	router.Use(logger.TraceMiddleware())     // 2. Generate/propagate X-Trace-ID & X-Span-ID
	router.Use(logger.AccessLogMiddleware()) // 3. Structured JSON access logging
	router.Use(logger.RecoveryMiddleware())  // 4. Structured panic recovery

	// 4. PUBLIC ROUTES
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "tenet-commerce-api",
		})
	})

	// 5. API v1 Routes
	v1 := router.Group("/api/v1")
	{
		// --- Unauthenticated Auth Endpoints ---
		authGroup := v1.Group("/auth")
		{
			// Capture request payload for login audit
			authGroup.POST("/login", logger.AuditBodyMiddleware(), authHandler.Login)
			authGroup.POST("/refresh", authHandler.RefreshToken)
		}

		// --- Authenticated Routes (JWT + Tenant Context) ---
		// All handlers below operate with a schema-scoped DB connection injected by
		// tenant.ContextMiddleware; they must use c.Get("db_conn") for DB access.
		protected := v1.Group("")
		protected.Use(internalAuth.JWTAuthMiddleware(jwtService))
		protected.Use(tenant.ContextMiddleware(db, tenantRepo))
		{
			protected.GET("/auth/me", authHandler.Me)

			// NOTE: Domain handlers (POS, Supply Chain, Ledger, AI Auditor)
			// will be wired here in Phase 2 from their respective internal packages.
			// The placeholder routes below will be removed as real handlers are implemented.

			// --- Phase 1 Verification Endpoints (placeholder) ---
			// TODO(Phase 2): Move to internal/inventory package handler
			protected.GET("/products",
				internalAuth.RequirePermission("inventory:read"),
				placeholderProductsHandler,
			)

			// TODO(Phase 2): Move to internal/manager package handler
			protected.GET("/manager/dashboard",
				internalAuth.RequireRole("MANAGER", "SUPER_ADMIN"),
				placeholderManagerDashboardHandler,
			)

			// TODO(Phase 3): Move to internal/ledger package handler
			protected.GET("/finance/ledger",
				internalAuth.RequirePermission("ledger:read"),
				placeholderLedgerHandler,
			)
		}
	}

	// 6. Start API Server
	logger.Info("Tenet Commerce API server initialized",
		"port", port,
		"env", os.Getenv("APP_ENV"),
		"log_level", os.Getenv("LOG_LEVEL"),
	)
	if err := router.Run(":" + port); err != nil {
		logger.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
