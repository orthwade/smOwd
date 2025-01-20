package pql

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"smOwd/logs"

	_ "github.com/lib/pq"
)

func PrintTableColumnsNamesAndTypes(ctx context.Context, db *sql.DB, table_name string) {

	logger := ctx.Value("logger").(*logs.Logger)

	// Query to get column names and types from the users table
	query := fmt.Sprintf(`
		SELECT c.column_name, c.data_type, t.typname AS array_element_type
		FROM information_schema.columns c
		LEFT JOIN pg_catalog.pg_type t 
			ON c.udt_name = t.typname
		WHERE c.table_name = '%s'
	`, table_name)

	// Execute the query
	rows, err := db.Query(query)
	if err != nil {
		logger.Fatal("Error getting column names and types", "error", err)
	}
	defer rows.Close()

	// Print the column names and types
	fmt.Println("Column Name | Data Type       | Array Element Type")
	fmt.Println("---------------------------------------------------")
	for rows.Next() {
		var columnName, dataType, arrayElementType sql.NullString
		err := rows.Scan(&columnName, &dataType, &arrayElementType)
		if err != nil {
			log.Fatal(err)
		}

		// Check if the column is an array
		if dataType.String == "ARRAY" && arrayElementType.Valid {
			// If it's an array, print the element type
			fmt.Printf("%-12s | %-15s | %s\n", columnName.String, dataType.String, arrayElementType.String)
		} else {
			// If it's not an array, just print the data type
			fmt.Printf("%-12s | %-15s | N/A\n", columnName.String, dataType.String)
		}
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}
