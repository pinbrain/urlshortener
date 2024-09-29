package middleware

import "testing"

func BenchmarkBuildJWTString(b *testing.B) {
	userID := 1
	for i := 0; i < b.N; i++ {
		_, err := BuildJWTString(userID)
		if err != nil {
			b.Fatalf("failed build jwt string: %v", err)
		}
	}
}

func BenchmarkGetJWTClaims(b *testing.B) {
	userID := 1
	tokenString, err := BuildJWTString(userID)
	if err != nil {
		b.Fatalf("failed build jwt string: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = getJWTClaims(tokenString)
		if err != nil {
			b.Fatalf("failed get jwt claims: %v", err)
		}
	}
}
