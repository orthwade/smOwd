package animes

import (
	"context"
	"database/sql"
	"smOwd/pql"
)

const tableName = "animes"

type Anime struct {
	ID            int //PRIMARY KEY
	ShikiID       int
	MalId         int
	English       string
	Japanese      string
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
		mal_id INT UNIQUE,
		english TEXT,
		japanese TEXT,
		episodes INT NOT NULL,
		episodes_aired INT NOT NULL
	`
	indexName := "idx_shiki_id"
	indexColumn := "shiki_id"
	return pql.CreateTable(ctx, db, tableName, columns, indexName, indexColumn)
}

func Add(ctx context.Context, db *sql.DB, a Anime) error {
	columns := []string{"shiki_id", "mal_id", "english", "japanese", "episodes", "episodes_aired"}
	values := []interface{}{a.ShikiID, a.MalId, a.English, a.Japanese, a.Episodes, a.EpisodesAired}
	return pql.AddRecord(ctx, db, tableName, columns, values, "shiki_id")
}

func Get(ctx context.Context, db *sql.DB, id int) (*Anime, error) {
	var a Anime
	err := pql.GetRecord(ctx, db, tableName, "id", id, &a)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func Remove(ctx context.Context, db *sql.DB, id int) error {
	return pql.RemoveRecord(ctx, db, tableName, id)
}
