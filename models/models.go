package models

import (
	"time"
	"github.com/jinzhu/gorm"
)

type User struct {
	gorm.Model
	//gorm.Model proviedes fields like Id created At updatedAt and deletedAt
	//gorm: unique enforces that a particular field is unique
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email" gorm:"unique"`
}


type Document struct{
	gorm.Model
	Content string `json:"content"`
	Version int `json:"version"`
	Title string `json:"title"`
}

type DocumentEvent struct{
	gorm.Model
	DocID     string    `json:"doc_id"`
    UserID    string    `json:"user_id"`
    Operation string    `json:"operation"`
    Timestamp time.Time `json:"timestamp"`
	Position  int 		`json:"position"`
	Length     int 		`json:"length"`
	Content	 string 	`json:"content"`
	Version int 	`json:"doc_version"` 
	Title string    `json:"title"`
}



