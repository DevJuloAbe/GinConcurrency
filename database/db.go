package database

import (
	"gingonic-concurrency/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Connect(dsn string) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
	})
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&model.User{})
}
