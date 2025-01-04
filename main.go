package main

import (
	"log"
	"net/http"
	"real-time-collab/config"
	"real-time-collab/middleware"
	"real-time-collab/routes"
	"real-time-collab/utils"
)


func main() {

	DB := config.InitDb()

	workers := 40

	utils.AutoMigrateModels(DB)

	pool := config.NewConnectionPool(workers,DB) 

	go pool.StartBroadcasting()

	mux:= http.NewServeMux()
	routes.SetRoutesForMux(mux,DB,pool)

	handler:= middleware.AddCORSMiddleware(mux)
	
	log.Println("Starting server on :8080")

	err:= http.ListenAndServe(":8080", handler)

	if(err != nil){
		log.Fatalf("Could not start server due to error : %v", err)
	}

}