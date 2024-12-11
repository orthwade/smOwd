package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	// "github.com/joho/godotenv/autoload"

	"smOwd/pql"
	"smOwd/telegram_bot"
)

func TestPQL() *sql.DB {
	godotenv.Load()
	// Load environment variables
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD") // Password from .env
	dbName := os.Getenv("DB_NAME")

	// Connection string with password
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

	db, err := pql.ConnectToDB(connStr)
	if err != nil {
		log.Fatal(err)
	}
	// err = db.Ping()
	// if err != nil {
	// log.Fatal("Error Ping postgres: ", err)
	// }

	fmt.Printf("Successfully connected to %s db\n", dbName)

	custom_type_name := "anime_id_and_last_episode"

	if pql.IsCustomTypeCreated(db, custom_type_name) {
		fmt.Printf("Column %s is already created\n", custom_type_name)
	} else {
		fmt.Printf("Column %s is not created\n", custom_type_name)
		pql.CreateCustomTypeAnimeIdAndLastEpisode(db)
	}

	err = pql.CreateTableNamedUsers(db)
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Table users created successfully")
	}

	pql.CheckAnimeIdAndLastEpisodeColumn(db)
	pql.CheckChatIdColumn(db)

	return db
}

func TestPostgresConnection() {
	godotenv.Load()
	// Load environment variables
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD") // Password from .env
	dbName := os.Getenv("DB_NAME")

	// Connection string with password
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

	// Open a connection to the database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error opening connection: %v\n", err)
	}
	defer db.Close()

	// Ping the database to check if the connection is successful
	err = db.Ping()
	if err != nil {
		log.Fatalf("Error connecting to database: %v\n", err)
	}

	// If successful, print a success message
	fmt.Println("Successfully connected to PostgreSQL!")
}

func main() {
	// Create a new logger instance with a TextHandler
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a context with the logger stored in it
	ctx := context.WithValue(context.Background(), "logger", logger)

	// TestPostgresConnection()
	// return
	db := TestPQL()
	defer db.Close()
	// pql.DeleteColumn(db, "users", "anime_ids")
	pql.PrintTableColumnsNamesAndTypes(db, "users")
	telegram_bot.StartBotAndHandleUpdates(ctx, db)
}
