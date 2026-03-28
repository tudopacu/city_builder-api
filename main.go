package main

import (
	"API/database"
	"API/redis"
	"API/routing"
)

func main() {
	database.InitDB()
	redis.InitRedis()
	routing.InitRouter()
}
