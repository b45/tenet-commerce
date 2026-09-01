package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/b45/tenet-commerce/backend/internal/pos"
	"github.com/b45/tenet-commerce/backend/internal/supplychain"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/database"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	pkgRedis "github.com/b45/tenet-commerce/backend/pkg/redis"
)

func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8081"
	}

	ctx := context.Background()

	// 1. Initialize PostgreSQL Database Connection Pool
	db, err := database.NewPostgresDB(ctx)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 2. Initialize Redis Client
	rdb, err := pkgRedis.NewRedisClient(ctx)
	if err != nil {
		logger.Error("Failed to initialize Redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// 3. Initialize Services & Repositories
	jwtService := pkgAuth.NewJWTService()
	tenantRepo := tenant.NewRepository(db)
	authRepo := internalAuth.NewRepository(db)
	authHandler := internalAuth.NewHandler(authRepo, jwtService)

	// Initialize Ledger first (needed by POS and Supply Chain)
	ledgerRepo := ledger.NewRepository()
	ledgerService := ledger.NewService(ledgerRepo)
	ledgerHandler := ledger.NewHandler(ledgerService)

	posRepo := pos.NewRepository()
	posService := pos.NewService(posRepo, ledgerService)
	posHandler := pos.NewHandler(posService)

	supplychainRepo := supplychain.NewRepository()
	supplychainService := supplychain.NewService(supplychainRepo, ledgerService)
	supplychainHandler := supplychain.NewHandler(supplychainService)

	// 4. Setup Gin Engine with Structured Observability Stack
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

	// 5. PUBLIC ROUTES
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "tenet-commerce-api",
		})
	})

	// 6. API v1 Routes
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

			// --- POS Domain Module (Phase 2) ---
			posGroup := protected.Group("/pos")
			{
				posGroup.GET("/products",
					internalAuth.RequirePermission("inventory:read"),
					posHandler.GetProducts,
				)
				posGroup.POST("/checkout",
					internalAuth.RequirePermission("pos:checkout"),
					pkgRedis.IdempotencyMiddleware(rdb, 24*time.Hour),
					posHandler.Checkout,
				)
			}

			// --- Supply Chain Domain Module ---
			supplychainGroup := protected.Group("")
			supplychainGroup.Use(internalAuth.RequirePermission("supply_chain:manage"))
			supplychainHandler.RegisterRoutes(supplychainGroup)

			// --- Placeholders for future phases ---
			// TODO(Phase 2): Move to internal/manager package handler
			protected.GET("/manager/dashboard",
				internalAuth.RequireRole("MANAGER", "SUPER_ADMIN"),
				placeholderManagerDashboardHandler,
			)

			// --- Ledger Domain Module (Phase 2) ---
			ledgerGroup := protected.Group("/ledger")
			{
				ledgerGroup.GET("/accounts",
					internalAuth.RequirePermission("ledger:read"),
					ledgerHandler.GetChartOfAccounts,
				)
				ledgerGroup.GET("/entries",
					internalAuth.RequirePermission("ledger:read"),
					ledgerHandler.GetJournalEntries,
				)
				ledgerGroup.POST("/entries",
					internalAuth.RequirePermission("ledger:write"),
					ledgerHandler.CreateManualEntry,
				)
				ledgerGroup.GET("/trial-balance",
					internalAuth.RequirePermission("ledger:read"),
					ledgerHandler.GetTrialBalance,
				)
			}
		}
	}

	// 7. Start API Server
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
