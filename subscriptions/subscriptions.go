package subscriptions

import (
	"context"
	"database/sql"
	"fmt"

	"smOwd/animes"
	"smOwd/logs"
	"smOwd/pql"
	// "smOwd/users"
)

const tableName = "subscriptions"

type Subscription struct {
	ID                  int //PRIMARY KEY
	TelegramID          int
	ShikiID             string
	LastEpisodeNotified int
	Anime               *animes.Anime
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
			telegram_id BIGINT NOT NULL,
			shiki_id TEXT NOT NULL,
			last_episode_notified INT DEFAULT 0,

			CONSTRAINT fk_telegram_id FOREIGN KEY (telegram_id) REFERENCES users (telegram_id) ON DELETE CASCADE,
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
		"CREATE INDEX IF NOT EXISTS idx_telegram_id ON subscriptions (telegram_id);")
	if err != nil {
		logger.Fatal("Failed to index user id column", "error", err)
		return fmt.Errorf("Failed to index user id column: %w", err)
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
        INSERT INTO subscriptions (telegram_id, shiki_id, last_episode_notified)
        VALUES ($1, $2, $3)
        ON CONFLICT (telegram_id, shiki_id) DO NOTHING
    `

	var id int
	// Execute the query with the provided Subscription data
	row := db.QueryRowContext(ctx, query,
		s.TelegramID, s.ShikiID, s.LastEpisodeNotified)

	err := row.Scan(&id)

	if err != nil {
		if err == sql.ErrNoRows {
			logger.Warn("No new row inserted due to conflict")
			return -1, nil
		} else {
			logger.Error("Error getting last insert ID", "error", err)
			return -1, err
		}
	}

	// Log successful insertion
	logger.Info(fmt.Sprintf("Subscription added for user %d and anime %d", s.TelegramID, s.ShikiID))
	return id, nil
}

func Find(ctx context.Context, db *sql.DB, telegramID int, shikiID string) *Subscription {
	logger := logs.DefaultFromCtx(ctx)

	query := fmt.Sprintf(`
		SELECT id, telegram_id, shiki_id, last_episode_notified
		FROM %s
		WHERE telegram_id = $1
		AND shiki_id = $2;
	`, tableName)

	var s Subscription
	err := db.QueryRowContext(ctx, query, telegramID, shikiID).Scan(
		&s.ID,
		&s.TelegramID,
		&s.ShikiID,
		&s.LastEpisodeNotified,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Warn("No subscrition found",
				"Telegram ID", telegramID,
				"Shiki ID", shikiID)
			return nil
		}
		logger.Error("Failed to retrieve subscription",
			"Telegram ID", telegramID,
			"Shiki ID", shikiID,
			"error", err)
		return nil
	}

	logger.Info("Subscription retrieved successfully",
		"Telegram ID", telegramID,
		"Shiki ID", shikiID)

	return &s
}

func FindAll(ctx context.Context, db *sql.DB, telegramID int) []Subscription {
	logger := logs.DefaultFromCtx(ctx)

	query := fmt.Sprintf(`
		SELECT id, telegram_id, shiki_id, last_episode_notified
		FROM %s
		WHERE telegram_id = $1;
	`, tableName)

	var subscriptions []Subscription

	rows, err := db.QueryContext(ctx, query, telegramID)

	defer rows.Close()

	if err != nil {
		logger.Error("Error searching subscriptions",
			"Telegram ID", telegramID,
			"error", err)

		return nil
	}

	for rows.Next() {
		var s Subscription

		err := rows.Scan(
			&s.ID,
			&s.TelegramID,
			&s.ShikiID,
			&s.LastEpisodeNotified,
		)

		if err != nil {
			logger.Error("Error processing row", "error", err)
			return nil
		}

		subscriptions = append(subscriptions, s)
	}

	logger.Info("Subscriptions retrieved successfully",
		"Telegram ID", telegramID)

	return subscriptions
}

func SelectAll(ctx context.Context, db *sql.DB) []Subscription {
	logger := logs.DefaultFromCtx(ctx)

	query := fmt.Sprintf(`
		SELECT id, telegram_id, shiki_id, last_episode_notified
		FROM %s;
	`, tableName)

	var subscriptions []Subscription

	rows, err := db.QueryContext(ctx, query)

	defer rows.Close()

	if err != nil {
		logger.Error("Error searching subscriptions",
			"error", err)

		return nil
	}

	for rows.Next() {
		var s Subscription

		err := rows.Scan(
			&s.ID,
			&s.TelegramID,
			&s.ShikiID,
			&s.LastEpisodeNotified,
		)

		if err != nil {
			logger.Error("Error processing row", "error", err)
			return nil
		}

		subscriptions = append(subscriptions, s)
	}

	logger.Info("Subscriptions retrieved successfully")

	return subscriptions
}

func SetLastEpisode(ctx context.Context, db *sql.DB, id int, n int) error {
	return pql.SetField(ctx, db, tableName, "id", id, "last_episode_notified", val)
}

func Remove(ctx context.Context, db *sql.DB, id int) error {
	return pql.RemoveRecord(ctx, db, tableName, id)
}
