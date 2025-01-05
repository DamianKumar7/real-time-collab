package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"real-time-collab/models"
	"strconv"
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
    MessageQueue chan []byte
}

func NewConnectionPool(workers int, DB *gorm.DB) *ConnectionPool{
    pool :=  &ConnectionPool{
        Connections: make(map[*websocket.Conn]bool),
        Broadcast: make(chan []byte),
        MessageQueue: make(chan []byte),
    }
    for i:= 0;i <workers;i++{
        go pool.worker(i,DB)
    }
    return pool
}

func (pool *ConnectionPool) worker(worker int, DB *gorm.DB) {
    for message := range pool.MessageQueue {
        var documentEvent models.DocumentEvent
        var document models.Document
        
        if err := json.Unmarshal(message, &documentEvent); err != nil {
            log.Printf("worker %d: failed to unmarshal data: %v", worker, err)
            continue
        }
        
        if err := validateDocumentEvent(&documentEvent); err != nil {
            log.Printf("worker %d: invalid document event: %v", worker, err)
            continue
        }
        
        if err := transformDocumentEvent(&documentEvent, DB, &document); err != nil {
            log.Printf("worker %d: transformation failed: %v", worker, err)
            continue
        }
        
        if err := PersistData(&documentEvent, DB, &document); err != nil {
            log.Printf("worker %d: failed to persist data: %v", worker, err)
            continue
        }
    
        transformedMsg, err := json.Marshal(documentEvent)
        if err != nil {
            log.Printf("worker %d: failed to marshal transformed event: %v", worker, err)
            continue
        }
        
        pool.Broadcast <- transformedMsg
    }
}

func validateDocumentEvent(event *models.DocumentEvent) error {
    if event.DocID == "" {
        return fmt.Errorf("document ID is required")
    }
    if event.Operation == "" {
        return fmt.Errorf("operation is required")
    }
    return nil
}

func transformDocumentEvent(CurrentDocumentEvent *models.DocumentEvent, DB *gorm.DB, Document *models.Document) error  {
    DB.Transaction( func(tx *gorm.DB) error {
        docID, err := strconv.ParseUint(CurrentDocumentEvent.DocID, 10, 64)
        if err!= nil{
            return fmt.Errorf("failed to parse document id")
        }
        err= tx.First(Document,"id =?",docID).Error
        if err!= nil{
            return fmt.Errorf("failed to fetch document: %w", err)
        }
        if CurrentDocumentEvent.Version < Document.Version{
            var prevDocumentChanges []models.DocumentEvent
            err = tx.Where("doc_id = ? and version > ?", CurrentDocumentEvent.DocID, CurrentDocumentEvent.Version).
            Order("version ASC").
            Find(&prevDocumentChanges).Error
            if err!= nil{
                return fmt.Errorf("failed to fetch previous document changes: %w", err)
            }
            for _,DocumentChange:= range prevDocumentChanges{
                ProcessTransformation(CurrentDocumentEvent,DocumentChange)
            }            
        }
        return nil
    })
    return nil
}

func ProcessTransformation(current *models.DocumentEvent, previous models.DocumentEvent) {
    switch {
    case current.Operation == "insert" && previous.Operation == "insert":
        if current.Position > previous.Position {
            current.Position += previous.Length
        }
    case current.Operation == "insert" && previous.Operation == "delete":
        if current.Position > previous.Position {
            current.Position -= previous.Length
        }
    case current.Operation == "delete" && previous.Operation == "insert":
        if current.Position > previous.Position {
            current.Position += previous.Length
        }
    case current.Operation == "delete" && previous.Operation == "delete":
        if current.Position > previous.Position {
            current.Position -= previous.Length
        }
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
        pool.MessageQueue <- message
    }
}


func PersistData(DocumentEvent *models.DocumentEvent, db *gorm.DB, Document *models.Document) error {
    return db.Transaction(func(tx *gorm.DB) error {
        if err := PersistDocumentSnapshot(DocumentEvent, tx, Document); err != nil {
            return fmt.Errorf("failed to update document: %w", err)
        }
        if err := tx.Create(&DocumentEvent).Error; err != nil {
            return fmt.Errorf("failed to save event: %w", err)
        }
        return nil
    })
}


func PersistDocumentSnapshot(event *models.DocumentEvent, db *gorm.DB, Doc *models.Document) error {
    if err := applyChangesToDocument(Doc, event); err != nil {
        return fmt.Errorf("failed to apply changes: %w", err)
    }
    return db.Save(Doc).Error
}

func applyChangesToDocument(doc *models.Document, event *models.DocumentEvent) error {
    if event.Position < 0 || event.Position > len(doc.Content) {
        return fmt.Errorf("invalid position: %d (content length: %d)", event.Position, len(doc.Content))
    }

    doc.Version++
    event.Version = doc.Version

    switch event.Operation {
    case "insert":
        return applyInsert(doc, event)
    case "delete":
        return applyDelete(doc, event)
    case "replace":
        return applyReplace(doc, event)
    default:
        return fmt.Errorf("unsupported operation: %s", event.Operation)
    }
}

func applyInsert(doc *models.Document, event *models.DocumentEvent) error {
    doc.Content = doc.Content[:event.Position] + event.Content + doc.Content[event.Position:]
    return nil
}

func applyDelete(doc *models.Document, event *models.DocumentEvent) error {
    if event.Position+event.Length > len(doc.Content) {
        return fmt.Errorf("deletion range exceeds content length")
    }
    doc.Content = doc.Content[:event.Position] + doc.Content[event.Position+event.Length:]
    return nil
}

func applyReplace(doc *models.Document, event *models.DocumentEvent) error {
    if event.Position+event.Length > len(doc.Content) {
        return fmt.Errorf("replacement range exceeds content length")
    }
    doc.Content = doc.Content[:event.Position] + event.Content + doc.Content[event.Position+event.Length:]
    return nil
}
