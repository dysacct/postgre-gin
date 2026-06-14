package models

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey"`
	Username     string    `gorm:"unique;not null;size:50"`
	PasswordHash string    `gorm:"column:password_hash;not null"`
	Role         string    `gorm:"size:20;default:user"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

func (User) TableName() string {
	return "users"
}
