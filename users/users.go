package users

import (
	"context"
	"database/sql"
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

func Add(ctx context.Context, db *sql.DB, u User) error {
	columns := []string{"telegram_id", "first_name", "last_name", "user_name", "language_code", "is_bot", "enabled"}
	values := []interface{}{u.TelegramID, u.FirstName, u.LastName, u.UserName, u.LanguageCode, u.IsBot, u.Enabled}
	return pql.AddRecord(ctx, db, tableName, columns, values, "telegram_id")
}

func Get(ctx context.Context, db *sql.DB, id int) (*User, error) {
	var u User
	err := pql.GetRecord(ctx, db, tableName, "id", id, &u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func Remove(ctx context.Context, db *sql.DB, id int) error {
	return pql.RemoveRecord(ctx, db, tableName, id)
}
