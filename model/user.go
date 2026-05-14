package model

import "time"

type User struct {
	ID           uint `gorm:"primaryKey"`
	GameID       uint `gorm:"not null"`
	IsVerified   bool `gorm:"not null"`
	VerifiedAt   *time.Time
	OTP          string `gorm:"column:otp;size:8;not null"`
	OTPExpiresAt time.Time
	Name         string `gorm:"size:100"`
	Password     string `gorm:"size:100"`
	Phone        string `gorm:"size:20"`
	Gender       string `gorm:"size:10"`
	Address      string `gorm:"size:255"`
	Email        string `gorm:"size:150;uniqueIndex"`
	Role         string `gorm:"size:50"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
