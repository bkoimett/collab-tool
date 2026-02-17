package main

import (
    "database/sql"
    "fmt"
    // "log"

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

    // Fixed: "postgres" not "postgress"
    db, err := sql.Open("postgres", psqlInfo)
    if err != nil {
        panic(err)
    }

    // Test the connection
    err = db.Ping()
    if err != nil {
        panic(err)
    }

    // Create table if not exists
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS documents (
            id VARCHAR(50) PRIMARY KEY,
            content TEXT
        )
    `)
    if err != nil {
        panic(err)
    }

    fmt.Println("Successfully connected to database!")
    
    // DON'T close here - return the open connection
    return db
}