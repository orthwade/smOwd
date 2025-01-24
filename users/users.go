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

func CreateTable(ctx context.Context, db *sql.DB, tableName string) error {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	createTableQuery := fmt.Sprintf(`
		CREATE TABLE %s (
			telegram_id SERIAL PRIMARY KEY,
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
		logger.Error("Failed to create table", "table", tableName, "error", err)
		return err
	}

	logger.Info("Table created successfully", "table", tableName)
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
		SELECT telegram_id, first_name, last_name, user_name, language_code, is_bot
		FROM users
		WHERE telegram_id = $1;
	`

	var u User
	err := db.QueryRowContext(ctx, query, telegramID).Scan(
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
			logger.Warn(fmt.Sprintf("No user found with TelegramID %d", telegramID))
			return nil, nil
		}
		logger.Error("Failed to retrieve user", "error", err)
		return nil, err
	}

	logger.Info(fmt.Sprintf("User with TelegramID %d retrieved successfully", telegramID))
	return &u, nil
}

func Remove(ctx context.Context, db *sql.DB, telegramID int) error {
	logger, ok := ctx.Value("logger").(*logs.Logger)

	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	query := `
		DELETE FROM users
		WHERE telegram_id = $1;
	`

	_, err := db.ExecContext(ctx, query, telegramID)

	if err != nil {
		logger.Error("Failed to delete user", "error", err, "Telegram ID", telegramID)
	} else {
		logger.Info("Deleted user", "Telegram ID", telegramID)
	}

	return err
}
