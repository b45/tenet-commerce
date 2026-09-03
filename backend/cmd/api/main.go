package main

import (
	"context"
	"net/http"
	"os"

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
)

func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8081"
	}

	if err := pkgAuth.ValidateConfiguration(); err != nil {
		logger.Error("Invalid authentication configuration", "error", err)
		os.Exit(1)
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

	// 3. Initialize Domain Services & Handlers (Dependency Injection Root)
	jwtService := pkgAuth.NewJWTService()
	tenantRepo := tenant.NewRepository(db)
	authRepo := internalAuth.NewRepository(db)
	authHandler := internalAuth.NewHandler(authRepo, jwtService)

	ledgerRepo := ledger.NewRepository()
	ledgerService := ledger.NewService(ledgerRepo)
	ledgerHandler := ledger.NewHandler(ledgerService)

	posRepo := pos.NewRepository()
	posService := pos.NewService(posRepo, ledgerService)
	posHandler := pos.NewHandler(posService)

	supplychainRepo := supplychain.NewRepository()
	supplychainService := supplychain.NewService(supplychainRepo, ledgerService)
	supplychainHandler := supplychain.NewHandler(supplychainService)

	managerRepo := manager.NewRepository()
	managerService := manager.NewService(managerRepo)
	managerHandler := manager.NewHandler(managerService)

	// 4. Setup Modular Router (Domain-Driven Routing)
	router := SetupRouter(RouterConfig{
		AuthHandler:        authHandler,
		POSHandler:         posHandler,
		SupplyChainHandler: supplychainHandler,
		LedgerHandler:      ledgerHandler,
		ManagerHandler:     managerHandler,
		TenantRepo:         tenantRepo,
		JWTService:         jwtService,
		RedisClient:        rdb,
		PostgresDB:         db,
	})

	// 5. Start API Server
	logger.Info("Tenet Commerce API server initialized",
		"port", port,
		"env", os.Getenv("APP_ENV"),
		"log_level", os.Getenv("LOG_LEVEL"),
	)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
