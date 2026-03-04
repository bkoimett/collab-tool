package main

import (
    "database/sql"
    "fmt"
    "log"  // Uncomment this
    
    _ "github.com/lib/pq"
)

type Document struct {
    ID      string `json:"id"`
    Content string `json:"content"`
}

const (
    host     = "localhost"
    port     = 5432
    user     = "editor_user"
    password = "password123"
    dbname   = "collab_editor"
)

func initDB() *sql.DB {
    psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
        "password=%s dbname=%s sslmode=disable",
        host, port, user, password, dbname)

    db, err := sql.Open("postgres", psqlInfo)
    if err != nil {
        log.Fatal(err)  // Use log.Fatal instead of panic
    }

    err = db.Ping()
    if err != nil {
        log.Fatal(err)
    }

    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS documents (
            id VARCHAR(50) PRIMARY KEY,
            content TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Successfully connected to database!")
    return db
}

// Add these helper functions
func getDocument(db *sql.DB, id string) (string, error) {
    var content string
    err := db.QueryRow("SELECT content FROM documents WHERE id = $1", id).Scan(&content)
    if err == sql.ErrNoRows {
        return "", nil
    }
    return content, err
}

func saveDocument(db *sql.DB, id string, content string) error {
    _, err := db.Exec(`
        INSERT INTO documents (id, content, updated_at) 
        VALUES ($1, $2, CURRENT_TIMESTAMP)
        ON CONFLICT (id) 
        DO UPDATE SET content = $2, updated_at = CURRENT_TIMESTAMP`,
        id, content)
    return err
}