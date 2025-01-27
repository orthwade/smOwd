package pql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"smOwd/logs"
	"strings"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

// ConnectToDB opens a connection to the PostgreSQL database.
func ConnectToDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func ConnectToDatabasePostgres(ctx context.Context) *sql.DB {
	logger, ok := ctx.Value("logger").(*logs.Logger)

	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbSuperuser := os.Getenv("DB_SUPERUSER")
	dbSuperuserPassword := os.Getenv("DB_SUPERUSER_PASSWORD")
	dbDefaultName := os.Getenv("DB_DEFAULT_NAME")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbSuperuser, dbSuperuserPassword, dbHost, dbPort, dbDefaultName)

	db, err := sql.Open(dbDefaultName, connStr)

	if err != nil {
		logger.Fatal("Error connecting to default database postgres", "error", err)
	} else {
		logger.Info("Succesfully connected to default database postgres")
	}

	return db
}

func CheckIfDatabaseSubscriptionsExists(ctx context.Context, postgresDb *sql.DB) bool {
	var result bool
	dbName := os.Getenv("DB_NAME")

	logger, ok := ctx.Value("logger").(*logs.Logger)

	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	queryRow := fmt.Sprintf(`SELECT EXISTS (
	SELECT 1 FROM pg_catalog.pg_database WHERE datname = '%s'
	);`, dbName)

	err := postgresDb.QueryRow(queryRow).Scan(&result)

	if err != nil {
		logger.Fatal(fmt.Sprintf("Error checking if db %s exists", dbName), "fatal", err)
	}

	return result
}

func ConnectToDatabaseSubscriptions(ctx context.Context, postgresDb *sql.DB) *sql.DB {
	// Get the logger from the context
	logger, ok := ctx.Value("logger").(*logs.Logger)

	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	if !CheckIfDatabaseSubscriptionsExists(ctx, postgresDb) {
		logger.Fatal("Database subscriptions doesn't exists")
	} else {
		logger.Info("Database subscriptions found. Attempting connection")
	}

	postgresDb.Close()

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	db, err := ConnectToDB(connStr)
	if err != nil {
		logger.Fatal("Error connecting to database", "error", err)
	}
	logger.Info("Successfully connected to database", "db", dbName)

	return db
}

// func CheckRecord(db *sql.DB, tableName string, key int) bool {
// 	logger, ok := ctx.Value("logger").(*logs.Logger)
// 	if !ok {
// 		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
// 	}
// }

