package controllers

import (
	"API/api/dto/requests"
	"API/api/services"
	"API/authentication"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func HandleRegister(c *gin.Context) {
	var request requests.PlayerRegistrationRequest

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var statusCode, response = services.Register(request)

	c.JSON(statusCode, response)
}

func HandleLogin(c *gin.Context) {
	var request requests.PlayerLoginRequest
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var statusCode, response, cookie = services.Login(request)

	if cookie != nil {
		http.SetCookie(c.Writer, cookie)
	}

	c.JSON(statusCode, response)
}

func HandleLogout(c *gin.Context) {
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(c.Writer, cookie)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// requirePlayerID returns the player that AuthMiddleware authenticated. Handlers
// must take the player from here, never from the request. When there is no
// player it has already responded with 401, so the handler only has to return.
func requirePlayerID(c *gin.Context) (uint, bool) {
	playerID, ok := authentication.PlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
	}
	return playerID, ok
}

func GetPlayer(c *gin.Context) {
	playerID, ok := requirePlayerID(c)
	if !ok {
		return
	}

	var statusCode, response = services.PlayerByID(playerID)

	c.JSON(statusCode, response)
}
