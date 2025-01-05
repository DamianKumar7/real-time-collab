package routes

import (
	"net/http"
	"real-time-collab/config"
	"real-time-collab/controller"
	"gorm.io/gorm"
)

func SetRoutesForMux(mux *http.ServeMux, DB *gorm.DB,pool *config.ConnectionPool){

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
        controller.RegisterUser(w, r, DB)
    })
	mux.HandleFunc("/login",func(w http.ResponseWriter, r *http.Request) {
		controller.LoginUser(w,r,DB)
	})
	mux.HandleFunc("/ws",func(w http.ResponseWriter, r *http.Request) {
		controller.HandleWebSocketConnection(w,r,pool,DB)
	})

	mux.HandleFunc("/save-document",func(w http.ResponseWriter, r *http.Request) {
		controller.StoreDocument(w,r,DB)
	})

	mux.HandleFunc("/documents/get/{id}",func(w http.ResponseWriter, r *http.Request) {
		DocId := r.PathValue("id")
		controller.GetDocumentById(w,r,DB,DocId)
	})

}