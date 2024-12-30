package services

import (
	"real-time-collab/models"
	"gorm.io/gorm"
)

func IsUserPresent(user *models.User, DB *gorm.DB, email string) (bool, error) {
	result := DB.Where("email = ?", email).First(user)
    
    if result.Error == gorm.ErrRecordNotFound {
        return false, nil  
    }
    
    if result.Error != nil {
        return false, result.Error  
    }
    
    return true, nil
}

func FindUserByUsername(user *models.User, DB *gorm.DB, username string) (bool,error){
	result := DB.Where("username = ?", username).First(user)
    
    if result.Error == gorm.ErrRecordNotFound {
        return false, nil  
    }
    
    if result.Error != nil {
        return false, result.Error  
    }
    
    return true, nil
}