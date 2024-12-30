package models

import "github.com/jinzhu/gorm"

type User struct {
	gorm.Model
	//gorm.Model proviedes fields like Id created At updatedAt and deletedAt
	//gorm: unique enforces that a particular field is unique
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email" gorm:"unique"`
}