func CheckTable(ctx context.Context, db *sql.DB, tableName string) (bool, error) {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_name = $1
		);
	`
	var exists bool
	err := db.QueryRowContext(ctx, query, tableName).Scan(&exists)
	if err != nil {
		logger.Error("Failed to check if table exists", "error", err)
		return false, err
	}

	return exists, nil
}

func CreateTable(ctx context.Context, db *sql.DB, tableName string,
	columns string, indexName string, indexColumn string) error {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Create table query
	createTableQuery := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			%s
		);
	`, tableName, columns)

	_, err := db.ExecContext(ctx, createTableQuery)
	if err != nil {
		logger.Error("Failed to create table", "error", err)
		return err
	}

	// Create index query if index info is provided
	if indexName != "" && indexColumn != "" {
		createIndexQuery := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s (%s);`, indexName, tableName, indexColumn)
		_, err = db.ExecContext(ctx, createIndexQuery)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to create index on %s", indexColumn), "error", err)
			return err
		}
	}

	logger.Info(fmt.Sprintf("Table '%s' and index '%s' created successfully", tableName, indexName))
	return nil
}

func PrintTableColumnsNamesAndTypes(
	ctx context.Context, db *sql.DB, tableName string) {

	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
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

func AddRecord(ctx context.Context, db *sql.DB, tableName string, columns []string, values []interface{}, conflictColumn string) error {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Construct the column and value placeholders for the query
	columnsStr := "(" + strings.Join(columns, ", ") + ")"
	placeholders := "(" + strings.Repeat("$", len(values)-1) + "$" + fmt.Sprint(len(values)) + ")"

	// Construct the full SQL query with ON CONFLICT clause
	query := fmt.Sprintf(`
		INSERT INTO %s %s
		VALUES %s
		ON CONFLICT (%s) DO NOTHING;
	`, tableName, columnsStr, placeholders, conflictColumn)

	// Execute the query with the provided values
	_, err := db.ExecContext(ctx, query, values...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to add record to %s", tableName), "error", err)
		return err
	}

	logger.Info(fmt.Sprintf("Record added successfully to %s", tableName))
	return nil
}

func GetRecord(ctx context.Context, db *sql.DB, tableName string,
	idFieldName string, idValue int, dest interface{}) error {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	query := fmt.Sprintf(`
		SELECT * FROM %s WHERE %s = $1;
	`, tableName, idFieldName)

	// Execute the query and scan the result into the destination struct
	err := db.QueryRowContext(ctx, query, idValue).Scan(dest)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Warn(fmt.Sprintf("No record found in %s with %s = %v",
				tableName, idFieldName, idValue))
			return nil
		}
		logger.Error(fmt.Sprintf("Failed to retrieve record from %s", tableName), "error", err, idFieldName, idValue)
		return fmt.Errorf("failed to retrieve record from %s with %s = %v: %w", tableName, idFieldName, idValue, err)
	}

	logger.Info(fmt.Sprintf("Record from %s with %s = %v retrieved successfully", tableName, idFieldName, idValue))
	return nil
}

func RemoveRecord(ctx context.Context, db *sql.DB, tableName string, id int) error {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE id = $1;
	`, tableName)

	_, err := db.ExecContext(ctx, query, id)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to delete from %s", tableName), "error", err, "ID", id)
	} else {
		logger.Info(fmt.Sprintf("Deleted from %s", tableName), "ID", id)
	}

	return err
}

// DbExists checks if the specified database exists.
func DbExists(ctx context.Context, db *sql.DB, dbName string) bool {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	logger.Info("Checking if database exists", "dbName", dbName)

	var exists bool
	query := `SELECT EXISTS (
		SELECT 1 FROM pg_database WHERE datname = $1
	);`

	err := db.QueryRow(query, dbName).Scan(&exists)
	if err != nil {
		logger.Error("Error checking if database exists", "error", err)
		panic("Error checking if database exists") // Panic instead of log.Fatal
	}

	return exists
}

// CreateDatabase creates a new PostgreSQL database.
func CreateDatabase(db *sql.DB, dbName string) error {
	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	return err
}

// CreateTableNamedUsers checks if the users table exists and creates it if it does not.
func CreateTableNamedUsers(ctx context.Context, db *sql.DB) error {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Log the start of the table creation process
	logger.Info("Checking if users table is already created, if not -- creating it")

	// Create the table with the specified table name
	createTableSQL := `CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		enabled BOOL
	);`

	// Execute the SQL statement
	_, err := db.Exec(createTableSQL)
	if err != nil {
		logger.Error("Error creating table", "error", err)
		panic("Error creating table") // Panic instead of log.Fatal
	}

	// Log the success message
	logger.Info("Table users created successfully (or already exists)")

	return nil
}

// UserExists checks if a user with the specified userID exists in the users table.
func UserExists(ctx context.Context, db *sql.DB, userID int64) bool {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Log that we're checking if the user exists
	logger.Info("Checking if user exists", "userID", userID)

	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	if err != nil {
		logger.Error("Error checking if user exists", "userID", userID, "error", err)
		panic("Error checking if user exists") // Panic instead of log.Fatal
	}

	return exists
}

func SetEnabled(ctx context.Context, db *sql.DB, userID int64, newEnabledStatus bool) {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Update the user's enabled status
	_, err := db.Exec(`
		UPDATE users
		SET enabled = $1
		WHERE id = $2;`, newEnabledStatus, userID)
	if err != nil {
		logger.Error("Error updating user enabled status", "userID", userID, "error", err)
		panic("Error updating user enabled status")
	}

	// Check if the user exists after update
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	if err != nil {
		logger.Error("Error checking if user exists after update", "userID", userID, "error", err)
		panic("Error checking if user exists after update")
	}
}

