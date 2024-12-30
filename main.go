package main

import (
	"fmt"
	"log"
	"net/http"
	"real-time-collab/config"
	"real-time-collab/controller"
)


func main() {

	DB := config.InitDb()

	mux:= http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
        controller.RegisterUser(w, r, DB)
    })
	mux.HandleFunc("/login",func(w http.ResponseWriter, r *http.Request) {
		controller.LoginUser(w,r,DB)
	})
	mux.HandleFunc("/validate",controller.ValidateJwtToken)

	log.Println("Starting server on :8080")

	err:= http.ListenAndServe(":8080", mux)

	if(err != nil){
		log.Fatalf("Could not start server due to error : %v", err)
	}

}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World")
}