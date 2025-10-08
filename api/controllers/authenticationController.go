package controllers

import (
	"API/api/dto"
	"API/api/services"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func HandleRegister(c *gin.Context) {
	var request dto.PlayerRegistrationRequest

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var statusCode, response = services.Register(request)

	c.JSON(statusCode, response)
}

func HandleLogin(c *gin.Context) {
	var request dto.PlayerLoginRequest
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
