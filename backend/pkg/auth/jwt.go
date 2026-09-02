package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrInvalidType  = errors.New("invalid token type")
)

// CustomClaims encapsulates user and tenant context in the JWT payload
type CustomClaims struct {
	UserID      string   `json:"sub"`
	TenantID    string   `json:"tenant_id"`
	TenantSlug  string   `json:"tenant_slug"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	TokenType   string   `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// JWTService handles token generation and verification
type JWTService struct {
	secretKey     []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewJWTService initializes a new JWTService
func NewJWTService() *JWTService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "super_secret_default_jwt_key_32_chars_long_minimum!"
	}

	return &JWTService{
		secretKey:     []byte(secret),
		accessExpiry:  15 * time.Minute,
		refreshExpiry: 7 * 24 * time.Hour,
	}
}

// GetPermissionsForRole returns the predefined permissions for each RBAC role
func GetPermissionsForRole(role string) []string {
	switch role {
	case "CASHIER":
		return []string{
			"pos:checkout",
			"pos:read",
			"pos:void",
			"inventory:read",
		}
	case "MANAGER":
		return []string{
			"pos:checkout",
			"pos:read",
			"pos:void",
			"inventory:read",
			"inventory:write",
			"supply_chain:manage",
			"ledger:read",
			"ai_audit:view",
		}
	case "COMPLIANCE_OFFICER":
		return []string{
			"inventory:read",
			"supply_chain:manage",
			"ledger:read",
			"ai_audit:view",
		}
	case "FINANCIAL_ADMIN":
		return []string{
			"inventory:read",
			"ledger:read",
			"ledger:write",
			"ai_audit:view",
		}
	case "SUPER_ADMIN":
		return []string{
			"pos:checkout",
			"pos:read",
			"pos:void",
			"inventory:read",
			"inventory:write",
			"supply_chain:manage",
			"ledger:read",
			"ledger:write",
			"ai_audit:view",
			"tenant:manage",
		}
	default:
		return []string{}
	}
}

// GenerateTokenPair creates both an Access Token (15m) and a Refresh Token (7d)
func (s *JWTService) GenerateTokenPair(userID, tenantID, tenantSlug, role string) (string, string, int64, error) {
	permissions := GetPermissionsForRole(role)
	now := time.Now()

	// 1. Access Token
	accessExp := now.Add(s.accessExpiry)
	accessClaims := &CustomClaims{
		UserID:      userID,
		TenantID:    tenantID,
		TenantSlug:  tenantSlug,
		Role:        role,
		Permissions: permissions,
		TokenType:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "tenet-commerce-auth",
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(s.secretKey)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to sign access token: %w", err)
	}

	// 2. Refresh Token
	refreshExp := now.Add(s.refreshExpiry)
	refreshClaims := &CustomClaims{
		UserID:      userID,
		TenantID:    tenantID,
		TenantSlug:  tenantSlug,
		Role:        role,
		Permissions: permissions,
		TokenType:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "tenet-commerce-auth",
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
		},
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(s.secretKey)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessToken, refreshToken, int64(s.accessExpiry.Seconds()), nil
}

// ValidateToken parses and cryptographically validates a JWT token string
func (s *JWTService) ValidateToken(tokenString string, expectedType string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if expectedType != "" && claims.TokenType != expectedType {
		return nil, ErrInvalidType
	}

	return claims, nil
}
