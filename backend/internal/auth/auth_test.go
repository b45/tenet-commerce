package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
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
