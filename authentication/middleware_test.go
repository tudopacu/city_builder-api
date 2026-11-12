package authentication

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func useTestJwtSecret(t *testing.T) []byte {
	t.Helper()
	previous := JwtSecret
	JwtSecret = []byte("middleware-test-secret-0123456789abcdef")
	t.Cleanup(func() { JwtSecret = previous })
	return JwtSecret
}

func signTestToken(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims, key any) string {
	t.Helper()
	signed, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func unexpiredClaims(subject string) jwt.MapClaims {
	return jwt.MapClaims{"sub": subject, "exp": time.Now().Add(time.Hour).Unix()}
}

func newTestRequest(token string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if token != "" {
		request.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	}
	return request
}

// performProtectedRequest runs the middleware in front of a handler that
// reports the player ID it was given.
func performProtectedRequest(request *http.Request) *httptest.ResponseRecorder {
	router := gin.New()
	router.GET("/protected", (&Server{}).AuthMiddleware(), func(c *gin.Context) {
		id, ok := PlayerID(c)
		c.JSON(http.StatusOK, gin.H{"player_id": id, "ok": ok})
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestAuthMiddlewareAcceptsValidTokens(t *testing.T) {
	token := signTestToken(t, jwt.SigningMethodHS256, unexpiredClaims("7"), useTestJwtSecret(t))
	bearer := httptest.NewRequest(http.MethodGet, "/protected", nil)
	bearer.Header.Set("Authorization", "Bearer "+token)
	for name, request := range map[string]*http.Request{"cookie": newTestRequest(token), "bearer header": bearer} {
		t.Run(name, func(t *testing.T) {
			response := performProtectedRequest(request)
			var body struct {
				PlayerID uint `json:"player_id"`
				OK       bool `json:"ok"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || response.Code != http.StatusOK || !body.OK || body.PlayerID != 7 {
				t.Fatalf("response = %d %s, want player 7", response.Code, response.Body.String())
			}
		})
	}
}

func TestAuthMiddlewareRejectsInvalidTokens(t *testing.T) {
	secret := useTestJwtSecret(t)
	expired := jwt.MapClaims{"sub": "7", "exp": time.Now().Add(-time.Hour).Unix()}
	for _, tc := range []struct {
		name  string
		token string
		error string
	}{
		{name: "missing", token: "", error: "unauthenticated"},
		{name: "malformed", token: "not-a-token", error: "invalid token"},
		{name: "wrong secret", token: signTestToken(t, jwt.SigningMethodHS256, unexpiredClaims("7"), []byte("a-different-secret-0123456789abcdef")), error: "invalid token"},
		{name: "previously hardcoded secret", token: signTestToken(t, jwt.SigningMethodHS256, unexpiredClaims("7"), []byte("replace-with-a-strong-secret")), error: "invalid token"},
		{name: "expired", token: signTestToken(t, jwt.SigningMethodHS256, expired, secret), error: "invalid token"},
		{name: "no expiry", token: signTestToken(t, jwt.SigningMethodHS256, jwt.MapClaims{"sub": "7"}, secret), error: "invalid token"},
		{name: "other HMAC algorithm", token: signTestToken(t, jwt.SigningMethodHS512, unexpiredClaims("7"), secret), error: "invalid token"},
		{name: "unsigned", token: signTestToken(t, jwt.SigningMethodNone, unexpiredClaims("7"), jwt.UnsafeAllowNoneSignatureType), error: "invalid token"},
		{name: "missing subject", token: signTestToken(t, jwt.SigningMethodHS256, jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix()}, secret), error: "invalid token subject"},
		{name: "SQL after player ID", token: signTestToken(t, jwt.SigningMethodHS256, unexpiredClaims("7 UNION SELECT id, password FROM players -- "), secret), error: "invalid user id in token"},
		{name: "letters after player ID", token: signTestToken(t, jwt.SigningMethodHS256, unexpiredClaims("7abc"), secret), error: "invalid user id in token"},
		{name: "negative player ID", token: signTestToken(t, jwt.SigningMethodHS256, unexpiredClaims("-7"), secret), error: "invalid user id in token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := performProtectedRequest(newTestRequest(tc.token))
			var body struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || response.Code != http.StatusUnauthorized || body.Error != tc.error {
				t.Fatalf("response = %d %s, want 401 %q", response.Code, response.Body.String(), tc.error)
			}
		})
	}
}

func TestAuthMiddlewareRejectsTokensWhenSecretIsNotConfigured(t *testing.T) {
	previous := JwtSecret
	JwtSecret = nil
	t.Cleanup(func() { JwtSecret = previous })
	// An unconfigured server would otherwise verify this token with an empty key.
	token := signTestToken(t, jwt.SigningMethodHS256, unexpiredClaims("7"), []byte{})
	response := performProtectedRequest(newTestRequest(token))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("response = %d %s, want 401", response.Code, response.Body.String())
	}
}

func TestPlayerIDIsMissingWithoutMiddleware(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if id, ok := PlayerID(c); ok || id != 0 {
		t.Fatalf("PlayerID() = (%d, %v), want (0, false)", id, ok)
	}
}
