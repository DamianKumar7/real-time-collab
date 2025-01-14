package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"real-time-collab/config"
	"real-time-collab/models"
	"real-time-collab/services"
	"real-time-collab/utils"
	"strconv"
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
	w.Header().Set("Content-Type", "*")
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
	log.Println("Inside register user method")
	//parse the request body and decode it into the User struct
	err:= json.NewDecoder(r.Body).Decode(&user)
	log.Println("Deoded the request body in valid user")
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

	exists,err := services.FindUserByEmailId(&userFromDb,DB,user.Email)

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

	log.Print("Login Successful for user")

	SendJSONResponse(w,http.StatusAccepted,map[string]string{"token":jwtToken,"username":userFromDb.Username,"userId":strconv.FormatUint(uint64(userFromDb.ID), 10)})

}

func ValidateJwtToken(w http.ResponseWriter, r *http.Request) (string, error) {
    // Step 1: Retrieve the Authorization header
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        return "", errors.New("authorization header is missing")
    }

    // Step 2: Extract the JWT token from the Authorization header
    const bearerPrefix = "Bearer "
    if !strings.HasPrefix(authHeader, bearerPrefix) {
        return "", errors.New("bearer prefix not present in the Authorization Header")
    }

    jwtToken := strings.TrimPrefix(authHeader, bearerPrefix)
    if jwtToken == "" {
        return "", errors.New("jwt token is missing")
    }

    // Step 3: Extract and validate claims
    claims, err := utils.ExtractClaims(jwtToken)
    if err != nil {
        log.Printf("Error extracting claims: %v", err)
        return "", fmt.Errorf("error extracting claims from token: %w", err)
    }

    // Step 4: Check token expiration
    if expiryTime, exists := claims["exp"].(float64); exists {
        exp := time.Unix(int64(expiryTime), 0)
        if time.Now().After(exp) {
            return "", errors.New("token expired")
        }
    } else {
        return "", errors.New("expiration claim missing from token")
    }

    // Step 5: Extract and return user ID
    if userId, exists := claims["sub"]; exists {
        if userIdStr, ok := userId.(string); ok {
            log.Printf("JWT authentication successful for user: %s", userIdStr)
            return userIdStr, nil
        }
        return "", errors.New("user id claim is not a string")
    }
    
    return "", errors.New("user id claim missing from token")
}

func HandleWebSocketConnection(w http.ResponseWriter, r *http.Request, pool *config.ConnectionPool, DB *gorm.DB){

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


func StoreDocument(w http.ResponseWriter, r *http.Request, DB *gorm.DB){
	var Document models.Document
	log.Printf("request body is %v", r.Body)
	err:= json.NewDecoder(r.Body).Decode(&Document)
	if err != nil{
		SendErrorResponse(w,http.StatusBadRequest,"Wrong request body")
	}
	tx :=DB.Save(&Document)
	if(tx.Error != nil){
		SendErrorResponse(w,http.StatusInternalServerError, tx.Error.Error())
	}
	SendJSONResponse(w,http.StatusOK,"created document in DB")
}

func GetDocuments(w http.ResponseWriter, r *http.Request, DB *gorm.DB){
	var Documents []models.Document
	userId,err:= ValidateJwtToken(w,r)
	if err!= nil{
		SendErrorResponse(w,http.StatusUnauthorized,"authentication failed")
	}
	tx := DB.Where("createdBy= ? ",userId).Find(&Documents)
	if tx.Error != nil{
		SendErrorResponse(w,http.StatusInternalServerError,"Failed to retrive data from the DB")
	}
	SendJSONResponse(w,http.StatusOK,Documents)
}

func GetDocumentById(w http.ResponseWriter, r *http.Request,DB *gorm.DB, DocId string){
	if _,err:=ValidateJwtToken(w,r);err!=nil{
		SendErrorResponse(w,http.StatusUnauthorized,"authentication failed")
	}
	var Document models.Document
	DocIdUint,err := strconv.ParseUint(DocId,10,64)
	if err!= nil{
		SendErrorResponse(w,http.StatusInternalServerError,"Error Parsing the document id")
	}
	tx := DB.First(&Document, "id=?", DocIdUint)
	if tx.Error != nil{
		SendErrorResponse(w,http.StatusBadRequest, "Error fetching the data from the DB")
	}
	SendJSONResponse(w,http.StatusOK,Document)
}