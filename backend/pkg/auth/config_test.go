package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		secret      string
		wantErr     bool
	}{
		{name: "development permits local default", environment: "development", secret: ""},
		{name: "unset environment preserves local development behavior", environment: "", secret: ""},
		{name: "production rejects absent secret", environment: "production", secret: "", wantErr: true},
		{name: "production rejects default secret", environment: "production", secret: defaultDevelopmentJWTSecret, wantErr: true},
		{name: "production accepts configured secret", environment: "production", secret: "a-unique-production-secret-with-sufficient-length"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_ENV", tt.environment)
			t.Setenv("JWT_SECRET", tt.secret)

			err := ValidateConfiguration()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
