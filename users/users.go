package users

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"smOwd/logs"
	"smOwd/pql"
)

type User struct {
	ID           int
	TelegramID   int
	FirstName    string
	LastName     string
	UserName     string
	LanguageCode string
	IsBot        bool
	Enabled      bool
}

func CheckTable(ctx context.Context, db *sql.DB) (bool, error) {
	return pql.CheckTable(ctx, db, "users")
}

func CreateTable(ctx context.Context, db *sql.DB) error {
	tableName := "users"

	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	createTableQuery := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SERIAL PRIMARY KEY,             -- Primary key with an index
			telegram_id BIGINT UNIQUE NOT NULL, -- Unique index for telegram_id
			first_name TEXT NOT NULL,
			last_name TEXT,
			user_name TEXT,
			language_code TEXT,
			is_bot BOOLEAN NOT NULL DEFAULT FALSE,
			enabled BOOLEAN NOT NULL DEFAULT TRUE
		);
	`, tableName)

	_, err := db.ExecContext(ctx, createTableQuery)
	if err != nil {
		logger.Error("Failed to create table", "error", err)
		return err
	}

	// Add index for telegram_id if needed
	createIndexQuery := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_telegram_id ON %s (telegram_id);`, tableName)
	_, err = db.ExecContext(ctx, createIndexQuery)
	if err != nil {
		logger.Error("Failed to create index on telegram_id", "error", err)
		return err
	}

	logger.Info("Table and index created successfully", "table", tableName)
	return nil
}

func Add(ctx context.Context, db *sql.DB, u User) error {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}
	query := `
		INSERT INTO users (telegram_id, first_name, last_name, user_name, language_code, is_bot, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (telegram_id) DO NOTHING;
	`

	_, err := db.Exec(query, u.TelegramID, u.FirstName, u.LastName, u.UserName,
		u.LanguageCode, u.IsBot, u.Enabled)

	if err != nil {
		logger.Error("Failed to store user", "error", err)
		return err
	}

	logger.Info(fmt.Sprintf("User with TelegramID %d stored successfully", u.TelegramID))
	return nil
}

func Get(ctx context.Context, db *sql.DB, telegramID int) (*User, error) {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	query := `
		SELECT id, telegram_id, first_name, last_name, user_name, language_code, is_bot, enabled
		FROM users
		WHERE telegram_id = $1;
	`

	var u User
	err := db.QueryRowContext(ctx, query, telegramID).Scan(
		&u.ID, // Fetch the primary key
		&u.TelegramID,
		&u.FirstName,
		&u.LastName,
		&u.UserName,
		&u.LanguageCode,
		&u.IsBot,
		&u.Enabled,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Warn("No user found", "Telegram ID", telegramID)
			return nil, nil // No rows found, return nil user
		}
		logger.Error("Failed to retrieve user", "error", err, "Telegram ID", telegramID)
		return nil, fmt.Errorf("failed to retrieve user with TelegramID %d: %w", telegramID, err)
	}

	logger.Info(fmt.Sprintf("User with TelegramID %d retrieved successfully", telegramID))
	return &u, nil
}

func Remove(ctx context.Context, db *sql.DB, id int) error {
	return pql.RemoveRecord(ctx, db, "users", id)
}
