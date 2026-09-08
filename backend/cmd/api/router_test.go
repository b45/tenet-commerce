package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMaintenanceMiddleware(t *testing.T) {
	previousMode, previousMessage := os.Getenv("APP_MAINTENANCE_MODE"), os.Getenv("APP_MAINTENANCE_MESSAGE")
	t.Cleanup(func() {
		_ = os.Setenv("APP_MAINTENANCE_MODE", previousMode)
		_ = os.Setenv("APP_MAINTENANCE_MESSAGE", previousMessage)
	})
	_ = os.Setenv("APP_MAINTENANCE_MODE", "true")
	_ = os.Setenv("APP_MAINTENANCE_MESSAGE", "Scheduled window")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(maintenanceMiddleware())
	router.POST("/mutation", func(c *gin.Context) { c.Status(http.StatusCreated) })
	router.GET("/read", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/api/v1/auth/logout", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	mutation := httptest.NewRecorder()
	router.ServeHTTP(mutation, httptest.NewRequest(http.MethodPost, "/mutation", nil))
	require.Equal(t, http.StatusServiceUnavailable, mutation.Code)
	require.Equal(t, "300", mutation.Header().Get("Retry-After"))
	require.Contains(t, mutation.Body.String(), "MAINTENANCE_MODE")

	read := httptest.NewRecorder()
	router.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/read", nil))
	require.Equal(t, http.StatusOK, read.Code)

	logout := httptest.NewRecorder()
	router.ServeHTTP(logout, httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil))
	require.Equal(t, http.StatusNoContent, logout.Code)
}
