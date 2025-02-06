package subscriptions

import (
	"context"
	"database/sql"
	"fmt"

	// "smOwd/animes"
	"smOwd/logs"
	"smOwd/pql"
	// "smOwd/users"
)

const tableName = "subscriptions"

type Subscription struct {
	ID                  int //PRIMARY KEY
	UserID              int
	TelegramID          int
	ShikiID             int
	LastEpisodeNotified int
}

func CheckTable(ctx context.Context, db *sql.DB) (bool, error) {
	return pql.CheckTable(ctx, db, tableName)
}

func CreateTable(ctx context.Context, db *sql.DB) error {
	logger := logs.DefaultFromCtx(ctx)

	// SQL query to create the subscriptions table
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SERIAL PRIMARY KEY,
			user_id SERIAL NOT NULL,
			telegram_id INT NOT NULL,
			shiki_id TEXT NOT NULL,
			last_episode_notified INT DEFAULT 0,

			CONSTRAINT fk_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
			CONSTRAINT fk_telegram_id FOREIGN KEY (telegram_id) REFERENCES users (telegram_id) ON DELETE CASCADE,
			UNIQUE (user_id, shiki_id),
			UNIQUE (telegram_id, shiki_id)
		);
	`, tableName)

	// Execute the query
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		logger.Fatal("Failed to create subscriptions table", "error", err)
		return fmt.Errorf("failed to create subscriptions table: %w", err)
	}

	_, err = db.ExecContext(ctx,
		"CREATE INDEX IF NOT EXISTS idx_user_id ON subscriptions (user_id);")
	if err != nil {
		logger.Fatal("Failed to index user id column", "error", err)
		return fmt.Errorf("Failed to index user id column: %w", err)
	}

	_, err = db.ExecContext(ctx,
		"CREATE INDEX IF NOT EXISTS idx_telegram_id ON subscriptions (telegram_id);")
	if err != nil {
		logger.Fatal("Failed to index telegram id column", "error", err)
		return fmt.Errorf("Failed to index telegram id column: %w", err)
	}

	_, err = db.ExecContext(ctx,
		"CREATE INDEX IF NOT EXISTS idx_shiki_id ON subscriptions USING HASH (shiki_id);")

	if err != nil {
		logger.Fatal("Failed to index anime id column", "error", err)
		return fmt.Errorf("Failed to index anime id column: %w", err)
	}

	logger.Info("Subscriptions table created successfully", "table", tableName)
	return nil
}

func Add(ctx context.Context, db *sql.DB, s Subscription) (int, error) {
	logger := logs.DefaultFromCtx(ctx)

	// Define the SQL query to insert a new subscription record
	query := `
        INSERT INTO subscriptions (user_id, telegram_id, shiki_id, last_episode_notified)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (user_id, telegram_id, shiki_id) DO NOTHING
    `

	// Execute the query with the provided Subscription data
	result, err := db.ExecContext(ctx, query, s.TelegramID, s.ShikiID, s.LastEpisodeNotified)
	if err != nil {
		logger.Error("Failed to add subscription", "error", err)
		return -1, err
	}

	rowsAffected, _ := result.RowsAffected()

	var id int

	if rowsAffected == 0 {
		logger.Warn("No new row inserted due to conflict")
		return -1, nil
	} else {
		id64, err := result.LastInsertId()

		if err != nil {
			logger.Error("Error getting last insert ID", "error", err)
			return -1, err
		}

		id = int(id64)
	}

	// Log successful insertion
	logger.Info(fmt.Sprintf("Subscription added for user %d and anime %d", s.TelegramID, s.ShikiID))
	return id, nil
}

func get(ctx context.Context,
	db *sql.DB, idFieldName string, idValue int) (*Subscription, error) {
	var s Subscription
	err := pql.GetRecord(ctx, db, tableName, idFieldName, idValue, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func GetByID(ctx context.Context, db *sql.DB, id int) (*Subscription, error) {
	return get(ctx, db, "id", id)
}

func GetByUserID(ctx context.Context,
	db *sql.DB, telegram_id int) (*Subscription, error) {
	return get(ctx, db, "telegram_id", telegram_id)
}

func GetByAnimeID(ctx context.Context,
	db *sql.DB, shiki_id int) (*Subscription, error) {
	return get(ctx, db, "shiki_id", shiki_id)
}

func Remove(ctx context.Context, db *sql.DB, id int) error {
	return pql.RemoveRecord(ctx, db, tableName, id)
}
