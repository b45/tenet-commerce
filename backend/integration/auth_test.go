package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/database"
	pkgRedis "github.com/b45/tenet-commerce/backend/pkg/redis"
)

func setupAuthIntegrationRouter(t *testing.T, db *database.PostgresDB, rdb *pkgRedis.Client) (*gin.Engine, *pkgAuth.JWTService) {
	gin.SetMode(gin.TestMode)

	jwtService := pkgAuth.NewJWTService()
	tenantRepo := tenant.NewRepository(db)
	authRepo := internalAuth.NewRepository(db)
	authHandler := internalAuth.NewHandler(authRepo, jwtService, rdb)

	router := gin.New()
	apiV1 := router.Group("/api/v1")
	{
		authPublic := apiV1.Group("/auth")
		authHandler.RegisterPublicRoutes(authPublic)

		protected := apiV1.Group("")
		protected.Use(
			internalAuth.JWTAuthMiddleware(jwtService, rdb),
			tenant.ContextMiddleware(db, tenantRepo),
		)

		authHandler.RegisterProtectedRoutes(protected.Group("/auth"))
	}

	return router, jwtService
}

func TestAuth_LoginLogoutAndRevocationLifecycle(t *testing.T) {
	db := newTestDatabase(t)
	rdb := newTestRedisClient(t)
	router, _ := setupAuthIntegrationRouter(t, db, rdb)

	// =========================================================================
	// 1. Initial Login
	// =========================================================================
	loginPayload := map[string]string{
		"tenant_slug": "al-barakah-mart",
		"email":       "cashier1@albarakah.com",
		"password":    "Password123!",
	}
	body, err := json.Marshal(loginPayload)
	require.NoError(t, err)

	wLogin := httptest.NewRecorder()
	reqLogin, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	reqLogin.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wLogin, reqLogin)

	require.Equal(t, http.StatusOK, wLogin.Code)

	var loginResp struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			User         struct {
				Email string `json:"email"`
				Role  string `json:"role"`
			} `json:"user"`
		} `json:"data"`
	}
	err = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	require.NoError(t, err)
	require.True(t, loginResp.Success)
	require.NotEmpty(t, loginResp.Data.AccessToken)
	require.NotEmpty(t, loginResp.Data.RefreshToken)
	assert.Equal(t, "cashier1@albarakah.com", loginResp.Data.User.Email)
	assert.Equal(t, "CASHIER", loginResp.Data.User.Role)

	accessToken := loginResp.Data.AccessToken
	refreshToken := loginResp.Data.RefreshToken

	// =========================================================================
	// 2. Validate Session with GET /api/v1/auth/me
	// =========================================================================
	wMe := httptest.NewRecorder()
	reqMe, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(wMe, reqMe)

	require.Equal(t, http.StatusOK, wMe.Code)
	assert.Contains(t, wMe.Body.String(), "al-barakah-mart")

	// =========================================================================
	// 3. Logout (Revoking Access Token and Refresh Token)
	// =========================================================================
	logoutPayload := map[string]string{
		"refresh_token": refreshToken,
	}
	logoutBody, err := json.Marshal(logoutPayload)
	require.NoError(t, err)

	wLogout := httptest.NewRecorder()
	reqLogout, _ := http.NewRequest("POST", "/api/v1/auth/logout", bytes.NewReader(logoutBody))
	reqLogout.Header.Set("Authorization", "Bearer "+accessToken)
	reqLogout.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wLogout, reqLogout)

	require.Equal(t, http.StatusOK, wLogout.Code)
	assert.Contains(t, wLogout.Body.String(), "Logged out successfully")

	// =========================================================================
	// 4. Assert Revoked Access Token Is Rejected
	// =========================================================================
	wMeRevoked := httptest.NewRecorder()
	reqMeRevoked, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
	reqMeRevoked.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(wMeRevoked, reqMeRevoked)

	assert.Equal(t, http.StatusUnauthorized, wMeRevoked.Code)
	assert.Contains(t, wMeRevoked.Body.String(), "TOKEN_REVOKED")

	// =========================================================================
	// 5. Assert Revoked Refresh Token Is Rejected
	// =========================================================================
	refreshPayload := map[string]string{
		"refresh_token": refreshToken,
	}
	refreshBody, err := json.Marshal(refreshPayload)
	require.NoError(t, err)

	wRefreshRevoked := httptest.NewRecorder()
	reqRefreshRevoked, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
	reqRefreshRevoked.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wRefreshRevoked, reqRefreshRevoked)

	assert.Equal(t, http.StatusUnauthorized, wRefreshRevoked.Code)
	assert.Contains(t, wRefreshRevoked.Body.String(), "TOKEN_REVOKED")

	// =========================================================================
	// 6. Test Refresh Token Rotation Lifecycle
	// =========================================================================
	// 6a. Fresh Login
	wLogin2 := httptest.NewRecorder()
	reqLogin2, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	reqLogin2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wLogin2, reqLogin2)
	require.Equal(t, http.StatusOK, wLogin2.Code)

	var loginResp2 struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wLogin2.Body.Bytes(), &loginResp2))

	// 6b. First Refresh (Succeeds and rotates old refresh token)
	rotPayload := map[string]string{"refresh_token": loginResp2.Data.RefreshToken}
	rotBody, _ := json.Marshal(rotPayload)

	wRot := httptest.NewRecorder()
	reqRot, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(rotBody))
	reqRot.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wRot, reqRot)

	require.Equal(t, http.StatusOK, wRot.Code)
	var rotResp struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wRot.Body.Bytes(), &rotResp))
	assert.NotEmpty(t, rotResp.Data.AccessToken)
	assert.NotEqual(t, loginResp2.Data.RefreshToken, rotResp.Data.RefreshToken)

	// 6c. Second Refresh with OLD Refresh Token (Must Fail with TOKEN_REVOKED)
	wReplay := httptest.NewRecorder()
	reqReplay, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(rotBody))
	reqReplay.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wReplay, reqReplay)

	assert.Equal(t, http.StatusUnauthorized, wReplay.Code)
	assert.Contains(t, wReplay.Body.String(), "TOKEN_REVOKED")
}
