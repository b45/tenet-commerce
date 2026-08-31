package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/response"
)

// JWTAuthMiddleware validates Bearer JWT access tokens and injects claims into Gin Context.
// This must run BEFORE tenant.ContextMiddleware so that tenant_slug is available from the token.
func JWTAuthMiddleware(jwtService *pkgAuth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.AbortUnauthorized(c, "MISSING_AUTH_HEADER", "Authorization header is required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" && strings.TrimSpace(parts[1]) != "") {
			response.AbortUnauthorized(c, "INVALID_AUTH_FORMAT", "Authorization header format must be Bearer <token>")
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		claims, err := jwtService.ValidateToken(tokenString, "access")
		if err != nil {
			response.AbortUnauthorized(c, "INVALID_OR_EXPIRED_TOKEN", "Token is invalid or expired")
			return
		}

		// Inject all authenticated context into Gin.
		// These keys are the canonical source of truth for identity throughout the request.
		c.Set("jwt_claims", claims)
		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("tenant_slug", claims.TenantSlug)
		c.Set("role", claims.Role)
		c.Set("permissions", claims.Permissions)

		c.Next()
	}
}

// RequirePermission checks if the authenticated user possesses a specific permission.
// Must be used after JWTAuthMiddleware.
func RequirePermission(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get("jwt_claims")
		if !exists {
			response.AbortUnauthorized(c, "UNAUTHORIZED", "Authentication required")
			return
		}

		claims, ok := claimsVal.(*pkgAuth.CustomClaims)
		if !ok {
			response.AbortInternalServerError(c, "CONTEXT_TYPE_ERROR", "Failed to parse auth claims")
			return
		}

		for _, p := range claims.Permissions {
			if p == requiredPermission {
				c.Next()
				return
			}
		}

		response.AbortForbidden(c, "FORBIDDEN_INSUFFICIENT_PERMISSION",
			"You do not have the required permission: "+requiredPermission)
	}
}

// RequireRole checks if the authenticated user has one of the specified roles.
// Must be used after JWTAuthMiddleware.
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get("jwt_claims")
		if !exists {
			response.AbortUnauthorized(c, "UNAUTHORIZED", "Authentication required")
			return
		}

		claims, ok := claimsVal.(*pkgAuth.CustomClaims)
		if !ok {
			response.AbortInternalServerError(c, "CONTEXT_TYPE_ERROR", "Failed to parse auth claims")
			return
		}

		for _, r := range allowedRoles {
			if claims.Role == r {
				c.Next()
				return
			}
		}

		response.AbortForbidden(c, "FORBIDDEN_ROLE_MISMATCH", "Access restricted for your current role")
	}
}
