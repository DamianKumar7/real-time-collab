package utils

import (
	"fmt"
	"real-time-collab/models"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

var jwtSecret = []byte("real_time_collab")


func GenerateJWT(userID uint, email string) (string, error) {
	// Define the token claims
	claims := jwt.MapClaims{
		"sub":  userID,                    // Subject (user ID)
		"email": email,                    // User email
		"exp":   time.Now().Add(time.Hour * 24).Unix(), // Expiration time (24 hours)
		"iat":   time.Now().Unix(),        // Issued at
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret key
	signedToken, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}


func ExtractClaims(tokenString string) (map[string]interface{}, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	// Extract claims if the token is valid
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func AutoMigrateModels(DB *gorm.DB){
	DB.AutoMigrate(&models.DocumentEvent{})
	DB.AutoMigrate(&models.Document{})
	DB.AutoMigrate(&models.User{})
}