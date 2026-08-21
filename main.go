package main

import (
	"API/cron"
	"API/database"
	"API/redis"
	"API/routing"
	"context"
)

func main() {
	database.InitDB()
	redis.InitRedis()
	cron.StartCronJobs(context.Background())
	routing.InitRouter()
}
