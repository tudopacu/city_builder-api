package routing

import (
	"API/api/controllers"
	"API/authentication"
	"github.com/gin-gonic/gin"
	"log"
)

func InitRouter() {
	s := &authentication.Server{}
	r := gin.Default()

	r.POST("/register", controllers.HandleRegister)
	r.POST("/login", controllers.HandleLogin)
	r.POST("/logout", controllers.HandleLogout)

	auth := r.Group("/").Use(s.AuthMiddleware())
	auth.GET("/maps", controllers.GetMaps)
	auth.GET("/map/:id", controllers.GetMap)

	addr := ":5000"
	log.Printf("listening on %s", addr)
	r.Run(addr)
}
