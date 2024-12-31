package config

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DbConnection *gorm.DB

// Initialize the database connection
func InitDb() *gorm.DB {
    // Database connection details
    connectionDetails := fmt.Sprintf(
        "host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
        os.Getenv("DB_HOST"),    // Read from environment variable
        os.Getenv("DB_USER"),    // Read from environment variable
        os.Getenv("DB_PASSWORD"),// Read from environment variable
        os.Getenv("DB_NAME"),    // Read from environment variable
        os.Getenv("DB_PORT"),    // Read from environment variable
    )

    // Open the database connection using GORM
    db, err := gorm.Open(postgres.Open(connectionDetails), &gorm.Config{})
    if err != nil {
        log.Fatalf("Error connecting to the database: %v", err)
    }

    log.Println("Successfully connected to the database")
    return db
}

type ConnectionPool struct{
    Connections map[*websocket.Conn]bool
    sync.Mutex
    Broadcast chan []byte
}

func NewConnectionPool() *ConnectionPool{
    return &ConnectionPool{
        Connections: make(map[*websocket.Conn]bool),
        Broadcast: make(chan []byte),
    }
}

func (pool *ConnectionPool) StartBroadcasting(){
    for{
        log.Printf("started broadcasting messages")
        message := <-pool.Broadcast
        log.Printf("message: %v",message)
        pool.Mutex.Lock()
        for connection:= range pool.Connections{
            err:= connection.WriteMessage(websocket.TextMessage,message)
            if(err != nil){
                log.Printf("Error writing message: %v", err)
                connection.Close()
                delete(pool.Connections, connection)
            }
        }
        pool.Mutex.Unlock()
    }
}


func (pool *ConnectionPool) ReadMessage(connection *websocket.Conn){
    defer func(){
        pool.Mutex.Lock()
        delete(pool.Connections, connection)
        pool.Mutex.Unlock()
        connection.Close()
    }()
    for{

        log.Printf("trying to read message")
        _,message,err := connection.ReadMessage()
        if(err != nil){
            log.Printf("error reading message  %v",err.Error())
            connection.Close()
            return
        }
        pool.Broadcast <- message
    }
}