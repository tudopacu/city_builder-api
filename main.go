package main

import (
	"API/authentication"
	"API/database"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
)

var DB *gorm.DB

func main() {

	DB = database.GetDB()

	s := &authentication.Server{Db: DB}
	r := gin.Default()

	r.POST("/register", s.HandleRegister)
	r.POST("/login", s.HandleLogin)
	r.POST("/logout", s.HandleLogout)
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "it works")
	})

	auth := r.Group("/").Use(s.AuthMiddleware())
	auth.GET("/me", s.HandleMe)

	addr := ":5000"
	log.Printf("listening on %s", addr)
	r.Run(addr)
}
