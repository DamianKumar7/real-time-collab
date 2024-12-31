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
	ID string `json:"id"`
	Content string `json:"content"`
	CreatedAt string `json:"createdAt"`
	ModifiedAt string `json:"modifiedAt"`
}

type DocumentEvent struct{
	DocID     string    `json:"doc_id"`
    UserID    string    `json:"user_id"`
    Operation string    `json:"operation"` // JSON string of the operation
    Timestamp time.Time `json:"timestamp"`
}

