package users

import (
	"context"
	"database/sql"
	"fmt"
	"smOwd/logs"
	"smOwd/pql"
)

const tableName = "users"

type User struct {
	ID           int //PRIMARY KEY
	TelegramID   int
	FirstName    string
	LastName     string
	UserName     string
	LanguageCode string
	IsBot        bool
	Enabled      bool
}

func CheckTable(ctx context.Context, db *sql.DB) (bool, error) {
	return pql.CheckTable(ctx, db, tableName)
}

func CreateTable(ctx context.Context, db *sql.DB) error {
	tableName := tableName
	columns := `
		id SERIAL PRIMARY KEY,
		telegram_id BIGINT UNIQUE NOT NULL,
		first_name TEXT NOT NULL,
		last_name TEXT,
		user_name TEXT,
		language_code TEXT,
		is_bot BOOLEAN NOT NULL DEFAULT FALSE,
		enabled BOOLEAN NOT NULL DEFAULT TRUE
	`
	indexName := "idx_telegram_id"
	indexColumn := "telegram_id"
	return pql.CreateTable(ctx, db, tableName, columns, indexName, indexColumn)
}

func Add(ctx context.Context, db *sql.DB, u *User) error {
	logger := logs.DefaultFromCtx(ctx)

	query := `
		INSERT INTO users (telegram_id, first_name, last_name, user_name, language_code, is_bot, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (telegram_id) DO NOTHING;
	`

	_, err := db.ExecContext(ctx, query, u.TelegramID, u.FirstName, u.LastName, u.UserName, u.LanguageCode, u.IsBot, u.Enabled)
	if err != nil {
		logger.Error("Failed to add record to users", "error", err)
		return err
	}

	logger.Info(fmt.Sprintf("User with TelegramID %d added successfully", u.TelegramID))
	return nil
}

func Find(ctx context.Context, db *sql.DB, fieldName string, fieldValue int) *User {
	logger := logs.DefaultFromCtx(ctx)

	query := fmt.Sprintf(`
		SELECT id, telegram_id, first_name, last_name, user_name, language_code, is_bot, enabled
		FROM %s
		WHERE %s = $1;
	`, tableName, fieldName)

	var user User
	err := db.QueryRowContext(ctx, query, fieldValue).Scan(
		&user.ID,
		&user.TelegramID,
		&user.FirstName,
		&user.LastName,
		&user.UserName,
		&user.LanguageCode,
		&user.IsBot,
		&user.Enabled,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Warn("No user found with", fieldName, fieldValue)
			return nil // Return nil if the user is not found
		}
		logger.Error("Failed to retrieve user", fieldName, fieldValue, "error", err)
		return nil // Return nil if there's any other error
	}

	logger.Info("User retrieved successfully", fieldName, fieldValue)
	return &user
}

func FindById(ctx context.Context, db *sql.DB, id int) *User {
	return Find(ctx, db, "id", id)
}

func FindByTelegramID(ctx context.Context, db *sql.DB, telegramID int) *User {
	return Find(ctx, db, "telegram_id", telegramID)
}

func Remove(ctx context.Context, db *sql.DB, id int) error {
	return pql.RemoveRecord(ctx, db, tableName, id)
}
