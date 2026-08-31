package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/database"
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
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// 2. Initialize Services & Repositories
	jwtService := pkgAuth.NewJWTService()
	tenantRepo := tenant.NewRepository(db)
	authRepo := internalAuth.NewRepository(db)
	authHandler := internalAuth.NewHandler(authRepo, jwtService)

	// 3. Setup Gin Engine
	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("APP_DEBUG") == "true" {
		gin.SetMode(gin.DebugMode)
	}
	router := gin.Default()

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
			authGroup.POST("/login", authHandler.Login)
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
	log.Printf("Tenet Commerce API starting on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
