package routes

import (
	"log"
	"net/http"
	"real-time-collab/config"
	"real-time-collab/controller"
	"gorm.io/gorm"
)

func SetRoutesForMux(mux *http.ServeMux, DB *gorm.DB,pool *config.ConnectionPool){

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Explicitly setting CORS to allow all origins")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if(r.Method == http.MethodOptions){
			w.WriteHeader(http.StatusOK)
			return
		}
        controller.RegisterUser(w, r, DB)
    })
	mux.HandleFunc("/login",func(w http.ResponseWriter, r *http.Request) {
		if(r.Method == http.MethodOptions){
			w.WriteHeader(http.StatusOK)
			return
		}
		controller.LoginUser(w,r,DB)
	})
	mux.HandleFunc("/ws",func(w http.ResponseWriter, r *http.Request) {
		controller.HandleWebSocketConnection(w,r,pool,DB)
	})

	mux.HandleFunc("/save-document",func(w http.ResponseWriter, r *http.Request) {
		if(r.Method == http.MethodOptions){
			w.WriteHeader(http.StatusOK)
			return
		}
		controller.StoreDocument(w,r,DB)
	})

}