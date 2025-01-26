package subscriptions

import (
	"smOwd/animes"
	"smOwd/pql"
	"smOwd/users"
)

const tableName = "subscriptions"

type Subscription struct {
	ID                  int
	UserID              int
	AnimeID             int
	LastEpisodeNotified int
}

func CheckTable(ctx context.Context, db *sql.DB) (bool, error) {
	return pql.CheckTable(ctx, db, tableName)
}

func CreateTable(ctx context.Context, db *sql.DB) error {
	// Initialize logger
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// SQL query to create the subscriptions table
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SERIAL PRIMARY KEY,
			user_id INT NOT NULL,
			anime_id INT NOT NULL,
			last_episode_notified INT DEFAULT 0,

			CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
			CONSTRAINT fk_anime FOREIGN KEY (anime_id) REFERENCES animes (id) ON DELETE CASCADE,
			UNIQUE (user_id, anime_id)
		);
	`, tableName)

	// Execute the query
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		logger.Error("Failed to create subscriptions table", "error", err)
		return fmt.Errorf("failed to create subscriptions table: %w", err)
	}

	logger.Info("Subscriptions table created successfully", "table", tableName)
	return nil
}
