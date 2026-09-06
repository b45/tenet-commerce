package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	pkgRedis "github.com/b45/tenet-commerce/backend/pkg/redis"
)

func TestPasswordHashing(t *testing.T) {
	rawPassword := "Password123!"

	hash, err := pkgAuth.HashPassword(rawPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	t.Logf("BCRYPT_HASH_OF_PASSWORD123: %s", hash)

	// Valid password check
	assert.True(t, pkgAuth.CheckPasswordHash(rawPassword, hash))

	// Invalid password check
	assert.False(t, pkgAuth.CheckPasswordHash("WrongPassword!", hash))
}

func TestJWTGenerationAndValidation(t *testing.T) {
	jwtService := pkgAuth.NewJWTService()

	userID := "usr-12345"
	tenantID := "ten-67890"
	tenantSlug := "al-barakah-mart"
	role := "CASHIER"

	accessToken, refreshToken, expiresIn, err := jwtService.GenerateTokenPair(userID, tenantID, tenantSlug, role)
	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.Equal(t, int64(900), expiresIn)

	// Validate Access Token
	claims, err := jwtService.ValidateToken(accessToken, "access")
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, tenantID, claims.TenantID)
	assert.Equal(t, tenantSlug, claims.TenantSlug)
	assert.Equal(t, "CASHIER", claims.Role)
	assert.Contains(t, claims.Permissions, "pos:checkout")
	assert.Contains(t, claims.Permissions, "inventory:read")

	// Validate Token Type mismatch
	_, err = jwtService.ValidateToken(accessToken, "refresh")
	assert.ErrorIs(t, err, pkgAuth.ErrInvalidType)
}

func TestRequireRoleMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := pkgAuth.NewJWTService()

	router := gin.New()
	router.Use(internalAuth.JWTAuthMiddleware(jwtService))
	router.GET("/admin-only", internalAuth.RequireRole("MANAGER", "SUPER_ADMIN"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 1. Cashier attempts to access manager endpoint -> 403 Forbidden
	cashierToken, _, _, _ := jwtService.GenerateTokenPair("usr-1", "ten-1", "al-barakah-mart", "CASHIER")
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/admin-only", nil)
	req1.Header.Set("Authorization", "Bearer "+cashierToken)
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusForbidden, w1.Code)

	// 2. Manager attempts to access manager endpoint -> 200 OK
	managerToken, _, _, _ := jwtService.GenerateTokenPair("usr-2", "ten-1", "al-barakah-mart", "MANAGER")
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/admin-only", nil)
	req2.Header.Set("Authorization", "Bearer "+managerToken)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestRequirePermissionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := pkgAuth.NewJWTService()

	router := gin.New()
	router.Use(internalAuth.JWTAuthMiddleware(jwtService))
	router.GET("/ledger", internalAuth.RequirePermission("ledger:read"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 1. Cashier lacks 'ledger:read' -> 403 Forbidden
	cashierToken, _, _, _ := jwtService.GenerateTokenPair("usr-1", "ten-1", "al-barakah-mart", "CASHIER")
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/ledger", nil)
	req1.Header.Set("Authorization", "Bearer "+cashierToken)
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusForbidden, w1.Code)

	// 2. Financial Admin has 'ledger:read' -> 200 OK
	financeToken, _, _, _ := jwtService.GenerateTokenPair("usr-3", "ten-1", "al-barakah-mart", "FINANCIAL_ADMIN")
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/ledger", nil)
	req2.Header.Set("Authorization", "Bearer "+financeToken)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestTokenHashing(t *testing.T) {
	token := "sample.jwt.token"
	h1 := internalAuth.HashToken(token)
	h2 := internalAuth.HashToken(token)
	assert.NotEmpty(t, h1)
	assert.Equal(t, h1, h2)
	assert.Equal(t, 64, len(h1)) // sha256 hex length
}

func TestJWTAuthMiddleware_Revocation(t *testing.T) {
	ctx := context.Background()
	rdb, err := pkgRedis.NewRedisClient(ctx)
	if err != nil {
		t.Skipf("Skipping Redis revocation test: %v", err)
	}
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	jwtService := pkgAuth.NewJWTService()

	router := gin.New()
	router.Use(internalAuth.JWTAuthMiddleware(jwtService, rdb))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	token, _, _, err := jwtService.GenerateTokenPair("usr-revoked", "ten-1", "al-barakah-mart", "CASHIER")
	assert.NoError(t, err)

	// 1. Before revocation: 200 OK
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/protected", nil)
	req1.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// 2. Blacklist token in Redis
	tokenHash := internalAuth.HashToken(token)
	err = rdb.RDB.Set(ctx, internalAuth.BlacklistKeyPrefix+tokenHash, "revoked", 10*time.Second).Err()
	assert.NoError(t, err)
	defer rdb.RDB.Del(ctx, internalAuth.BlacklistKeyPrefix+tokenHash)

	// 3. After revocation: 401 Unauthorized with TOKEN_REVOKED
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/protected", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
	assert.Contains(t, w2.Body.String(), "TOKEN_REVOKED")
}

