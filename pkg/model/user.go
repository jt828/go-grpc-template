package model

import "time"

func (dataEntity *UserDataEntity) ToDomain() User {
	return User(*dataEntity)
}

type UserDataEntity struct {
	Id        int64     `gorm:"column:id"`
	Email     string    `gorm:"column:email"`
	Username  string    `gorm:"column:username"`
	Password  string    `gorm:"column:password"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (dataEntity *UserDataEntity) TableName() string {
	return "main.users"
}

type User struct {
	Id        int64
	Email     string
	Username  string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
