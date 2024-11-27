package pql

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

// ConnectToDB opens a connection to the PostgreSQL database.
func ConnectToDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// dbExists checks if the specified database exists.
func DbExists(db *sql.DB, dbName string) bool {
	var exists bool
	query := `SELECT EXISTS (
		SELECT 1 FROM pg_database WHERE datname = $1
	);`
	err := db.QueryRow(query, dbName).Scan(&exists)
	if err != nil {
		log.Fatal("Error checking if database exists:", err)
	}
	return exists
}

// CreateDatabase creates a new PostgreSQL database.
func CreateDatabase(db *sql.DB, dbName string) error {
	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	return err
}

func CreateTable(db *sql.DB) error {
	// Create the table with the specified table name
	createTableSQL := `CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		enabled BOOL,
		anime_ids BIGINT[]
	);`

	_, err := db.Exec(createTableSQL)
	return err
}
func UserExists(db *sql.DB, userID int64) bool {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	if err != nil {
		log.Fatal(err)
	}
	return exists
}

func SetUser(db *sql.DB, userID int64, enabled bool, anime_id []int64) {

	// Insert the user, or update if a conflict on the primary key occurs
	_, err := db.Exec(`
		INSERT INTO users (id, enabled, anime_ids)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE
		SET enabled = EXCLUDED.enabled, anime_ids = EXCLUDED.anime_ids`,
		userID, enabled, pq.Array(anime_id))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("User inserted or updated successfully")
}
