package pql

import (
	"database/sql"
	"fmt"
	"log"

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

// dbExists checks if the specified database exists.
func DbExists(db *sql.DB, dbName string) bool {
	var exists bool
	query := `SELECT EXISTS (
		SELECT 1 FROM pg_database WHERE datname = $1
	);`
	err := db.QueryRow(query, dbName).Scan(&exists)
	if err != nil {
		log.Fatal("Error checking if database exists:", err)
	}
	return exists
}

// CreateDatabase creates a new PostgreSQL database.
func CreateDatabase(db *sql.DB, dbName string) error {
	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	return err
}

func CreateTableNamedUsers(db *sql.DB) error {
	// Create the table with the specified table name
	createTableSQL := `CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		enabled BOOL
	);`

	_, err := db.Exec(createTableSQL)
	return err
}
func UserExists(db *sql.DB, userID int64) bool {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	if err != nil {
		log.Fatal(err)
	}
	return exists
}

func SetEnabled(db *sql.DB, userID int64, newEnabledStatus bool) {
	_, err := db.Exec(`
		UPDATE users
		SET enabled = $1
		WHERE id = $2;`, newEnabledStatus, userID)
	if err != nil {
		log.Fatal(err)
	}
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	if err != nil {
		log.Fatal(err)
	}
}

func GetEnabled(db *sql.DB, userID int64) bool {
	var enabled bool
	err := db.QueryRow(`
		SELECT enabled
		FROM users
		WHERE id = $1;`, userID).Scan(&enabled)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No user found with that ID.")
		} else {
			log.Fatal(err)
		}
	} else {
		var str_ string
		if enabled {
			str_ = "ENABLED"
		} else {
			str_ = "DISABLED"
		}
		fmt.Printf("User %d status is %s\n", userID, str_)
	}

	return enabled
}

func AddAnimeId(db *sql.DB, userID int64, newAnimeID int64) {
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
		log.Fatal(err)
	}
	fmt.Println("Anime ID added if not already present.")
}

// GetSliceAnimeId retrieves the anime_ids array for a specific user
func GetSliceAnimeId(db *sql.DB, userID int64) []int64 {
	// Declare a slice to hold the anime_ids
	var animeIDs pq.Int64Array // This will automatically handle the PostgreSQL array type

	// Retrieve the anime_ids array for the specific user
	err := db.QueryRow(`
		SELECT anime_ids
		FROM users
		WHERE id = $1;`, userID).Scan(pq.Array(&animeIDs)) // Use pq.Array to scan the array
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No user found with that ID.")
		} else {
			log.Fatal(err)
		}
	} else {
		// Print the retrieved anime_ids slice
		fmt.Printf("Anime IDs for user %d: %v\n", userID, animeIDs)
	}

	// Convert the pq.Int64Array to a regular []int64 slice and return
	return animeIDs
}

func SetUser(db *sql.DB, userID int64, enabled bool, anime_id []int64) {

	// Insert the user, or update if a conflict on the primary key occurs
	_, err := db.Exec(`
		INSERT INTO users (id, enabled, anime_ids)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE
		SET enabled = EXCLUDED.enabled, anime_ids = EXCLUDED.anime_ids`,
		userID, enabled, pq.Array(anime_id))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("User inserted or updated successfully")
}

func DeleteColumn(db *sql.DB, table_name string, column_name string) {
	// Query to drop the anime_ids column
	query := fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s;`, table_name, column_name)

	// Execute the query
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}

	// Print success message
	fmt.Printf("Column %s has been removed successfully.", column_name)
}

func IsCustomTypeCreated(db *sql.DB, custom_type_name string) bool {
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
		log.Fatal("Failed to check if type exists:", err)
	}

	return exists
}

func CreateCustomTypeAnimeIdAndLastEpisode(db *sql.DB) {
	// Create the custom type if it doesn't exist
	typeName := "anime_id_and_last_episode"
	createTypeQuery := fmt.Sprintf(`
        CREATE TYPE %s AS (
            anime_id INT,
            last_episode INT
        );
        `, typeName)
	_, err := db.Exec(createTypeQuery)
	if err != nil {
		log.Fatal("Failed to create type:", err)
	}
	fmt.Printf("Custom type '%s' created successfully!\n", typeName)
}

func CheckAnimeIdAndLastEpisodeColumn(db *sql.DB) {
	alterTableQuery := `
	ALTER TABLE users
	ADD COLUMN IF NOT EXISTS anime_data anime_id_and_last_episode[];
`
	_, err := db.Exec(alterTableQuery)
	if err != nil {
		log.Fatal("Failed to alter table:", err)
	}
	fmt.Println("Table 'users' altered to add 'anime_data' column.")
}

type AnimeIDAndLastEpisode struct {
	AnimeID     int
	LastEpisode int
}

// GetSliceAnimeIdAndLastEpisode retrieves the anime data for a user from the database
func GetSliceAnimeIdAndLastEpisode(db *sql.DB, userID int64) ([]AnimeIDAndLastEpisode, error) {
	// Prepare the query to select the anime_data array for the user
	query := `SELECT anime_data FROM users WHERE id = $1`

	// Declare a variable to hold the anime data
	var animeData []string // Use []string for scanning array of tuples

	// Execute the query
	row := db.QueryRow(query, userID)
	if err := row.Scan(pq.Array(&animeData)); err != nil {
		// If no data is found or any error occurs
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no anime data found for user ID %d", userID)
		}
		return nil, fmt.Errorf("failed to retrieve anime data: %v", err)
	}

	// Convert the array elements into a slice of AnimeIDAndLastEpisode
	var result []AnimeIDAndLastEpisode
	for _, elem := range animeData {
		// Assuming the array elements are in the format "(anime_id, last_episode)"
		var animeID, lastEpisode int
		// Parsing the array element into AnimeIDAndLastEpisode struct
		if _, err := fmt.Sscanf(elem, "(%d,%d)", &animeID, &lastEpisode); err != nil {
			log.Printf("Error parsing array element %v: %v", elem, err)
			continue
		}
		result = append(result, AnimeIDAndLastEpisode{
			AnimeID:     animeID,
			LastEpisode: lastEpisode,
		})
	}

	return result, nil
}
