package database

import (
	"API/configuration"
	"log"
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	once sync.Once
)

var DB *gorm.DB

func InitDB() {
	once.Do(func() {
		host := configuration.MustGetEnv("DB_HOST")
		dbName := configuration.MustGetEnv("DB_NAME")
		user := configuration.MustGetEnv("DB_USER")
		password := configuration.MustGetEnv("DB_PASSWORD")

		dsn := user + ":" + password + "@tcp(" + host + ")/" + dbName + "?charset=utf8mb4&parseTime=True&loc=Local"

		conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to connect to database: %v", err)
		}

		db = conn
	})

	DB = db
}
