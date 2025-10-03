package main

import (
	"API/database"
	"API/routing"
)

func main() {
	database.InitDB()
	routing.InitRouter()
}
