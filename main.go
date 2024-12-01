package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"smOwd/pql"
	"smOwd/telegram_bot"
)

func TestPQL() *sql.DB {
	user_name := os.Getenv("PQL_USER_NAME")
	password := os.Getenv("PQL_PASSWORD")

	connStr := fmt.Sprintf("user=%s password=%s sslmode=disable dbname=postgres", user_name, password)
	db, err := pql.ConnectToDB(connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	dbName := "smowd_users"
	if pql.DbExists(db, dbName) {
		fmt.Println("Database already exists: ", dbName)
	} else {
		err := pql.CreateDatabase(db, dbName)
		if err != nil {
			log.Fatal("Error creating database: ", err)
		} else {
			fmt.Println("Database created successfully: ", dbName)
		}
	}
	db.Close()
	connStr = fmt.Sprintf("user=%s password=%s sslmode=disable dbname=%s", user_name, password, dbName)
	db, err = pql.ConnectToDB(connStr)

	if err != nil {
		msg := fmt.Sprintf("Error connecting to %s:", dbName)
		log.Fatal(msg, err)
	}

	fmt.Printf("Connected to %s\n", dbName)

	err = pql.CreateTableNamedUsers(db)
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Table users created successfully")
	}

	custom_type_name := "anime_id_and_last_episode"

	if pql.IsCustomTypeCreated(db, custom_type_name) {
		fmt.Printf("Column %s is already created\n", custom_type_name)
	} else {
		fmt.Printf("Column %s is not created\n", custom_type_name)
		pql.CreateCustomTypeAnimeIdAndLastEpisode(db)
	}
	pql.CheckAnimeIdAndLastEpisodeColumn(db)
	pql.CheckChatIdColumn(db)

	return db
}

func main() {
	db := TestPQL()
	defer db.Close()
	// pql.DeleteColumn(db, "users", "anime_ids")
	pql.PrintTableColumnsNamesAndTypes(db, "users")
	telegram_bot.StartBotAndHandleUpdates(db)
}
