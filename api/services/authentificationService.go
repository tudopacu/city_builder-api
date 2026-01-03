package services

import (
	"API/api/dto"
	"API/api/dto/requests"
	"API/api/dto/responses"
	"API/authentication"
	"API/database"
	"API/models"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

func Register(request requests.PlayerRegistrationRequest) (int, responses.PlayerRegistrationResponse) {
	request.Username = strings.TrimSpace(request.Username)
	if request.Username == "" || request.Password == "" || request.Email == "" {
		return http.StatusBadRequest, responses.PlayerRegistrationResponse{Error: "Invalid request"}
	}

	_, err := mail.ParseAddress(request.Email)

	if err != nil {
		return http.StatusBadRequest, responses.PlayerRegistrationResponse{Error: "Invalid email format"}
	}

	pwHash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return http.StatusInternalServerError, responses.PlayerRegistrationResponse{Error: "Internal error"}
	}

	player := &models.Player{
		Username: request.Username,
		Password: string(pwHash),
		Email:    request.Email,
	}

	if err := database.DB.Create(player).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			return http.StatusConflict, responses.PlayerRegistrationResponse{Error: "Player already exists"}
		}
		return http.StatusInternalServerError, responses.PlayerRegistrationResponse{Error: "Failed to create player"}
	}

	return http.StatusOK, responses.PlayerRegistrationResponse{Player: &dto.Player{Id: player.ID, Username: player.Username}}
}

func Login(request requests.PlayerLoginRequest) (int, responses.PlayerLoginResponse, *http.Cookie) {
	var player models.Player

	if err := database.DB.Where("username = ?", request.Username).First(&player).Error; err != nil {
		return http.StatusUnauthorized, responses.PlayerLoginResponse{Error: "Invalid credentials"}, nil
	}

	if err := bcrypt.CompareHashAndPassword([]byte(player.Password), []byte(request.Password)); err != nil {
		return http.StatusUnauthorized, responses.PlayerLoginResponse{Error: "Invalid credentials"}, nil
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", player.ID),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(authentication.JwtSecret)
	if err != nil {
		return http.StatusInternalServerError, responses.PlayerLoginResponse{Error: "Failed to create token"}, nil
	}

	//todo this cookie needs to change on https and in production, set Secure to true and SAmeSite to proper value
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    signed,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600 * 24,
	}

	return http.StatusOK, responses.PlayerLoginResponse{Player: &dto.Player{Id: player.ID, Username: player.Username}}, cookie
}

func PlayerByCookie(cookie *http.Cookie) (int, responses.PlayerByCookieResponse) {
	token, err := jwt.ParseWithClaims(cookie.Value, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return authentication.JwtSecret, nil
	})

	if err != nil || !token.Valid {
		return http.StatusUnauthorized, responses.PlayerByCookieResponse{Error: "Invalid token"}
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return http.StatusUnauthorized, responses.PlayerByCookieResponse{Error: "Invalid token claims"}
	}

	var player models.Player
	if err := database.DB.First(&player, claims.Subject).Error; err != nil {
		return http.StatusNotFound, responses.PlayerByCookieResponse{Error: "Player not found"}
	}

	return http.StatusOK, responses.PlayerByCookieResponse{Player: &dto.Player{Id: player.ID, Username: player.Username}}
}
