package models

import (
	"time"
)

type PlayerStatus string

const (
	Active    PlayerStatus = "active"
	Banned    PlayerStatus = "banned"
	Suspended PlayerStatus = "suspended"
)

type Player struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Username    string    `gorm:"size:255;not null;unique" json:"username"`
	Email       string    `gorm:"size:255;not null;unique" json:"email"`
	Password    string    `gorm:"size:255;not null" json:"password"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	LastLoginAt time.Time `gorm:"autoUpdateTime" json:"last_login_at"`
	Status      string    `gorm:"type:enum('active','banned','suspended');not null;default:'active'"`
}
