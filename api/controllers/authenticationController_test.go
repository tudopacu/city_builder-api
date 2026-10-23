package controllers

import (
	"API/authentication"
	"API/models"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func newAuthenticationControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return newControllerTestDB(t, `CREATE TABLE players (
		id INTEGER PRIMARY KEY, username TEXT NOT NULL UNIQUE, email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL, created_at DATETIME, last_login_at DATETIME, status TEXT)`)
}

func seedAuthenticationPlayer(t *testing.T, db *gorm.DB) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO players (id, username, email, password, status)
		VALUES (?, ?, ?, ?, ?)`, 7, "builder", "builder@example.com", string(hash), "active").Error; err != nil {
		t.Fatal(err)
	}
}

func authenticationTestCookie(t *testing.T, subject string, expiresAt time.Time, secret []byte) *http.Cookie {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: subject, ExpiresAt: jwt.NewNumericDate(expiresAt),
	})
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: "auth_token", Value: signed}
}

func TestAuthenticationControllersRejectInvalidJSON(t *testing.T) {
	for _, endpoint := range []struct {
		name    string
		path    string
		handler gin.HandlerFunc
	}{
		{name: "register", path: "/register", handler: HandleRegister},
		{name: "login", path: "/login", handler: HandleLogin},
	} {
		for _, body := range []struct {
			name string
			json string
		}{
			{name: "empty", json: ""},
			{name: "malformed", json: `{"username":`},
			{name: "wrong field type", json: `{"username":42,"password":"secret"}`},
			{name: "array", json: `[]`},
		} {
			t.Run(endpoint.name+"/"+body.name, func(t *testing.T) {
				response := performControllerRequest(http.MethodPost, endpoint.path, endpoint.path, body.json, endpoint.handler)
				assertControllerError(t, response, http.StatusBadRequest, "invalid request")
				if len(response.Result().Cookies()) != 0 {
					t.Fatal("invalid request unexpectedly set a cookie")
				}
			})
		}
	}
}

func TestHandleRegisterCreatesPlayerWithoutExposingCredentials(t *testing.T) {
	db := newAuthenticationControllerTestDB(t)
	response := performControllerRequest(http.MethodPost, "/register", "/register",
		`{"username":"  builder  ","email":"builder@example.com","password":"correct-password"}`, HandleRegister)
	assertControllerJSON(t, response, http.StatusOK, `{"player":{"id":1,"username":"builder"}}`)

	var player models.Player
	if err := db.First(&player, 1).Error; err != nil {
		t.Fatal(err)
	}
	if player.Username != "builder" || player.Email != "builder@example.com" {
		t.Fatalf("stored player = %+v", player)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(player.Password), []byte("correct-password")); err != nil {
		t.Fatalf("password was not stored as a valid bcrypt hash: %v", err)
	}
}

func TestHandleRegisterPropagatesValidationErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{name: "empty request", body: `{}`, want: "Invalid request"},
		{name: "blank username", body: `{"username":"  ","email":"builder@example.com","password":"secret"}`, want: "Invalid request"},
		{name: "missing password", body: `{"username":"builder","email":"builder@example.com"}`, want: "Invalid request"},
		{name: "missing email", body: `{"username":"builder","password":"secret"}`, want: "Invalid request"},
		{name: "invalid email", body: `{"username":"builder","email":"not-an-email","password":"secret"}`, want: "Invalid email format"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := performControllerRequest(http.MethodPost, "/register", "/register", tc.body, HandleRegister)
			assertControllerError(t, response, http.StatusBadRequest, tc.want)
		})
	}
}

func TestHandleRegisterPropagatesStorageFailure(t *testing.T) {
	newControllerTestDB(t)
	response := performControllerRequest(http.MethodPost, "/register", "/register",
		`{"username":"builder","email":"builder@example.com","password":"secret"}`, HandleRegister)
	assertControllerError(t, response, http.StatusInternalServerError, "Failed to create player")
}

func TestHandleRegisterPropagatesPasswordHashFailure(t *testing.T) {
	body, err := json.Marshal(map[string]string{
		"username": "builder", "email": "builder@example.com", "password": strings.Repeat("p", 73),
	})
	if err != nil {
		t.Fatal(err)
	}
	response := performControllerRequest(http.MethodPost, "/register", "/register", string(body), HandleRegister)
	assertControllerError(t, response, http.StatusInternalServerError, "Internal error")
}

func TestHandleLoginReturnsPlayerAndUsableAuthenticationCookie(t *testing.T) {
	db := newAuthenticationControllerTestDB(t)
	seedAuthenticationPlayer(t, db)
	before := time.Now()
	response := performControllerRequest(http.MethodPost, "/login", "/login",
		`{"username":"builder","password":"correct-password"}`, HandleLogin)
	after := time.Now()
	assertControllerJSON(t, response, http.StatusOK, `{"player":{"id":7,"username":"builder"}}`)
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %v, want one authentication cookie", cookies)
	}
	cookie := cookies[0]
	if cookie.Name != "auth_token" || cookie.Value == "" || cookie.Path != "/" || cookie.MaxAge != 24*60*60 {
		t.Fatalf("unexpected authentication cookie: %v", cookie)
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("cookie SameSite = %v, want Lax", cookie.SameSite)
	}
	if cookie.Expires.Before(before.Add(24*time.Hour).Truncate(time.Second)) || cookie.Expires.After(after.Add(24*time.Hour)) {
		t.Errorf("cookie expiry = %v, want 24 hours from login", cookie.Expires)
	}
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (any, error) {
		return authentication.JwtSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		t.Fatalf("login did not return a valid signed token: %v", err)
	}
	if claims.Subject != "7" || claims.ExpiresAt == nil || claims.IssuedAt == nil {
		t.Fatalf("unexpected token claims: %+v", claims)
	}
	if !claims.ExpiresAt.Time.Equal(cookie.Expires) {
		t.Errorf("token expiry = %v, cookie expiry = %v", claims.ExpiresAt.Time, cookie.Expires)
	}
	playerResponse := performControllerRequest(http.MethodGet, "/game/get_player", "/game/get_player", "", GetPlayer, cookie)
	assertControllerJSON(t, playerResponse, http.StatusOK, `{"player":{"id":7,"username":"builder"}}`)
}

func TestHandleLoginRejectsInvalidCredentialsWithoutSettingCookie(t *testing.T) {
	db := newAuthenticationControllerTestDB(t)
	seedAuthenticationPlayer(t, db)
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "unknown username", body: `{"username":"unknown","password":"correct-password"}`},
		{name: "wrong password", body: `{"username":"builder","password":"wrong-password"}`},
		{name: "missing credentials", body: `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := performControllerRequest(http.MethodPost, "/login", "/login", tc.body, HandleLogin)
			assertControllerError(t, response, http.StatusUnauthorized, "Invalid credentials")
			if len(response.Result().Cookies()) != 0 {
				t.Fatal("failed login unexpectedly set a cookie")
			}
		})
	}
}

