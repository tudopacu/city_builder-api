package main

import (
	"API/authentication"
	"API/cron"
	"API/database"
	"API/redis"
	"API/routing"
	"context"
)

func main() {
	authentication.InitJwtSecret()
	database.InitDB()
	redis.InitRedis()
	cron.StartCronJobs(context.Background())
	routing.InitRouter()
}
