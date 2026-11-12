package authentication

import (
	"strings"
	"testing"
)

func TestLoadJwtSecretRequiresAtLeast32Bytes(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		valid bool
	}{
		{name: "missing", value: "", valid: false},
		{name: "one byte short", value: strings.Repeat("s", 31), valid: false},
		{name: "previously hardcoded secret", value: "replace-with-a-strong-secret", valid: false},
		{name: "minimum length", value: strings.Repeat("s", 32), valid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", tc.value)
			secret, err := loadJwtSecret()
			if tc.valid {
				if err != nil || string(secret) != tc.value {
					t.Fatalf("loadJwtSecret() = (%q, %v), want the configured secret", secret, err)
				}
				return
			}
			if err == nil || secret != nil {
				t.Fatalf("loadJwtSecret() = (%q, %v), want an error", secret, err)
			}
			// The error is logged at startup, so it must never echo the value.
			if tc.value != "" && strings.Contains(err.Error(), tc.value) {
				t.Fatalf("error %q contains the secret", err)
			}
		})
	}
}

func TestInitJwtSecretUsesEnvironmentValue(t *testing.T) {
	previous := JwtSecret
	t.Cleanup(func() { JwtSecret = previous })
	value := strings.Repeat("k", 48)
	t.Setenv("JWT_SECRET", value)
	InitJwtSecret()
	if string(JwtSecret) != value {
		t.Fatalf("JwtSecret = %q, want the JWT_SECRET value", JwtSecret)
	}
}
