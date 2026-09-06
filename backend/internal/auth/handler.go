package auth

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	pkgRedis "github.com/b45/tenet-commerce/backend/pkg/redis"
	"github.com/b45/tenet-commerce/backend/pkg/response"
)

// Handler manages authentication HTTP endpoints
type Handler struct {
	repo       *Repository
	jwtService *pkgAuth.JWTService
	redis      *pkgRedis.Client
}

// NewHandler creates a new auth handler with optional Redis for token revocation
func NewHandler(repo *Repository, jwtService *pkgAuth.JWTService, redisClient ...*pkgRedis.Client) *Handler {
	var rdb *pkgRedis.Client
	if len(redisClient) > 0 && redisClient[0] != nil {
		rdb = redisClient[0]
	}
	return &Handler{
		repo:       repo,
		jwtService: jwtService,
		redis:      rdb,
	}
}

// RegisterPublicRoutes registers unauthenticated auth endpoints (login, refresh)
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/login", h.Login)
	rg.POST("/refresh", h.RefreshToken)
}

// RegisterProtectedRoutes registers authenticated identity endpoints (me, logout)
func (h *Handler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	rg.GET("/me", h.Me)
	rg.POST("/logout", h.Logout)
}

// Login handles user authentication and JWT generation
// POST /api/v1/auth/login
func (h *Handler) Login(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Login validation failed", "tenant_slug", req.TenantSlug, "error", err)
		response.BadRequest(c, "VALIDATION_ERROR", "Invalid request format", err.Error())
		return
	}

	// 1. Fetch user by tenant_slug and email
	user, err := h.repo.GetUserByEmailAndTenantSlug(c.Request.Context(), req.TenantSlug, req.Email)
	if err != nil {
		log.Warn("Login failed: user not found", "tenant_slug", req.TenantSlug, "email", req.Email)
		// NOTE: We intentionally return a generic message to prevent user enumeration attacks.
		response.Unauthorized(c, "INVALID_CREDENTIALS", "Invalid email, password, or tenant slug")
		return
	}

	// 2. Validate password
	if !pkgAuth.CheckPasswordHash(req.Password, user.PasswordHash) {
		log.Warn("Login failed: password mismatch", "tenant_slug", req.TenantSlug, "email", req.Email)
		response.Unauthorized(c, "INVALID_CREDENTIALS", "Invalid email, password, or tenant slug")
		return
	}

	// 3. Generate Token Pair
	accessToken, refreshToken, expiresIn, err := h.jwtService.GenerateTokenPair(
		user.ID,
		user.TenantID,
		user.TenantSlug,
		user.Role,
	)
	if err != nil {
		log.Error("Failed to generate token pair", "tenant_slug", req.TenantSlug, "user_id", user.ID, "error", err)
		response.InternalServerError(c, "TOKEN_GENERATION_FAILED", "Failed to create session token")
		return
	}

	log.Info("User logged in successfully",
		"tenant_slug", user.TenantSlug,
		"user_id", user.ID,
		"role", user.Role,
	)

	response.OK(c, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		User: UserProfile{
			ID:          user.ID,
			Email:       user.Email,
			FullName:    user.FullName,
			Role:        user.Role,
			TenantSlug:  user.TenantSlug,
			Permissions: pkgAuth.GetPermissionsForRole(user.Role),
		},
	})
}

