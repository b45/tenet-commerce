package auth

import (
	"github.com/gin-gonic/gin"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/response"
)

// Handler manages authentication HTTP endpoints
type Handler struct {
	repo       *Repository
	jwtService *pkgAuth.JWTService
}

// NewHandler creates a new auth handler
func NewHandler(repo *Repository, jwtService *pkgAuth.JWTService) *Handler {
	return &Handler{
		repo:       repo,
		jwtService: jwtService,
	}
}

// Login handles user authentication and JWT generation
// POST /api/v1/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Invalid request format", err.Error())
		return
	}

	// 1. Fetch user by tenant_slug and email
	user, err := h.repo.GetUserByEmailAndTenantSlug(c.Request.Context(), req.TenantSlug, req.Email)
	if err != nil {
		// NOTE: We intentionally return a generic message to prevent user enumeration attacks.
		response.Unauthorized(c, "INVALID_CREDENTIALS", "Invalid email, password, or tenant slug")
		return
	}

	// 2. Validate password
	if !pkgAuth.CheckPasswordHash(req.Password, user.PasswordHash) {
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
		response.InternalServerError(c, "TOKEN_GENERATION_FAILED", "Failed to create session token")
		return
	}

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
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "refresh_token is required")
		return
	}

	// Validate the submitted token is specifically a refresh token
	claims, err := h.jwtService.ValidateToken(req.RefreshToken, "refresh")
	if err != nil {
		response.Unauthorized(c, "INVALID_REFRESH_TOKEN", "Refresh token is invalid or expired")
		return
	}

	// Re-verify user is still active in the database (accounts may be deactivated between sessions)
	user, err := h.repo.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		response.Unauthorized(c, "USER_INACTIVE", "User account is no longer active")
		return
	}

	// Generate a fresh token pair (rotation pattern)
	newAccessToken, newRefreshToken, expiresIn, err := h.jwtService.GenerateTokenPair(
		user.ID,
		user.TenantID,
		user.TenantSlug,
		user.Role,
	)
	if err != nil {
		response.InternalServerError(c, "TOKEN_REFRESH_FAILED", "Failed to refresh token")
		return
	}

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

// Me returns the currently authenticated user's profile from JWT claims
// GET /api/v1/auth/me
func (h *Handler) Me(c *gin.Context) {
	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		response.Unauthorized(c, "UNAUTHORIZED", "Authentication context not found")
		return
	}

	claims, ok := claimsVal.(*pkgAuth.CustomClaims)
	if !ok {
		response.InternalServerError(c, "CONTEXT_TYPE_ERROR", "Failed to parse auth claims")
		return
	}

	response.OK(c, UserProfile{
		ID:          claims.UserID,
		TenantSlug:  claims.TenantSlug,
		Role:        claims.Role,
		Permissions: claims.Permissions,
	})
}
