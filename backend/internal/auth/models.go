package auth

import (
	"time"
)

// User represents a user record from public.users
type User struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	TenantSlug   string    `json:"tenant_slug"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// LoginRequest defines the request body for POST /api/v1/auth/login
type LoginRequest struct {
	TenantSlug string `json:"tenant_slug" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required"`
}

// UserProfile is the public representation of a user
type UserProfile struct {
	ID         string   `json:"id"`
	Email      string   `json:"email"`
	FullName   string   `json:"full_name"`
	Role       string   `json:"role"`
	TenantSlug string   `json:"tenant_slug"`
	Permissions []string `json:"permissions"`
}

// LoginResponse defines the successful response body for login
type LoginResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int64       `json:"expires_in"` // seconds
	User         UserProfile `json:"user"`
}

// RefreshTokenRequest defines the request body for POST /api/v1/auth/refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