// RefreshToken validates a refresh token and generates a new access token
// POST /api/v1/auth/refresh
func (h *Handler) RefreshToken(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Token refresh validation failed", "error", err)
		response.BadRequest(c, "VALIDATION_ERROR", "Invalid request format", err.Error())
		return
	}

	// 1. Validate the refresh token cryptographically
	claims, err := h.jwtService.ValidateToken(req.RefreshToken, "refresh")
	if err != nil {
		log.Warn("Invalid refresh token provided", "error", err)
		response.Unauthorized(c, "INVALID_REFRESH_TOKEN", "Refresh token is invalid or expired")
		return
	}

	// 2. Check if the refresh token is blacklisted in Redis
	if h.redis != nil && h.redis.RDB != nil {
		refreshHash := HashToken(req.RefreshToken)
		blacklisted, err := h.redis.RDB.Exists(c.Request.Context(), BlacklistKeyPrefix+refreshHash).Result()
		if err == nil && blacklisted > 0 {
			log.Warn("Attempted refresh with revoked token", "user_id", claims.UserID)
			response.Unauthorized(c, "TOKEN_REVOKED", "Refresh token has been revoked")
			return
		}
	}

	// 3. Fetch the user to ensure account is still active and retrieve latest permissions
	user, err := h.repo.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		log.Warn("Refresh token user not found or deactivated", "user_id", claims.UserID, "error", err)
		response.Unauthorized(c, "USER_INACTIVE", "User account is no longer active")
		return
	}

	// 4. Issue a new token pair
	newAccessToken, newRefreshToken, expiresIn, err := h.jwtService.GenerateTokenPair(
		user.ID,
		user.TenantID,
		user.TenantSlug,
		user.Role,
	)
	if err != nil {
		log.Error("Failed to issue refreshed token pair", "user_id", user.ID, "error", err)
		response.InternalServerError(c, "TOKEN_REFRESH_FAILED", "Failed to refresh token")
		return
	}

	// 5. Invalidate the old refresh token (Refresh Token Rotation)
	if h.redis != nil && h.redis.RDB != nil {
		remainingTTL := time.Until(claims.ExpiresAt.Time)
		if remainingTTL > 0 {
			refreshHash := HashToken(req.RefreshToken)
			_ = h.redis.RDB.Set(c.Request.Context(), BlacklistKeyPrefix+refreshHash, "rotated", remainingTTL).Err()
		}
	}

	log.Info("Token refreshed successfully", "user_id", user.ID, "tenant_slug", user.TenantSlug)

	response.OK(c, LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		User: UserProfile{
			ID:          user.ID,
			Email:       user.Email,
			FullName:    user.FullName,
			Role:        user.Role,
			TenantSlug:  user.TenantSlug,
			Permissions: pkgAuth.GetPermissionsForRole(user.Role),
		},
	})
}

// Logout invalidates the active session tokens by blacklisting them in Redis until their expiration
// POST /api/v1/auth/logout
func (h *Handler) Logout(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	// 1. Retrieve access token from context (or header fallback)
	tokenString := c.GetString("raw_token")
	if tokenString == "" {
		authHeader := c.GetHeader("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			tokenString = strings.TrimSpace(parts[1])
		}
	}

	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		log.Warn("Logout endpoint accessed without claims context")
		response.Unauthorized(c, "UNAUTHORIZED", "Authentication context not found")
		return
	}

	claims, ok := claimsVal.(*pkgAuth.CustomClaims)
	if !ok {
		log.Error("Failed to cast jwt_claims context to *CustomClaims")
		response.InternalServerError(c, "CONTEXT_TYPE_ERROR", "Failed to parse auth claims")
		return
	}

	// 2. Blacklist the Access Token in Redis for its remaining lifetime
	if h.redis != nil && h.redis.RDB != nil && tokenString != "" {
		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl > 0 {
			accessHash := HashToken(tokenString)
			if err := h.redis.RDB.Set(c.Request.Context(), BlacklistKeyPrefix+accessHash, "revoked", ttl).Err(); err != nil {
				log.Error("Failed to blacklist access token in Redis", "error", err, "user_id", claims.UserID)
			}
		}
	}

	// 3. Optional: Revoke Refresh Token if supplied in request body
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
		refreshClaims, err := h.jwtService.ValidateToken(req.RefreshToken, "refresh")
		if err == nil && refreshClaims.UserID == claims.UserID {
			if h.redis != nil && h.redis.RDB != nil {
				refreshTTL := time.Until(refreshClaims.ExpiresAt.Time)
				if refreshTTL > 0 {
					refreshHash := HashToken(req.RefreshToken)
					if err := h.redis.RDB.Set(c.Request.Context(), BlacklistKeyPrefix+refreshHash, "revoked", refreshTTL).Err(); err != nil {
						log.Error("Failed to blacklist refresh token in Redis", "error", err, "user_id", claims.UserID)
					}
				}
			}
		}
	}

	log.Info("User logged out successfully", "user_id", claims.UserID, "tenant_slug", claims.TenantSlug)

	response.OK(c, gin.H{
		"message": "Logged out successfully",
	})
}

// Me returns the currently authenticated user's profile from JWT claims
// GET /api/v1/auth/me
func (h *Handler) Me(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		log.Warn("Me endpoint accessed without claims context")
		response.Unauthorized(c, "UNAUTHORIZED", "Authentication context not found")
		return
	}

	claims, ok := claimsVal.(*pkgAuth.CustomClaims)
	if !ok {
		log.Error("Failed to cast jwt_claims context to *CustomClaims")
		response.InternalServerError(c, "CONTEXT_TYPE_ERROR", "Failed to parse auth claims")
		return
	}

	log.Debug("Authenticated profile retrieved", "user_id", claims.UserID, "tenant_slug", claims.TenantSlug)

	response.OK(c, UserProfile{
		ID:          claims.UserID,
		TenantSlug:  claims.TenantSlug,
		Role:        claims.Role,
		Permissions: claims.Permissions,
	})
}
