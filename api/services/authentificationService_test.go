package services

import (
	"API/api/dto"
	"API/api/dto/requests"
	"API/authentication"
	"API/models"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func newAuthenticationServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return newServiceTestDB(t, `CREATE TABLE players (
		id INTEGER PRIMARY KEY, username TEXT NOT NULL UNIQUE, email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL, created_at DATETIME, last_login_at DATETIME, status TEXT)`)
}

func seedAuthenticationServicePlayer(t *testing.T, db *gorm.DB) {
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

func useAuthenticationServiceTestSecret(t *testing.T) {
	t.Helper()
	previous := authentication.JwtSecret
	authentication.JwtSecret = []byte("service-test-secret-0123456789abcdef")
	t.Cleanup(func() { authentication.JwtSecret = previous })
}

func TestRegisterCreatesPlayerWithTrimmedUsernameAndHashedPassword(t *testing.T) {
	db := newAuthenticationServiceTestDB(t)
	status, response := Register(requests.PlayerRegistrationRequest{
		Username: "  builder  ", Email: "builder@example.com", Password: "correct-password",
	})
	if status != http.StatusOK || response.Error != "" || !reflect.DeepEqual(response.Player, &dto.Player{Id: 1, Username: "builder"}) {
		t.Fatalf("Register = (%d, %+v)", status, response)
	}
	var player models.Player
	if err := db.First(&player, 1).Error; err != nil {
		t.Fatal(err)
	}
	if player.Username != "builder" || player.Email != "builder@example.com" || player.Status != "active" {
		t.Fatalf("stored player has unexpected profile: %+v", player)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(player.Password), []byte("correct-password")); err != nil {
		t.Fatalf("stored password is not a valid bcrypt hash: %v", err)
	}
	if cost, err := bcrypt.Cost([]byte(player.Password)); err != nil || cost != bcrypt.DefaultCost {
		t.Fatalf("password hash cost = %d, err = %v, want %d", cost, err, bcrypt.DefaultCost)
	}
}

func TestRegisterRejectsInvalidInputsWithoutCreatingPlayer(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request requests.PlayerRegistrationRequest
		status  int
		error   string
	}{
		{"empty request", requests.PlayerRegistrationRequest{}, http.StatusBadRequest, "Invalid request"},
		{"blank username", requests.PlayerRegistrationRequest{Username: " \t", Email: "builder@example.com", Password: "secret"}, http.StatusBadRequest, "Invalid request"},
		{"missing email", requests.PlayerRegistrationRequest{Username: "builder", Password: "secret"}, http.StatusBadRequest, "Invalid request"},
		{"missing password", requests.PlayerRegistrationRequest{Username: "builder", Email: "builder@example.com"}, http.StatusBadRequest, "Invalid request"},
		{"invalid email", requests.PlayerRegistrationRequest{Username: "builder", Email: "not-an-email", Password: "secret"}, http.StatusBadRequest, "Invalid email format"},
		{"unhashable password", requests.PlayerRegistrationRequest{Username: "builder", Email: "builder@example.com", Password: strings.Repeat("p", 73)}, http.StatusInternalServerError, "Internal error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newAuthenticationServiceTestDB(t)
			status, response := Register(tc.request)
			if status != tc.status || response.Error != tc.error || response.Player != nil {
				t.Fatalf("Register = (%d, %+v), want (%d, %q)", status, response, tc.status, tc.error)
			}
			var count int64
			if err := db.Model(&models.Player{}).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("invalid registration created %d players", count)
			}
		})
	}
}

func TestRegisterReturnsStorageErrors(t *testing.T) {
	for _, tc := range []struct {
		name      string
		duplicate bool
		status    int
		error     string
	}{
		{"database unavailable", false, http.StatusInternalServerError, "Failed to create player"},
		{"duplicate player", true, http.StatusConflict, "Player already exists"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newAuthenticationServiceTestDB(t)
			if tc.duplicate {
				// Production uses MySQL; inject its duplicate-key error at the
				// persistence boundary because SQLite reports a different error.
				if err := db.Callback().Create().Before("gorm:create").Register("test:duplicate_player", func(tx *gorm.DB) {
					if tx.Statement.Table == "players" {
						tx.AddError(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'builder' for key 'players.username'"})
					}
				}); err != nil {
					t.Fatal(err)
				}
			} else {
				execServiceTestSQL(t, db, `DROP TABLE players`)
			}
			status, response := Register(requests.PlayerRegistrationRequest{
				Username: "builder", Email: "builder@example.com", Password: "correct-password",
			})
			if status != tc.status || response.Error != tc.error || response.Player != nil {
				t.Fatalf("Register = (%d, %+v), want (%d, %q)", status, response, tc.status, tc.error)
			}
		})
	}
}

