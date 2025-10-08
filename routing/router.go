package routing

import (
	"API/api/controllers"
	"API/authentication"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"time"
)

func InitRouter() {
	s := &authentication.Server{}
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.POST("/register", controllers.HandleRegister)
	r.POST("/login", controllers.HandleLogin)
	r.POST("/logout", controllers.HandleLogout)

	auth := r.Group("/").Use(s.AuthMiddleware())
	auth.GET("/maps", controllers.GetMaps)
	auth.GET("/map/:id", controllers.GetMap)

	addr := ":5000"
	log.Printf("listening on %s", addr)
	err := r.Run(addr)
	if err != nil {
		return
	}
}
