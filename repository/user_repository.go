package repository

import (
	"context"

	"gingonic-concurrency/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUsers(users []model.User) error {
	return r.CreateUsersWithContext(context.Background(), users)
}

func (r *UserRepository) CreateUsersWithContext(ctx context.Context, users []model.User) error {
	return r.db.WithContext(ctx).CreateInBatches(users, len(users)).Error
}

func (r *UserRepository) FetchUsers(limit int, offset int) ([]model.User, error) {
	return r.FetchUsersWithContext(context.Background(), limit, offset)
}

func (r *UserRepository) FetchUsersWithContext(ctx context.Context, limit int, offset int) ([]model.User, error) {
	var users []model.User

	err := r.db.WithContext(ctx).
		Order("id ASC").
		Limit(limit).
		Offset(offset).
		Find(&users).
		Error

	return users, err
}