func TestHandleLogoutExpiresAuthenticationCookie(t *testing.T) {
	response := performControllerRequest(http.MethodPost, "/logout", "/logout", "", HandleLogout,
		&http.Cookie{Name: "auth_token", Value: "old-token"})
	assertControllerJSON(t, response, http.StatusOK, `{"message":"logged out"}`)
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %v, want one expired authentication cookie", cookies)
	}
	cookie := cookies[0]
	if cookie.Name != "auth_token" || cookie.Value != "" || cookie.Path != "/" || cookie.MaxAge != -1 || !cookie.HttpOnly {
		t.Fatalf("unexpected logout cookie: %v", cookie)
	}
	if !cookie.Expires.Equal(time.Unix(0, 0)) {
		t.Errorf("logout cookie expiry = %v, want Unix epoch", cookie.Expires)
	}
}

func TestGetPlayerRequiresAuthenticationCookie(t *testing.T) {
	response := performControllerRequest(http.MethodGet, "/game/get_player", "/game/get_player", "", GetPlayer)
	assertControllerError(t, response, http.StatusUnauthorized, "missing auth token")
}

func TestGetPlayerRejectsInvalidAuthenticationCookies(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cookie *http.Cookie
	}{
		{name: "malformed", cookie: &http.Cookie{Name: "auth_token", Value: "not-a-token"}},
		{name: "empty", cookie: &http.Cookie{Name: "auth_token"}},
		{name: "expired", cookie: authenticationTestCookie(t, "7", time.Now().Add(-time.Hour), authentication.JwtSecret)},
		{name: "wrong signature", cookie: authenticationTestCookie(t, "7", time.Now().Add(time.Hour), []byte("different-test-secret"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := performControllerRequest(http.MethodGet, "/game/get_player", "/game/get_player", "", GetPlayer, tc.cookie)
			assertControllerError(t, response, http.StatusUnauthorized, "Invalid token")
		})
	}
}

func TestGetPlayerReturnsNotFoundForMissingPlayer(t *testing.T) {
	newAuthenticationControllerTestDB(t)
	cookie := authenticationTestCookie(t, "999", time.Now().Add(time.Hour), authentication.JwtSecret)
	response := performControllerRequest(http.MethodGet, "/game/get_player", "/game/get_player", "", GetPlayer, cookie)
	assertControllerError(t, response, http.StatusNotFound, "Player not found")
}
