package users

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"smOwd/logs"
)

type User struct {
	TelegramID   int
	FirstName    string
	LastName     string
	UserName     string
	LanguageCode string
	IsBot        bool
}

func StoreUser(ctx context.Context, db *sql.DB, u User) error {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}
	query := `
		INSERT INTO users (telegram_id, first_name, last_name, user_name, language_code, is_bot)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (telegram_id) DO NOTHING;
	`

	_, err := db.Exec(query, u.TelegramID, u.FirstName, u.LastName, u.UserName, u.LanguageCode, u.IsBot)
	if err != nil {
		logger.Error("Failed to store user", "error", err)
		return err
	}

	logger.Info(fmt.Sprintf("User with TelegramID %d stored successfully", u.TelegramID))
	return nil
}

func GetUserByTelegramID(ctx context.Context, db *sql.DB, telegramID int) (*User, error) {
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
