package database

import (
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
		dsn := "root:Hamham1miau!@tcp(127.0.0.1:3306)/game?charset=utf8mb4&parseTime=True&loc=Local"

		conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to connect to database: %v", err)
		}

		db = conn
	})

	DB = db
}
