package animes

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"smOwd/logs"
	"smOwd/pql"
)

type Anime struct {
	ShikiID       int
	MalId         int
	English       string
	Japanese      string
	Episodes      int
	EpisodesAired int
}

func CheckTable(ctx context.Context, db *sql.DB) (bool, error) {
	return pql.CheckTable(ctx, db, "animes")
}

func CreateTable(ctx context.Context, db *sql.DB) error {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	createTableQuery := `
		CREATE TABLE animes (
			shiki_id SERIAL PRIMARY KEY,
			mal_id INT UNIQUE,
			english TEXT,
			japanese TEXT,
			episodes INT NOT NULL,
			episodes_aired INT NOT NULL
		);
	`

	_, err := db.ExecContext(ctx, createTableQuery)
	if err != nil {
		logger.Error("Failed to create 'animes' table", "error", err)
		return err
	}

	logger.Info("'animes' table created successfully")
	return nil
}

func Add(ctx context.Context, db *sql.DB, a Anime) error {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}
	query := `
		INSERT INTO users (shiki_id, mal_id, english, japanese, episodes, episodes_aired)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (shiki_id) DO NOTHING;
	`

	_, err := db.Exec(query, a.ShikiID, a.MalId, a.English, a.Japanese, a.Episodes,
		a.EpisodesAired)

	if err != nil {
		logger.Error("Failed to add anime", "error", err)
		return err
	}

	logger.Info(fmt.Sprintf("Anime with ShikiID %d stored successfully", a.ShikiID))
	return nil
}

func Get(ctx context.Context, db *sql.DB, shikiID int) (*Anime, error) {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	query := `
		SELECT shiki_id, mal_id, english, japanese, episodes, episodes_aired
		FROM animes
		WHERE shiki_id = $1;
	`

	var a Anime
	err := db.QueryRowContext(ctx, query, shikiID).Scan(
		&a.ShikiID,
		&a.MalId,
		&a.English,
		&a.Japanese,
		&a.Episodes,
		&a.EpisodesAired,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Warn(fmt.Sprintf("No anime found with ShikiID %d", shikiID))
			return nil, nil
		}
		logger.Error("Failed to retrieve anime", "error", err)
		return nil, err
	}

	logger.Info(fmt.Sprintf("Anime with ShikiID %d retrieved successfully", shikiID))
	return &a, nil
}

func Remove(ctx context.Context, db *sql.DB, shikiID int) error {
	logger, ok := ctx.Value("logger").(*logs.Logger)

	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	query := `
		DELETE FROM animes
		WHERE shiki_id = $1;
	`

	_, err := db.ExecContext(ctx, query, shikiID)

	if err != nil {
		logger.Error("Failed to delete anime", "error", err, "Shiki ID", shikiID)
	} else {
		logger.Info("Deleted anime", "Shiki ID", shikiID)
	}

	return err
}
