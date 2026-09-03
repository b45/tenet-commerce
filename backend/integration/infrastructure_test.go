package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/b45/tenet-commerce/backend/internal/pos"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/database"
	pkgRedis "github.com/b45/tenet-commerce/backend/pkg/redis"
)

const (
	testDatabaseName = "tenet_commerce"
	testDatabaseUser = "postgres"
	testDatabasePass = "postgres"
)

var integrationDSN string

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	initScript := filepath.Join("..", "..", "scripts", "init_dev_db.sql")
	if _, err := os.Stat(initScript); err != nil {
		fmt.Fprintf(os.Stderr, "integration setup failed: canonical init script unavailable: %v\n", err)
		os.Exit(1)
	}

	postgresContainer, err := tcpostgres.Run(
		ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase(testDatabaseName),
		tcpostgres.WithUsername(testDatabaseUser),
		tcpostgres.WithPassword(testDatabasePass),
		tcpostgres.WithInitScripts(initScript),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration setup failed: PostgreSQL container: %v\n", err)
		os.Exit(1)
	}

	redisContainer, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		_ = testcontainers.TerminateContainer(postgresContainer)
		fmt.Fprintf(os.Stderr, "integration setup failed: Redis container: %v\n", err)
		os.Exit(1)
	}

	integrationDSN, err = postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(redisContainer)
		_ = testcontainers.TerminateContainer(postgresContainer)
		fmt.Fprintf(os.Stderr, "integration setup failed: PostgreSQL connection string: %v\n", err)
		os.Exit(1)
	}

	redisURL, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(redisContainer)
		_ = testcontainers.TerminateContainer(postgresContainer)
		fmt.Fprintf(os.Stderr, "integration setup failed: Redis connection string: %v\n", err)
		os.Exit(1)
	}

	if err := configureIntegrationEnvironment(integrationDSN, redisURL); err != nil {
		_ = testcontainers.TerminateContainer(redisContainer)
		_ = testcontainers.TerminateContainer(postgresContainer)
		fmt.Fprintf(os.Stderr, "integration setup failed: environment: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	if err := testcontainers.TerminateContainer(redisContainer); err != nil {
		fmt.Fprintf(os.Stderr, "integration cleanup warning: Redis container: %v\n", err)
	}
	if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
		fmt.Fprintf(os.Stderr, "integration cleanup warning: PostgreSQL container: %v\n", err)
	}
	os.Exit(code)
}

func configureIntegrationEnvironment(postgresDSN, redisDSN string) error {
	postgresURL, err := url.Parse(postgresDSN)
	if err != nil {
		return fmt.Errorf("parse PostgreSQL DSN: %w", err)
	}
	postgresHost, postgresPort, err := net.SplitHostPort(postgresURL.Host)
	if err != nil {
		return fmt.Errorf("parse PostgreSQL endpoint: %w", err)
	}

	redisURL, err := url.Parse(redisDSN)
	if err != nil {
		return fmt.Errorf("parse Redis DSN: %w", err)
	}
	redisHost, redisPort, err := net.SplitHostPort(redisURL.Host)
	if err != nil {
		return fmt.Errorf("parse Redis endpoint: %w", err)
	}

	for key, value := range map[string]string{
		"DATABASE_HOST":     postgresHost,
		"DATABASE_PORT":     postgresPort,
		"DATABASE_USER":     testDatabaseUser,
		"DATABASE_PASSWORD": testDatabasePass,
		"DATABASE_NAME":     testDatabaseName,
		"DATABASE_SSLMODE":  "disable",
		"REDIS_HOST":        redisHost,
		"REDIS_PORT":        redisPort,
		"REDIS_PASSWORD":    "",
		"REDIS_DB":          "0",
	} {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}

	return nil
}

func newPoolSizeOneDatabase(t *testing.T) *database.PostgresDB {
	t.Helper()

	config, err := pgxpool.ParseConfig(integrationDSN)
	require.NoError(t, err)
	config.MaxConns = 1
	config.MinConns = 0

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return &database.PostgresDB{Pool: pool}
}

func newTestDatabase(t *testing.T) *database.PostgresDB {
	t.Helper()

	config, err := pgxpool.ParseConfig(integrationDSN)
	require.NoError(t, err)
	config.MaxConns = 10
	config.MinConns = 1

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return &database.PostgresDB{Pool: pool}
}

