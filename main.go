package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"

	"smOwd/logger"
	"smOwd/pql"
	"smOwd/telegram_bot"

	"github.com/joho/godotenv"
)

func TestPQL(ctx context.Context) *sql.DB {
	// Get the logger from the context
	logger := ctx.Value("logger").(*slog.Logger)

	// Load environment variables only once
	if err := godotenv.Load(); err != nil {
		logger.Error("Error loading .env file", "error", err)
		log.Fatal()
	}

	// Load environment variables
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD") // Password from .env
	dbName := os.Getenv("DB_NAME")

	// Connection string with password
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

	// Try to connect to the database
	db, err := pql.ConnectToDB(connStr)
	if err != nil {
		logger.Error("Error connecting to database", "error", err)
		log.Fatal(err)
	}
	logger.Info("Successfully connected to database", "db", dbName)

	// Check for the custom type "anime_id_and_last_episode"
	customTypeName := "anime_id_and_last_episode"
	if pql.IsCustomTypeCreated(ctx, db, customTypeName) {
		logger.Info("Custom type already created", "type", customTypeName)
	} else {
		logger.Warn("Custom type not found, creating...", "type", customTypeName)
		pql.CreateCustomTypeAnimeIdAndLastEpisode(ctx, db)
	}

	// Create the users table if it doesn't exist
	if err := pql.CreateTableNamedUsers(ctx, db); err != nil {
		logger.Error("Error creating users table", "error", err)
		log.Fatal(err)
	} else {
		logger.Info("Table users created successfully")
	}

	// Check and add necessary columns
	pql.CheckAnimeIdAndLastEpisodeColumn(ctx, db)
	pql.CheckChatIdColumn(ctx, db)

	return db
}

func main() {
	// Create a new logger instance with a TextHandler
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a context with the logger stored in it
	ctx := context.WithValue(context.Background(), "logger", logger)

	// Test database connection and operations
	db := TestPQL(ctx)
	defer db.Close()

	pql.PrintTableColumnsNamesAndTypes(db, "users")
	telegram_bot.StartBotAndHandleUpdates(ctx, db)
}
