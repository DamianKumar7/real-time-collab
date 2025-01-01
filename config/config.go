package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"real-time-collab/models"
	"sync"

	"github.com/gorilla/websocket"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DbConnection *gorm.DB
var DocumentEvent models.DocumentEvent

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
    MessageQueue []byte
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


func (pool *ConnectionPool) ReadMessage(connection *websocket.Conn, DB *gorm.DB){
    defer func(){
        pool.Mutex.Lock()
        delete(pool.Connections, connection)
        pool.Mutex.Unlock()
        connection.Close()
    }()
    for{

        log.Printf("trying to read message")
        _,message,err := connection.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("Error reading message: %v", err)
            }
            return
        }
        go func(){
            err:= PersistData(message, DB)
            if(err!= nil){
                log.Printf("error persisting data error %v",err.Error())
            }
        }()
        pool.Broadcast <- message
    }
}


func PersistData(message []byte, db *gorm.DB) error {
    var documentEvent models.DocumentEvent
    if err := json.Unmarshal(message, &documentEvent); err != nil {
        return fmt.Errorf("failed to unmarshal data: %w", err)
    }

    // Validate required fields
    if documentEvent.DocID == "" || documentEvent.Operation == "" {
        return fmt.Errorf("missing required fields")
    }

    // Use a transaction for atomic operations
    return db.Transaction(func(tx *gorm.DB) error {
        if err := tx.Create(&documentEvent).Error; err != nil {
            return fmt.Errorf("failed to save event: %w", err)
        }

        if err := PersistDocumentSnapshot(documentEvent, tx); err != nil {
            return fmt.Errorf("failed to update document: %w", err)
        }

        return nil
    })
}


func PersistDocumentSnapshot(event models.DocumentEvent, db *gorm.DB) error {
    var document models.Document
    if err := db.First(&document, "id = ?", event.DocID).Error; err != nil {
        return fmt.Errorf("failed to fetch document: %w", err)
    }

    if err := applyChangesToDocument(&document, event); err != nil {
        return fmt.Errorf("failed to apply changes: %w", err)
    }

    return db.Save(&document).Error
}

func applyChangesToDocument(doc *models.Document, event models.DocumentEvent) error {
    contentLen := len(doc.Content)
    
    if event.Position < 0 || event.Position > contentLen {
        return fmt.Errorf("invalid position: %d", event.Position)
    }

    switch event.Operation {
    case "insert":
        doc.Content = doc.Content[:event.Position] + event.Content + doc.Content[event.Position:]
    
    case "delete":
        if event.Position+event.Length > contentLen {
            return fmt.Errorf("deletion range exceeds content length")
        }
        doc.Content = doc.Content[:event.Position] + doc.Content[event.Position+event.Length:]
    
    case "replace":
        if event.Position+event.Length > contentLen {
            return fmt.Errorf("replacement range exceeds content length")
        }
        doc.Content = doc.Content[:event.Position] + event.Content + doc.Content[event.Length+event.Position:]
    
    default:
        return fmt.Errorf("invalid operation: %s", event.Operation)
    }

    return nil
}
