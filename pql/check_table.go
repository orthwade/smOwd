package pql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"smOwd/logs"
	"strings"

	_ "github.com/lib/pq"
)

func PrintTableColumnsNamesAndTypes(ctx context.Context, db *sql.DB, tableName string) {

	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}
	// Query to get table structure
	query := `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = $1;
	`

	// Execute the query
	rows, err := db.Query(query, tableName)
	if err != nil {
		logger.Fatal("Query failed", "error", err)
	}
	defer rows.Close()

	// Print the table structure
	fmt.Printf("Structure of %s table:\n", tableName)
	fmt.Printf("%-20s %-15s %-10s\n", "Column Name", "Data Type", "Is Nullable")
	fmt.Println(strings.Repeat("-", 50))

	for rows.Next() {
		var columnName, dataType, isNullable string
		if err := rows.Scan(&columnName, &dataType, &isNullable); err != nil {
			logger.Fatal("Failed to scan row", "error", err)
		}
		fmt.Printf("%-20s %-15s %-10s\n", columnName, dataType, isNullable)
	}

	// Check for errors after iterating through rows
	if err := rows.Err(); err != nil {
		logger.Fatal("Error iterating rows", "error", err)
	}
}