func TestLoginReturnsPlayerAndSignedExpiringCookie(t *testing.T) {
	useAuthenticationServiceTestSecret(t)
	db := newAuthenticationServiceTestDB(t)
	seedAuthenticationServicePlayer(t, db)
	before := time.Now()
	status, response, cookie := Login(requests.PlayerLoginRequest{Username: "builder", Password: "correct-password"})
	after := time.Now()
	if status != http.StatusOK || response.Error != "" || !reflect.DeepEqual(response.Player, &dto.Player{Id: 7, Username: "builder"}) {
		t.Fatalf("Login = (%d, %+v)", status, response)
	}
	if cookie == nil || cookie.Name != "auth_token" || cookie.Path != "/" || cookie.Value == "" || cookie.MaxAge != 86400 || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected login cookie: %v", cookie)
	}
	if cookie.Expires.Before(before.Add(24*time.Hour)) || cookie.Expires.After(after.Add(24*time.Hour)) {
		t.Fatalf("cookie expiry = %v, want 24 hours after login", cookie.Expires)
	}
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (any, error) {
		return authentication.JwtSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		t.Fatalf("invalid signed login token: %v", err)
	}
	if claims.Subject != "7" || claims.ExpiresAt == nil || claims.IssuedAt == nil {
		t.Fatalf("unexpected token claims: %+v", claims)
	}
	if !claims.ExpiresAt.Time.Equal(cookie.Expires.Truncate(time.Second)) {
		t.Fatalf("token expiry = %v, cookie expiry = %v", claims.ExpiresAt.Time, cookie.Expires)
	}
	if claims.IssuedAt.Time.Before(before.Truncate(time.Second)) || claims.IssuedAt.Time.After(after) {
		t.Fatalf("token issued time = %v, outside login interval", claims.IssuedAt.Time)
	}
}

func TestLoginRejectsInvalidCredentialsWithoutCookie(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request requests.PlayerLoginRequest
		setup   string
	}{
		{"unknown user", requests.PlayerLoginRequest{Username: "unknown", Password: "correct-password"}, ""},
		{"wrong password", requests.PlayerLoginRequest{Username: "builder", Password: "wrong-password"}, ""},
		{"empty credentials", requests.PlayerLoginRequest{}, ""},
		{"invalid stored hash", requests.PlayerLoginRequest{Username: "builder", Password: "correct-password"}, `UPDATE players SET password = 'invalid-bcrypt-hash'`},
		{"database failure", requests.PlayerLoginRequest{Username: "builder", Password: "correct-password"}, `DROP TABLE players`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newAuthenticationServiceTestDB(t)
			seedAuthenticationServicePlayer(t, db)
			if tc.setup != "" {
				execServiceTestSQL(t, db, tc.setup)
			}
			status, response, cookie := Login(tc.request)
			if status != http.StatusUnauthorized || response.Error != "Invalid credentials" || response.Player != nil || cookie != nil {
				t.Fatalf("Login = (%d, %+v, %v), want invalid credentials without player or cookie", status, response, cookie)
			}
		})
	}
}

func TestPlayerByIDReturnsMatchingPlayer(t *testing.T) {
	db := newAuthenticationServiceTestDB(t)
	seedAuthenticationServicePlayer(t, db)
	execServiceTestSQL(t, db, `INSERT INTO players (id, username, email, password) VALUES (1, 'other', 'other@example.com', 'unused')`)
	status, response := PlayerByID(7)
	if status != http.StatusOK || response.Error != "" || !reflect.DeepEqual(response.Player, &dto.Player{Id: 7, Username: "builder"}) {
		t.Fatalf("PlayerByID = (%d, %+v)", status, response)
	}
}

func TestPlayerByIDReturnsMissingPlayerError(t *testing.T) {
	for _, unavailable := range []bool{false, true} {
		name := "unknown player"
		if unavailable {
			name = "database failure"
		}
		t.Run(name, func(t *testing.T) {
			db := newAuthenticationServiceTestDB(t)
			if unavailable {
				execServiceTestSQL(t, db, `DROP TABLE players`)
			}
			status, response := PlayerByID(999)
			if status != http.StatusNotFound || response.Error != "Player not found" || response.Player != nil {
				t.Fatalf("PlayerByID = (%d, %+v), want missing player", status, response)
			}
		})
	}
}
