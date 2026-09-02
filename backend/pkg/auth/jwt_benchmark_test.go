package auth

import (
	"testing"
)

func BenchmarkGenerateTokenPair(b *testing.B) {
	jwtSvc := NewJWTService()
	userID := "usr_9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"
	tenantID := "ten_a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"
	tenantSlug := "al-barakah-mart"
	role := "MANAGER"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _, err := jwtSvc.GenerateTokenPair(userID, tenantID, tenantSlug, role)
		if err != nil {
			b.Fatalf("failed to generate token: %v", err)
		}
	}
}

func BenchmarkValidateToken(b *testing.B) {
	jwtSvc := NewJWTService()
	userID := "usr_9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"
	tenantID := "ten_a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"
	tenantSlug := "al-barakah-mart"
	role := "CASHIER"

	accessToken, _, _, err := jwtSvc.GenerateTokenPair(userID, tenantID, tenantSlug, role)
	if err != nil {
		b.Fatalf("setup failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		claims, err := jwtSvc.ValidateToken(accessToken, "access")
		if err != nil || claims.UserID != userID {
			b.Fatalf("token validation failed: %v", err)
		}
	}
}

func BenchmarkValidateToken_Parallel(b *testing.B) {
	jwtSvc := NewJWTService()
	userID := "usr_9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"
	tenantID := "ten_a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"
	tenantSlug := "al-barakah-mart"
	role := "CASHIER"

	accessToken, _, _, err := jwtSvc.GenerateTokenPair(userID, tenantID, tenantSlug, role)
	if err != nil {
		b.Fatalf("setup failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			claims, err := jwtSvc.ValidateToken(accessToken, "access")
			if err != nil || claims.UserID != userID {
				b.Fatalf("token validation failed: %v", err)
			}
		}
	})
}

func BenchmarkGetPermissionsForRole(b *testing.B) {
	roles := []string{"CASHIER", "MANAGER", "SUPER_ADMIN", "COMPLIANCE_OFFICER", "FINANCIAL_ADMIN"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		role := roles[i%len(roles)]
		perms := GetPermissionsForRole(role)
		if len(perms) == 0 {
			b.Fatalf("expected permissions for role %s", role)
		}
	}
}