func SetChatID(ctx context.Context, db *sql.DB, userID int64, chatID int64) {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Update the user's chat ID
	_, err := db.Exec(`
		UPDATE users
		SET chat_id = $1
		WHERE id = $2;`, chatID, userID)
	if err != nil {
		logger.Error("Error updating user chat ID", "userID", userID, "chatID", chatID, "error", err)
		panic("Error updating user chat ID")
	}

	// Check if the user exists after the update
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	if err != nil {
		logger.Error("Error checking if user exists after chat ID update", "userID", userID, "error", err)
		panic("Error checking if user exists after chat ID update")
	}
}

func GetEnabled(ctx context.Context, db *sql.DB, userID int64) bool {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Query the user's enabled status
	var enabled bool
	err := db.QueryRow(`
		SELECT enabled
		FROM users
		WHERE id = $1;`, userID).Scan(&enabled)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Info("No user found with that ID", "userID", userID)
		} else {
			logger.Error("Error retrieving user enabled status", "userID", userID, "error", err)
			panic("Error retrieving user enabled status")
		}
	} else {
		var str_ string
		if enabled {
			str_ = "ENABLED"
		} else {
			str_ = "DISABLED"
		}
		logger.Info("User status", "userID", userID, "status", str_)
	}

	return enabled
}

func GetChatID(ctx context.Context, db *sql.DB, userID int64) int64 {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Query the user's chat ID
	var chatID int64
	err := db.QueryRow(`
		SELECT chat_id
		FROM users
		WHERE id = $1;`, userID).Scan(&chatID)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Info("No user found with that ID", "userID", userID)
		} else {
			logger.Error("Error retrieving user chat ID", "userID", userID, "error", err)
			panic("Error retrieving user chat ID")
		}
	}

	return chatID
}

func AddAnimeId(ctx context.Context, db *sql.DB, userID int64, newAnimeID int64) {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Update the anime_ids array for the user if the anime_id is not already present
	_, err := db.Exec(`
		UPDATE users
		SET anime_ids = 
			CASE 
				WHEN NOT anime_ids @> ARRAY[$1] THEN array_append(anime_ids, $1)
				ELSE anime_ids
			END
		WHERE id = $2;`,
		newAnimeID, userID)
	if err != nil {
		logger.Error("Error adding anime ID", "userID", userID, "animeID", newAnimeID, "error", err)
		panic("Error adding anime ID")
	}

	logger.Info("Anime ID added if not already present", "userID", userID, "animeID", newAnimeID)
}

// GetSliceAnimeId retrieves the anime_ids array for a specific user
func GetSliceAnimeId(ctx context.Context, db *sql.DB, userID int64) []int64 {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Declare a slice to hold the anime_ids
	var animeIDs pq.Int64Array // This will automatically handle the PostgreSQL array type

	// Retrieve the anime_ids array for the specific user
	err := db.QueryRow(`
		SELECT anime_ids
		FROM users
		WHERE id = $1;`, userID).Scan(pq.Array(&animeIDs)) // Use pq.Array to scan the array
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Info("No user found with that ID", "userID", userID)
		} else {
			logger.Error("Error retrieving anime IDs", "userID", userID, "error", err)
			panic("Error retrieving anime IDs")
		}
	} else {
		// Log the retrieved anime_ids slice
		logger.Info("Retrieved anime IDs", "userID", userID, "animeIDs", animeIDs)
	}

	// Convert the pq.Int64Array to a regular []int64 slice and return
	return animeIDs
}

func SetUser(ctx context.Context, db *sql.DB, userID int64, enabled bool, anime_id []int64) {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Insert the user, or update if a conflict on the primary key occurs
	_, err := db.Exec(`
		INSERT INTO users (id, enabled, anime_ids)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE
		SET enabled = EXCLUDED.enabled, anime_ids = EXCLUDED.anime_ids`,
		userID, enabled, pq.Array(anime_id))
	if err != nil {
		logger.Error("Error inserting or updating user", "userID", userID, "error", err)
		panic("Error inserting or updating user")
	}

	logger.Info("User inserted or updated successfully", "userID", userID)
}

