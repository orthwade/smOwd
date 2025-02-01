package animes

import (
	"context"
	"database/sql"
	"fmt"
	"smOwd/logs"
	"smOwd/pql"
)

const tableName = "animes"

type Anime struct {
	ID            int
	ShikiID       int
	MalID         int
	English       string
	Japanese      string
	Status        string
	Episodes      int
	EpisodesAired int
}

func CheckTable(ctx context.Context, db *sql.DB) (bool, error) {
	return pql.CheckTable(ctx, db, tableName)
}

func CreateTable(ctx context.Context, db *sql.DB) error {
	columns := `
		id SERIAL PRIMARY KEY,
		shiki_id BIGINT UNIQUE NOT NULL,
		mal_id BIGINT UNIQUE,
		english TEXT,
		japanese TEXT,
		status TEXT,
		episodes INT NOT NULL,
		episodes_aired INT NOT NULL
	`
	indexName := "idx_shiki_id"
	indexColumn := "shiki_id"
	return pql.CreateTable(ctx, db, tableName, columns, indexName, indexColumn)
}

func Add(ctx context.Context, db *sql.DB, a *Anime) error {
	logger := logs.DefaultFromCtx(ctx)

	query := `
		INSERT INTO animes (shiki_id, mal_id, english, japanese, status, episodes, episodes_aired)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (shiki_id) DO NOTHING;
	`

	_, err := db.ExecContext(ctx, query, a.ShikiID, a.MalID, a.English,
		a.Japanese, a.Status, a.Episodes, a.EpisodesAired)

	if err != nil {
		logger.Error("Failed to add anime", "error", err, "anime", a)
		return err
	}

	logger.Info(fmt.Sprintf("Anime with ShikiID %d added successfully", a.ShikiID))
	return nil
}

func Find(ctx context.Context, db *sql.DB, fieldName string, fieldValue int) *Anime {
	logger := logs.DefaultFromCtx(ctx)

	query := fmt.Sprintf(`
		SELECT id, shiki_id, mal_id, english, japanese, status, episodes, episodes_aired
		FROM %s
		WHERE %s = $1;
	`, tableName, fieldName)

	var anime Anime
	err := db.QueryRowContext(ctx, query, fieldValue).Scan(
		&anime.ID,
		&anime.ShikiID,
		&anime.MalID,
		&anime.English,
		&anime.Japanese,
		&anime.Status,
		&anime.Episodes,
		&anime.EpisodesAired,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Warn("No anime found with", fieldName, fieldValue)
			return nil // Return nil if the user is not found
		}
		logger.Error("Failed to retrieve anime", fieldName, fieldValue, "error", err)
		return nil // Return nil if there's any other error
	}

	logger.Info(fmt.Sprintf("Anime with %s = %d retrieved successfully", fieldName, fieldValue))
	return &anime
}

// FindByID queries the "animes" table by its primary key (id).
func FindByID(ctx context.Context, db *sql.DB, id int) *Anime {
	return Find(ctx, db, "id", id)
}

// FindByShikiID queries the "animes" table by ShikiID.
func FindByShikiID(ctx context.Context, db *sql.DB, shikiID int) *Anime {
	return Find(ctx, db, "shiki_id", shikiID)
}

// FindByMalID queries the "animes" table by MalID.
func FindByMalID(ctx context.Context, db *sql.DB, malID int) *Anime {
	return Find(ctx, db, "mal_id", malID)
}

func Remove(ctx context.Context, db *sql.DB, id int) error {
	return pql.RemoveRecord(ctx, db, tableName, id)
}
