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
	ID            int
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
			id SERIAL PRIMARY KEY,
			shiki_id INT UNIQUE,
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

func Get(ctx context.Context, db *sql.DB, id int) (*Anime, error) {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	query := `
		SELECT id, shiki_id, mal_id, english, japanese, episodes, episodes_aired
		FROM animes
		WHERE id = $1;
	`

	var a Anime
	err := db.QueryRowContext(ctx, query, id).Scan(
		&a.ID,
		&a.ShikiID,
		&a.MalId,
		&a.English,
		&a.Japanese,
		&a.Episodes,
		&a.EpisodesAired,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Warn(fmt.Sprintf("No anime found with ID %d", id))
			return nil, nil
		}
		logger.Error("Failed to retrieve anime", "error", err)
		return nil, err
	}

	logger.Info(fmt.Sprintf("Anime with ID %d retrieved successfully", id))
	return &a, nil
}

func Remove(ctx context.Context, db *sql.DB, id int) error {
	return pql.RemoveRecord(ctx, db, "animes", id)
}
