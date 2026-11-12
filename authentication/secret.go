package authentication

import (
	"fmt"
	"log"
	"os"
)

// HS256 needs a key at least as long as its 32-byte hash (RFC 7518, section 3.2).
const minJwtSecretLength = 32

// JwtSecret signs and verifies auth tokens. InitJwtSecret loads it at startup.
var JwtSecret []byte

func InitJwtSecret() {
	secret, err := loadJwtSecret()
	if err != nil {
		log.Fatalf("failed to load jwt secret: %v", err)
	}
	JwtSecret = secret
}

func loadJwtSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < minJwtSecretLength {
		return nil, fmt.Errorf("JWT_SECRET must be set to at least %d bytes", minJwtSecretLength)
	}
	return []byte(secret), nil
}