func TestCanonicalInitScriptCreatesRequiredInfrastructure(t *testing.T) {
	db := newPoolSizeOneDatabase(t)

	var tenantCount int
	err := db.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM public.tenants
		WHERE slug IN ('al-barakah-mart', 'darussalam-store') AND status = 'ACTIVE'
	`).Scan(&tenantCount)
	require.NoError(t, err)
	require.Equal(t, 2, tenantCount)

	for _, schema := range []string{"tenant_al_barakah_mart", "tenant_darussalam_store"} {
		var exists bool
		err := db.Pool.QueryRow(context.Background(), `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = $1 AND table_name = 'products'
			)
		`, schema).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "products table must exist in %s", schema)
	}
}

func TestTenantContextAlternatesSchemasOnSingleConnectionPool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPoolSizeOneDatabase(t)
	repo := tenant.NewRepository(db)

	router := gin.New()
	router.Use(tenant.ContextMiddleware(db, repo))
	router.GET("/schema", func(c *gin.Context) {
		conn, ok := tenant.GetConn(c)
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		var schema string
		if err := conn.QueryRow(c.Request.Context(), "SELECT current_schema()").Scan(&schema); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.String(http.StatusOK, schema)
	})

	for _, expected := range []struct {
		slug   string
		schema string
	}{
		{slug: "al-barakah-mart", schema: "tenant_al_barakah_mart"},
		{slug: "darussalam-store", schema: "tenant_darussalam_store"},
		{slug: "al-barakah-mart", schema: "tenant_al_barakah_mart"},
	} {
		request := httptest.NewRequest(http.MethodGet, "/schema", nil)
		request.Header.Set("X-Tenant-ID", expected.slug)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, expected.schema, response.Body.String())
	}

	conn, err := db.Pool.Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Release()

	var currentSchema string
	err = conn.QueryRow(context.Background(), "SELECT current_schema()").Scan(&currentSchema)
	require.NoError(t, err)
	require.Equal(t, "public", currentSchema, "a released tenant connection must not retain its search_path")
}

func TestRedisIdempotencyRunsAgainstContainer(t *testing.T) {
	client, err := pkgRedis.NewRedisClient(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.RDB.FlushDB(context.Background()).Err())

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_slug", "integration-tenant")
		c.Next()
	})
	router.Use(pkgRedis.IdempotencyMiddleware(client, time.Minute))

	executions := 0
	router.POST("/mutation", func(c *gin.Context) {
		executions++
		c.JSON(http.StatusCreated, gin.H{"execution_count": executions})
	})

	for attempt := 0; attempt < 2; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/mutation", bytes.NewBufferString(`{}`))
		request.Header.Set("Idempotency-Key", "integration-idempotency-key")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusCreated, response.Code)
	}

	require.Equal(t, 1, executions)
}

func TestPOSCheckoutIsExecutedOnceAgainstPostgresAndRedis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPoolSizeOneDatabase(t)
	tenantRepo := tenant.NewRepository(db)
	redisClient, err := pkgRedis.NewRedisClient(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = redisClient.Close() })
	require.NoError(t, redisClient.RDB.FlushDB(context.Background()).Err())

	posHandler := pos.NewHandler(pos.NewService(pos.NewRepository(), ledger.NewService(ledger.NewRepository())))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("jwt_claims", &pkgAuth.CustomClaims{Permissions: []string{"pos:checkout"}})
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	posHandler.RegisterRoutes(router.Group("/api/v1/pos"), redisClient)

	initialStock := stockForSKU(t, db, "SKU-BEEF-01")
	body := []byte(`{"items":[{"sku":"SKU-BEEF-01","quantity":1}],"payment_method":"CASH","cash_tendered":100000}`)

	var firstResponse string
	for attempt := 0; attempt < 2; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", "integration-checkout-key")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		require.Equal(t, http.StatusCreated, response.Code)
		if attempt == 0 {
			firstResponse = response.Body.String()
			continue
		}
		require.Equal(t, "HIT", response.Header().Get("X-Cache-Lookup"))
		require.Equal(t, firstResponse, response.Body.String())
	}

	require.Equal(t, initialStock-1, stockForSKU(t, db, "SKU-BEEF-01"))
}

func stockForSKU(t *testing.T, db *database.PostgresDB, sku string) int {
	t.Helper()

	conn, err := db.Pool.Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Release()

	_, err = conn.Exec(context.Background(), "SET search_path TO tenant_al_barakah_mart, public")
	require.NoError(t, err)

	var stock int
	err = conn.QueryRow(context.Background(), `
		SELECT inventory.stock_quantity
		FROM products
		JOIN inventory ON inventory.product_id = products.id
		WHERE products.sku = $1
	`, sku).Scan(&stock)
	require.NoError(t, err)

	return stock
}

func newTestRedisClient(t *testing.T) *pkgRedis.Client {
	t.Helper()
	client, err := pkgRedis.NewRedisClient(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.RDB.FlushDB(context.Background()).Err())
	return client
}