func DeleteColumn(ctx context.Context, db *sql.DB, table_name string, column_name string) {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Query to drop the anime_ids column
	query := fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s;`, table_name, column_name)

	// Execute the query
	_, err := db.Exec(query)
	if err != nil {
		logger.Error("Error deleting column", "table", table_name, "column", column_name, "error", err)
		panic("Error deleting column")
	}

	// Log success message
	logger.Info("Column removed successfully", "column", column_name)
}

func IsCustomTypeCreated(ctx context.Context, db *sql.DB, custom_type_name string) bool {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Check if the custom type exists
	var exists bool
	query := `
        SELECT EXISTS (
            SELECT 1
            FROM pg_type
            WHERE typname = $1
        );
    `
	err := db.QueryRow(query, custom_type_name).Scan(&exists)
	if err != nil {
		logger.Error("Failed to check if custom type exists", "custom_type_name", custom_type_name, "error", err)
		panic("Failed to check if custom type exists")
	}

	return exists
}

func CreateCustomTypeAnimeIdAndLastEpisode(ctx context.Context, db *sql.DB) {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Create the custom type if it doesn't exist
	var currentUser string
	err := db.QueryRow("SELECT current_user;").Scan(&currentUser)

	typeName := "anime_id_and_last_episode"
	createTypeQuery := fmt.Sprintf(`
        CREATE TYPE %s AS (
            anime_id INT,
            last_episode INT
        );
        `, typeName)
	_, err = db.Exec(createTypeQuery)
	if err != nil {
		logger.Error("Failed to create custom type", "type_name", typeName, "error", err)
		panic("Failed to create custom type")
	}

	logger.Info("Custom type created successfully", "type_name", typeName)
}

func CheckAnimeIdAndLastEpisodeColumn(ctx context.Context, db *sql.DB) {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	alterTableQuery := `
	ALTER TABLE users
	ADD COLUMN IF NOT EXISTS anime_data anime_id_and_last_episode[];
`
	_, err := db.Exec(alterTableQuery)
	if err != nil {
		logger.Error("Failed to alter table", "error", err)
		panic("Failed to alter table")
	}

	logger.Info("Table 'users' altered to add 'anime_data' column")
}

func CheckChatIdColumn(ctx context.Context, db *sql.DB) {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Query to add 'chat_id' column if it doesn't exist
	alterTableQuery := `
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS chat_id BIGINT;  -- Changed to BIGINT for larger IDs
	`

	_, err := db.Exec(alterTableQuery)
	if err != nil {
		logger.Error("Failed to alter table", "error", err)
		panic("Failed to alter table")
		return
	}

	logger.Info("Table 'users' altered to add 'chat_id' column")
}

type AnimeIDAndLastEpisode struct {
	AnimeID     int
	LastEpisode int
}

// GetSliceAnimeIdAndLastEpisode retrieves the anime data for a user from the database
func GetSliceAnimeIdAndLastEpisode(ctx context.Context, db *sql.DB, userID int64) ([]AnimeIDAndLastEpisode, error) {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Prepare the query to select the anime_data array for the user
	query := `SELECT anime_data FROM users WHERE id = $1`

	// Declare a variable to hold the anime data
	var animeData []string // Use []string for scanning array of tuples

	// Execute the query
	row := db.QueryRow(query, userID)
	if err := row.Scan(pq.Array(&animeData)); err != nil {
		// If no data is found or any error occurs
		if err == sql.ErrNoRows {
			logger.Info("No anime data found for user", "userID", userID)
			return nil, fmt.Errorf("no anime data found for user ID %d", userID)
		}
		logger.Error("Failed to retrieve anime data", "userID", userID, "error", err)
		return nil, fmt.Errorf("failed to retrieve anime data: %v", err)
	}

	// Convert the array elements into a slice of AnimeIDAndLastEpisode
	var result []AnimeIDAndLastEpisode
	for _, elem := range animeData {
		// Assuming the array elements are in the format "(anime_id, last_episode)"
		var animeID, lastEpisode int
		// Parsing the array element into AnimeIDAndLastEpisode struct
		if _, err := fmt.Sscanf(elem, "(%d,%d)", &animeID, &lastEpisode); err != nil {
			logger.Error("Error parsing array element", "element", elem, "error", err)
			continue
		}
		result = append(result, AnimeIDAndLastEpisode{
			AnimeID:     animeID,
			LastEpisode: lastEpisode,
		})
	}

	return result, nil
}

func AddAnimeIdAndLastEpisode(ctx context.Context, db *sql.DB, userID int64, animeID int, lastEpisode int) {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Step 1: Retrieve the current anime_data for userID
	var currentAnimeData []AnimeIDAndLastEpisode
	query := "SELECT anime_data FROM users WHERE id = $1"
	rows, err := db.Query(query, userID) // Get data for userID
	if err != nil {
		logger.Error("Failed to retrieve data", "userID", userID, "error", err)
		panic("Failed to retrieve data")
	}
	defer rows.Close()

	// Assuming anime_data is a valid PostgreSQL array of strings
	for rows.Next() {
		var animeData []string                                  // Slice of strings to hold the PostgreSQL array
		if err := rows.Scan(pq.Array(&animeData)); err != nil { // Use pq.Array to scan into slice
			logger.Error("Failed to scan rows", "userID", userID, "error", err)
			panic("Failed to scan rows")
		}

		// Convert the array of strings into AnimeIDAndLastEpisode
		for _, v := range animeData {
			// Assuming the data is in the form "(anime_id, last_episode)"
			var animeID, lastEpisode int
			fmt.Sscanf(v, "(%d,%d)", &animeID, &lastEpisode)
			currentAnimeData = append(currentAnimeData, AnimeIDAndLastEpisode{AnimeID: animeID, LastEpisode: lastEpisode})
		}
	}

	// Step 2: Append the new anime element {anime_id: animeID, last_episode: lastEpisode}
	newAnime := AnimeIDAndLastEpisode{
		AnimeID:     animeID,
		LastEpisode: lastEpisode,
	}
	currentAnimeData = append(currentAnimeData, newAnime)

	// Step 3: Update the anime_data in the database
	updateQuery := `
		UPDATE users
		SET anime_data = $1
		WHERE id = $2;
	`

	// Prepare the updated anime data as PostgreSQL array
	var updatedAnimeData []string
	for _, anime := range currentAnimeData {
		updatedAnimeData = append(updatedAnimeData, fmt.Sprintf("(%d,%d)", anime.AnimeID, anime.LastEpisode))
	}

	// Execute the update query with the new array
	_, err = db.Exec(updateQuery, pq.Array(updatedAnimeData), userID)
	if err != nil {
		logger.Error("Failed to update anime_data", "userID", userID, "error", err)
		panic("Failed to update anime_data")
	}

	logger.Info("Successfully appended new anime data and updated the database", "userID", userID)
}
func UpdateAnimeIdAndLastEpisode(ctx context.Context, db *sql.DB, userID int64, animeID int, lastEpisode int) {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Step 1: Start a new transaction
	tx, err := db.Begin()
	if err != nil {
		logger.Error("Failed to start transaction", "userID", userID, "error", err)
		panic("Failed to start transaction")
	}
	defer tx.Rollback() // Ensure rollback in case of an error

	// Step 2: Retrieve the current anime_data for userID
	var currentAnimeData []AnimeIDAndLastEpisode
	query := "SELECT anime_data FROM users WHERE id = $1"
	rows, err := tx.Query(query, userID) // Get data for userID
	if err != nil {
		logger.Error("Failed to retrieve data", "userID", userID, "error", err)
		panic("Failed to retrieve data")
	}
	defer rows.Close()

	// Assuming anime_data is a valid PostgreSQL array of strings
	for rows.Next() {
		var animeData []string
		if err := rows.Scan(pq.Array(&animeData)); err != nil {
			logger.Error("Failed to scan rows", "userID", userID, "error", err)
			panic("Failed to scan rows")
		}

		// Convert the array of strings into AnimeIDAndLastEpisode
		for _, v := range animeData {
			var id, ep int
			fmt.Sscanf(v, "(%d,%d)", &id, &ep)
			currentAnimeData = append(currentAnimeData, AnimeIDAndLastEpisode{AnimeID: id, LastEpisode: ep})
		}
	}

	// Step 3: Update the lastEpisode for the given animeID if it exists
	var updatedAnimeData []AnimeIDAndLastEpisode
	animeFound := false
	for _, anime := range currentAnimeData {
		if anime.AnimeID == animeID {
			// Update the last episode
			anime.LastEpisode = lastEpisode
			animeFound = true
		}
		updatedAnimeData = append(updatedAnimeData, anime)
	}

	// If the animeID wasn't found, append the new animeID and lastEpisode pair
	if !animeFound {
		updatedAnimeData = append(updatedAnimeData, AnimeIDAndLastEpisode{AnimeID: animeID, LastEpisode: lastEpisode})
	}

	// Step 4: Prepare the updated anime data as a PostgreSQL array
	var updatedAnimeDataStrings []string
	for _, anime := range updatedAnimeData {
		updatedAnimeDataStrings = append(updatedAnimeDataStrings, fmt.Sprintf("(%d,%d)", anime.AnimeID, anime.LastEpisode))
	}

	// Step 5: Update the anime_data in the database within the transaction
	updateQuery := `
		UPDATE users
		SET anime_data = $1
		WHERE id = $2;
	`
	_, err = tx.Exec(updateQuery, pq.Array(updatedAnimeDataStrings), userID)
	if err != nil {
		logger.Error("Failed to update anime_data", "userID", userID, "error", err)
		panic("Failed to update anime_data")
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		logger.Error("Failed to commit transaction", "userID", userID, "error", err)
		panic("Failed to commit transaction")
	}

	logger.Info("Successfully updated anime data for user", "userID", userID)
}

func RemoveAnimeIdAndLastEpisode(ctx context.Context, db *sql.DB, userID int64, animeID int) {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Step 1: Retrieve the current anime_data for userID
	var currentAnimeData []AnimeIDAndLastEpisode
	query := "SELECT anime_data FROM users WHERE id = $1"
	rows, err := db.Query(query, userID) // Get data for userID
	if err != nil {
		logger.Error("Failed to retrieve data", "userID", userID, "error", err)
		panic("Failed to retrieve data")
	}
	defer rows.Close()

	// Assuming anime_data is a valid PostgreSQL array of strings
	for rows.Next() {
		var animeData []string                                  // Slice of strings to hold the PostgreSQL array
		if err := rows.Scan(pq.Array(&animeData)); err != nil { // Use pq.Array to scan into slice
			logger.Error("Failed to scan rows", "userID", userID, "error", err)
			panic("Failed to scan rows")
		}

		// Convert the array of strings into AnimeIDAndLastEpisode
		for _, v := range animeData {
			// Assuming the data is in the form "(anime_id, last_episode)"
			var id, ep int
			fmt.Sscanf(v, "(%d,%d)", &id, &ep)
			currentAnimeData = append(currentAnimeData, AnimeIDAndLastEpisode{AnimeID: id, LastEpisode: ep})
		}
	}

	// Step 2: Remove the animeID from the currentAnimeData slice
	var updatedAnimeData []AnimeIDAndLastEpisode
	for _, anime := range currentAnimeData {
		if anime.AnimeID != animeID { // If it's not the animeID to remove, keep it
			updatedAnimeData = append(updatedAnimeData, anime)
		}
	}

	// Step 3: Prepare the updated anime data as a PostgreSQL array (formatted as strings)
	var updatedAnimeDataStrings []string
	for _, anime := range updatedAnimeData {
		updatedAnimeDataStrings = append(updatedAnimeDataStrings, fmt.Sprintf("(%d,%d)", anime.AnimeID, anime.LastEpisode))
	}

	// Step 4: Update the anime_data in the database with the new array (after removal)
	updateQuery := `
		UPDATE users
		SET anime_data = $1
		WHERE id = $2;
	`
	_, err = db.Exec(updateQuery, pq.Array(updatedAnimeDataStrings), userID)
	if err != nil {
		logger.Error("Failed to update anime_data", "userID", userID, "error", err)
		panic("Failed to update anime_data")
	}

	logger.Info("Successfully removed anime data for user", "userID", userID)
}
