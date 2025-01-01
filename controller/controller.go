package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"real-time-collab/config"
	"real-time-collab/models"
	"real-time-collab/services"
	"real-time-collab/utils"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

var upgradeConnection = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},	
}

type SuccessResponse[T any] struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    T  			`json:"data,omitempty"`
}

func SendJSONResponse(w http.ResponseWriter, status int, payload interface{}) {
	// Always set headers before writing the status code
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // For CORS, if needed
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(payload)
	
	if err != nil {
		// Log the error server-side
		log.Printf("Error encoding JSON response: %v", err)
		// If JSON encoding fails, send a plain text error
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to generate JSON response"))
		return
	}
}

func SendErrorResponse(w http.ResponseWriter, status int, message string) {
	errorResponse := ErrorResponse{
		Status:  "error",
		Message: message,
	}
	SendJSONResponse(w, status, errorResponse)
}

func RegisterUser(w http.ResponseWriter, r *http.Request, DB *gorm.DB){
	var user models.User


	//parse the request body and decode it into the User struct
	err:= json.NewDecoder(r.Body).Decode(&user)

	if(err!= nil){
		SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if(user.Username == "" || user.Email == "" || user.Password == ""){
		SendErrorResponse(w, http.StatusBadRequest, "All fields are required")
        return
	}

	exists,err := services.IsUserPresent(&user, DB, user.Email)
	if(err!= nil){
		SendErrorResponse(w, http.StatusInternalServerError, "error trying to get user from DB")
		return
	}

	if(exists){
		SendErrorResponse(w, http.StatusBadRequest, "User already exists in DB")
		return
	}

	hashedPassword,err := bcrypt.GenerateFromPassword([]byte(user.Password),bcrypt.DefaultCost)

	if(err!= nil){
		http.Error(w,err.Error(),http.StatusInternalServerError)
        return
	}

	user.Password = string(hashedPassword)
	res := DB.Create(&user)

	if(res.Error != nil){
		http.Error(w,res.Error.Error(),http.StatusInternalServerError)
		return
	}
	successResponse := SuccessResponse[map[string]interface{}]{
		Status:  "success",
		Message: "User registered successfully",
		Data: map[string]interface{}{
			"username": user.Username,
			"email": user.Email,
		},
	}
	SendJSONResponse(w, http.StatusCreated, successResponse)
}

func LoginUser(w http.ResponseWriter, r *http.Request, DB *gorm.DB){

	var user models.User

	err:= json.NewDecoder(r.Body).Decode(&user)

	if(err!= nil){
		SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if(user.Email == "" || user.Password == ""){
		SendErrorResponse(w,http.StatusBadRequest,"Please enter credentials")
		return
	}

	var userFromDb models.User

	exists,err := services.FindUserByUsername(&userFromDb,DB,user.Username)

	if(err!= nil){
		SendErrorResponse(w,http.StatusInternalServerError,"error occured while trying to fetch the DB")
		return
	}
	if(!exists){
		SendErrorResponse(w,http.StatusBadRequest,"user does not exist")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(userFromDb.Password),[]byte(user.Password))

	if(err!= nil){
		SendErrorResponse(w,http.StatusBadRequest,"Pssword Doesnt match")
		return
	}

	jwtToken,err := utils.GenerateJWT(userFromDb.ID,userFromDb.Email)

	if(err!= nil){
		SendErrorResponse(w,http.StatusInternalServerError,"error generating jwt")
	}

	SendJSONResponse(w,http.StatusAccepted,map[string]string{"token":jwtToken})

}

func ValidateJwtToken(w http.ResponseWriter, r *http.Request) {
    // Step 1: Retrieve the Authorization header
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        SendErrorResponse(w, http.StatusUnauthorized, "Authorization header is missing")
        return 
    }

    // Step 2: Extract the JWT token from the Authorization header
    const bearerPrefix = "Bearer "
    if !strings.HasPrefix(authHeader, bearerPrefix) {
        SendErrorResponse(w, http.StatusUnauthorized, "Invalid Authorization header format")
        return 
    }

    JwtToken := strings.TrimPrefix(authHeader, bearerPrefix)
    if JwtToken == "" {
        SendErrorResponse(w, http.StatusUnauthorized, "JWT token is missing")
        return 
    }

    claims, err := utils.ExtractClaims(JwtToken)
    if err != nil {
        log.Printf("Error extracting claims: %v", err)
        SendErrorResponse(w, http.StatusUnauthorized, "Invalid or expired token")
        return 
    }


	if expiryTime,exists:= claims["exp"].(float64);exists{
		exp := time.Unix(int64(expiryTime),0)
		if(time.Now().After(exp)){
			SendErrorResponse(w, http.StatusBadRequest, "token expired")
			return 
		}
	}

    log.Printf("Extracted claims: %+v", claims)
	SendJSONResponse(w, http.StatusOK, "token is valid")
}

func HandleWebSocketConnection(w http.ResponseWriter, r *http.Request, pool *config.ConnectionPool, DB *gorm.DB){

	// if(ValidateJwtToken(w,r)){
	// 	log.Printf("jwt token is expired")
	// 	return
	// }
	
	connection, err := upgradeConnection.Upgrade(w, r ,nil)
	if err != nil{
		log.Printf("connection refused")
		return 
	}

	pool.Mutex.Lock()
	pool.Connections[connection] = true
	pool.Mutex.Unlock()

	go pool.ReadMessage(connection, DB)
}
