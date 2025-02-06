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
	ID                  int
	UserID              int
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
			user_id INT NOT NULL,
			shiki_id INT NOT NULL,
			last_episode_notified INT DEFAULT 0,

			CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
			UNIQUE (user_id, shiki_id)
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
		"CREATE INDEX IF NOT EXISTS idx_shiki_id ON subscriptions (shiki_id);")
	if err != nil {
		logger.Fatal("Failed to index anime id column", "error", err)
		return fmt.Errorf("Failed to index anime id column: %w", err)
	}

	logger.Info("Subscriptions table created successfully", "table", tableName)
	return nil
}

func Add(ctx context.Context, db *sql.DB, s Subscription) error {
	logger := logs.DefaultFromCtx(ctx)

	// Define the SQL query to insert a new subscription record
	query := `
        INSERT INTO subscriptions (user_id, shiki_id, last_episode_notified)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id, shiki_id) DO NOTHING;  -- Avoid duplicate subscriptions for the same user and anime
    `

	// Execute the query with the provided Subscription data
	_, err := db.ExecContext(ctx, query, s.UserID, s.ShikiID, s.LastEpisodeNotified)
	if err != nil {
		logger.Error("Failed to add subscription", "error", err)
		return err
	}

	// Log successful insertion
	logger.Info(fmt.Sprintf("Subscription added for user %d and anime %d", s.UserID, s.ShikiID))
	return nil
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
	db *sql.DB, user_id int) (*Subscription, error) {
	return get(ctx, db, "user_id", user_id)
}

func GetByAnimeID(ctx context.Context,
	db *sql.DB, shiki_id int) (*Subscription, error) {
	return get(ctx, db, "shiki_id", shiki_id)
}

func Remove(ctx context.Context, db *sql.DB, id int) error {
	return pql.RemoveRecord(ctx, db, tableName, id)
}
