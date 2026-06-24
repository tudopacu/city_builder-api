package database

import (
	"log"
	"os"
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
		host := os.Getenv("DB_HOST")
		dbName := os.Getenv("DB_NAME")
		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")

		dsn := user + ":" + password + "@tcp(" + host + ")/" + dbName + "?charset=utf8mb4&parseTime=True&loc=Local"

		conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to connect to database: %v", err)
		}

		db = conn
	})

	DB = db
}
