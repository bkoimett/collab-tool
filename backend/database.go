package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// initDB opens the Postgres connection and creates the schema if needed.
// It reads DATABASE_URL from the environment first; falls back to hardcoded
// defaults for local development.
func initDB() *sql.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf(
			"host=localhost port=5432 user=editor_user password=password123 dbname=collab_editor sslmode=disable",
		)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("[db] open error: %v", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err = db.Ping(); err != nil {
		log.Fatalf("[db] ping error: %v", err)
	}

	migrate(db)
	log.Println("[db] connected and schema ready")
	return db
}

func migrate(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS documents (
			id          VARCHAR(100) PRIMARY KEY,
			content     TEXT         NOT NULL DEFAULT '',
			version     INTEGER      NOT NULL DEFAULT 0,
			created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("[db] migration failed: %v", err)
	}
}

// DBDoc holds a persisted document
type DBDoc struct {
	ID      string
	Content string
	Version int
}

func dbGetDocument(db *sql.DB, id string) (*DBDoc, error) {
	row := db.QueryRow(
		`SELECT id, content, version FROM documents WHERE id = $1`, id,
	)
	doc := &DBDoc{}
	err := row.Scan(&doc.ID, &doc.Content, &doc.Version)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func dbSaveDocument(db *sql.DB, id, content string, version int) error {
	_, err := db.Exec(`
		INSERT INTO documents (id, content, version, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (id)
		DO UPDATE SET content = $2, version = $3, updated_at = NOW()
	`, id, content, version)
	return err
}

func dbListDocuments(db *sql.DB) ([]DBDoc, error) {
	rows, err := db.Query(
		`SELECT id, content, version FROM documents ORDER BY updated_at DESC LIMIT 50`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []DBDoc
	for rows.Next() {
		var d DBDoc
		if err := rows.Scan(&d.ID, &d.Content, &d.Version); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func dbCreateDocument(db *sql.DB, id, content string) error {
	_, err := db.Exec(`
		INSERT INTO documents (id, content, version) VALUES ($1, $2, 0)
		ON CONFLICT DO NOTHING
	`, id, content)
	return err
}
