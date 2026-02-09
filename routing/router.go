package routing

import (
	"API/api/controllers"
	"API/authentication"
	"API/configuration"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"time"
)

func InitRouter() {
	s := &authentication.Server{}
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{configuration.GameURL, configuration.SiteURL},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, //todo restrict these
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	game := r.Group("/game")
	game.GET("/get_player", controllers.GetPlayer)
	game.GET("/get_player_buildings/:player_id/:map_id", controllers.GetPlayerBuildings)
	game.POST("/add_building", controllers.AddPlayerBuilding)

	game.GET("/map/:id", controllers.GetMap)

	game.GET("/tiles", controllers.GetTiles)

	game.GET("/buildings", controllers.GetBuildings)

	game.GET("/inventory/:player_id", controllers.GetPlayerInventory)

	r.POST("/register", controllers.HandleRegister)
	r.POST("/login", controllers.HandleLogin)
	r.POST("/logout", controllers.HandleLogout)

	r.GET("/news", controllers.GetNews)

	auth := r.Group("/").Use(s.AuthMiddleware())
	auth.GET("/maps", controllers.GetMaps)

	addr := ":5000"
	log.Printf("listening on %s", addr)
	err := r.Run(addr)
	if err != nil {
		return
	}
}
