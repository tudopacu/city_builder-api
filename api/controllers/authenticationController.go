package controllers

import (
	"API/api/dto/requests"
	"API/api/services"
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

func GetPlayer(c *gin.Context) {
	tokenCookie, err := c.Request.Cookie("auth_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing auth token"})
		return
	}

	var statusCode, response = services.PlayerByCookie(tokenCookie)

	c.JSON(statusCode, response)
}
