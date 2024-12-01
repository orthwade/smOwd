package telegram_bot

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"smOwd/pql"
	"smOwd/search_anime"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

type UserAndMessage struct {
	UserID int64 // Telegram user ID
	ChatID int64 // Telegram chat ID
	Text   string
}

func removeFirstCharIfPresent(s string, char rune) string {
	// Check if the string is not empty and the first character matches the provided char
	if len(s) > 0 && rune(s[0]) == char {
		// Slice the string to remove the first character
		return s[1:]
	}
	return s // Return the string as is if the first character doesn't match
}

// Define a custom type for handleUpdateMode
type HandleUpdateMode int

// Declare constants using iota to represent the values
const (
	HandleUpdateModeBasic HandleUpdateMode = iota
	HandleUpdateModeSearch
	HandleUpdateModeSubscribe
)

// String method for Color type to print meaningful names
func (c HandleUpdateMode) String() string {
	return [...]string{"Basic", "Search"}[c]
}

var usersMapHandleUpdMode = make(map[int64]HandleUpdateMode)

var usersMapLastAnimeIDList = make(map[int64][]int64)
var usersMapLastAnimeNameList = make(map[int64][]string)
var usersMapLastAnimeLastEpisodeList = make(map[int64][]int)

// Function to create an inline keyboard from listText and maxCols
func CreateInlineKeyboard(listText []string, maxCols int) tgbotapi.InlineKeyboardMarkup {
	// Create an empty slice to hold the keyboard rows
	var keyboard [][]tgbotapi.InlineKeyboardButton

	// Create buttons and group them into rows of maxCols
	for i := 0; i < len(listText); i += maxCols {
		// Get the slice of text for the current row
		end := i + maxCols
		if end > len(listText) {
			end = len(listText)
		}
		row := listText[i:end]

		// Create InlineKeyboardButton for each text in the row
		var buttons []tgbotapi.InlineKeyboardButton
		for _, text := range row {
			button := tgbotapi.NewInlineKeyboardButtonData(text, text) // Using text as callback data
			buttons = append(buttons, button)
		}

		// Add the row of buttons to the keyboard
		keyboard = append(keyboard, buttons)
	}

	// Return the inline keyboard markup
	return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

func GeneralMessage(msg_str string, keyboard tgbotapi.InlineKeyboardMarkup, msg tgbotapi.MessageConfig) (string, tgbotapi.InlineKeyboardMarkup, tgbotapi.MessageConfig) {
	// Modify msg_str by appending the text
	msg_str += "Please choose one of the options:\n"

	// Inline keyboard for subscription options
	keyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Enable\nnotifications", "enable"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Disable\nnotifications", "disable"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Show\nsubscriptions", "subscriptions"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Search anime by name", "search"),
		),
	)

	// Modify the msg object by adding the keyboard to it
	msg.ReplyMarkup = keyboard

	// Return the modified values
	return msg_str, keyboard, msg
}

// Unified function to handle both messages and inline button callbacks
func handleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *sql.DB) {
	var user_and_msg UserAndMessage
	var msg tgbotapi.MessageConfig
	var err error
	skip := true
	if update.Message != nil { // Handle regular messages like /start
		// Extract chatID and message text
		user_and_msg.ChatID = update.Message.Chat.ID
		user_and_msg.UserID = int64(update.Message.From.ID)
		user_and_msg.Text = removeFirstCharIfPresent(update.Message.Text, '/')
		skip = false

	} else if update.CallbackQuery != nil { // Handle inline button callback queries
		user_and_msg.Text = update.CallbackQuery.Data
		user_and_msg.UserID = int64(update.CallbackQuery.From.ID)
		user_and_msg.ChatID = update.CallbackQuery.Message.Chat.ID
		skip = false
		defer bot.AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, "Done"))
	}

	if !skip {

		if pql.UserExists(db, user_and_msg.UserID) {
			fmt.Println("User exists in db, user id: ", user_and_msg.UserID)
		} else {
			enabled := false
			animeIDs := "{}"

			_, err = db.Exec("INSERT INTO users (id, enabled, anime_ids) VALUES ($1, $2, $3)", user_and_msg.UserID, enabled, animeIDs)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("User inserted successfully")

		}
		_, ok := usersMapHandleUpdMode[user_and_msg.UserID]
		if !ok {
			usersMapHandleUpdMode[user_and_msg.UserID] = HandleUpdateModeBasic
		}

		// If the user sends "/start", show the inline keyboard with subscribe/unsubscribe buttons
		msg_str := ""
		msg = tgbotapi.NewMessage(user_and_msg.ChatID, "")

		handle_update_mode := usersMapHandleUpdMode[user_and_msg.UserID]

		var keyboard tgbotapi.InlineKeyboardMarkup

		if handle_update_mode == HandleUpdateModeBasic {
			if user_and_msg.Text == "enable" {
				pql.SetEnabled(db, user_and_msg.UserID, true)
				pql.GetEnabled(db, user_and_msg.UserID)
				msg_str += "You have enabled subscription notifications!\n"
			} else if user_and_msg.Text == "disable" {
				pql.SetEnabled(db, user_and_msg.UserID, false)
				pql.GetEnabled(db, user_and_msg.UserID)
				msg_str += "You have disabled subscription notifications.\n"
			} else if user_and_msg.Text == "subscriptions" {
				slice_anime_id_and_last_episode, err := pql.GetSliceAnimeIdAndLastEpisode(db, user_and_msg.UserID)

				if err != nil {
					fmt.Println("Error reading your list of subscritions")
				}
				// if len(slice_anime_id) == 0 {
				// slice_anime_id = append(slice_anime_id, 5081)
				// }
				if len(slice_anime_id_and_last_episode) == 0 {
					msg_str += "You are not subscribed to any anime notifications.\n"
				} else {
					msg_str += "You are subscribed to notificatins about following titles:\n"
					for i, id_and_last_episode := range slice_anime_id_and_last_episode {
						anime := search_anime.SearchAnimeById(int64(id_and_last_episode.AnimeID))
						msg_str += strconv.Itoa(i+1) + ". "
						msg_str += anime.Data.Animes[0].English
						msg_str += "\n"
					}
				}

			} else if user_and_msg.Text == "search" {
				usersMapHandleUpdMode[user_and_msg.UserID] = HandleUpdateModeSearch
				msg_str += "Enter a name of anime in english to search it.\n"

			}
			if usersMapHandleUpdMode[user_and_msg.UserID] == HandleUpdateModeBasic {
				msg_str, keyboard, msg = GeneralMessage(msg_str, keyboard, msg)
			}

		} else if handle_update_mode == HandleUpdateModeSearch {
			animeResp := search_anime.SearchAnimeByName(user_and_msg.Text)

			if len(animeResp.Data.Animes) > 0 {
				incomplete_count := 0
				LastAnimeSearchList := make([]int64, 0)
				LastAnimeSearchListName := make([]string, 0)
				LastAnimeSearchListLastEpisode := make([]int, 0)

				var list_button_text []string
				for i, anime := range animeResp.Data.Animes {
					msg_str += strconv.Itoa(i+1) + ". " + anime.English + "/ " + anime.Japanese + "\n"
					msg_str += anime.URL + "\n"
					animeID, err := strconv.Atoi(anime.ID)
					if err != nil {
						fmt.Println("Error reading animeID")
					}

					LastAnimeSearchList = append(LastAnimeSearchList, int64(animeID))
					LastAnimeSearchListName = append(LastAnimeSearchListName, anime.English)
					LastAnimeSearchListLastEpisode = append(LastAnimeSearchListLastEpisode, int(anime.EpisodesAired))

					if anime.Status != "released" {
						incomplete_count++
						list_button_text = append(list_button_text, strconv.Itoa(i+1))
					} else {
						msg_str += "Fully Released!\n"
					}
				}

				if incomplete_count == 0 {
					msg_str += "All found animes are complete. No need for notifications.\n"
					usersMapHandleUpdMode[user_and_msg.UserID] = HandleUpdateModeBasic
					msg_str, keyboard, msg = GeneralMessage(msg_str, keyboard, msg)
				} else {
					usersMapLastAnimeIDList[user_and_msg.UserID] = LastAnimeSearchList
					usersMapLastAnimeNameList[user_and_msg.UserID] = LastAnimeSearchListName
					usersMapLastAnimeLastEpisodeList[user_and_msg.UserID] = LastAnimeSearchListLastEpisode

					msg_str += "Some of the found animes are not complete\n"
					msg_str += "You can subscribe to be notified if new episodes are aired\n"
					msg_str += "Choose the number of anime from the list above to subscribe to:\n"
					keyboard = CreateInlineKeyboard(list_button_text, 5)
					msg.ReplyMarkup = keyboard
					usersMapHandleUpdMode[user_and_msg.UserID] = HandleUpdateModeSubscribe
				}

			}
		} else if usersMapHandleUpdMode[user_and_msg.UserID] == HandleUpdateModeSubscribe {
			i, err := strconv.Atoi(user_and_msg.Text)
			if err != nil {
				fmt.Println("Error getting anime ID from user's message")
			}
			i--
			animeID := usersMapLastAnimeIDList[user_and_msg.UserID][i]
			animeName := usersMapLastAnimeNameList[user_and_msg.UserID][i]
			lastEpisode := usersMapLastAnimeLastEpisodeList[user_and_msg.UserID][i]
			usersMapHandleUpdMode[user_and_msg.UserID] = HandleUpdateModeBasic
			pql.AddAnimeIdAndLastEpisode(db, user_and_msg.UserID, int(animeID), lastEpisode)
			checkSubscriptionAfterAdd, err := pql.GetSliceAnimeIdAndLastEpisode(db, user_and_msg.UserID)
			if len(checkSubscriptionAfterAdd) == 0 {
				log.Fatal("Something went wrong.\n")
			}
			msg_str += "You have subscribed to anime: " + animeName + "!\n\n"
			msg_str, keyboard, msg = GeneralMessage(msg_str, keyboard, msg)
		}

		msg.Text = msg_str
		msg.DisableWebPagePreview = true

		// Send the response message
		bot.Send(msg)
	}
}

func SignalAnimeComplete(db *sql.DB, userID int64, animeName string) {

}

func SignalAnimeNewEpisodes(db *sql.DB, userID int64, animeName string, newEpisode int) {

}

func processUsers(db *sql.DB) {
	// Query all users from the users table
	rows, err := db.Query("SELECT id, enabled FROM users")
	if err != nil {
		log.Fatal("Failed to query users:", err)
	}
	defer rows.Close()

	// Process each user
	for rows.Next() {
		var userID int64
		var enabled bool
		if err := rows.Scan(&userID, &enabled); err != nil {
			log.Fatal("Failed to scan row:", err)
		}

		// Example processing: Print the user ID and their enabled status
		fmt.Printf("Processing User ID: %d\n", userID)

		list, err := pql.GetSliceAnimeIdAndLastEpisode(db, userID)
		if err != nil {
			fmt.Printf("Error reading subscription for user: %d", userID)
		}

		if len(list) == 0 {
			fmt.Printf("Used ID %d is not subscribed to any anime notifications.\n", userID)
		} else {
			for _, id_and_last_episode := range list {

				animeID := id_and_last_episode.AnimeID

				anime := search_anime.SearchAnimeById(int64(animeID))
				animeName := anime.Data.Animes[0].English
				storedAnimeLastEpisode := id_and_last_episode.LastEpisode
				actualAnimeLastEpisode := anime.Data.Animes[0].EpisodesAired
				animeStatus := anime.Data.Animes[0].Status
				if animeStatus == "released" {
					SignalAnimeComplete(db, userID, animeName)
					pql.RemoveAnimeIdAndLastEpisode(db, userID, animeID)
				} else if actualAnimeLastEpisode > storedAnimeLastEpisode {
					SignalAnimeNewEpisodes(db, userID, animeName, actualAnimeLastEpisode)
					pql.UpdateAnimeIdAndLastEpisode(db, userID, animeID, actualAnimeLastEpisode)
				} else {
					fmt.Printf("Nothing new for User %d.\n", userID)
				}
			}
		}

		// You can add your custom processing logic here
	}

	// Check for errors in the row iteration
	if err := rows.Err(); err != nil {
		log.Fatal("Error reading rows:", err)
	}
}

func StartBotAndHandleUpdates(db *sql.DB) {
	// Get the Telegram bot token from an environment variable
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not set")
	}

	// Initialize the bot
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	// Set bot to debug mode (optional)
	bot.Debug = true
	fmt.Println("Authorized on account", bot.Self.UserName)

	// Configure the update channel (long polling)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Get updates (messages and callback queries) from Telegram
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatal(err)
	}

	// Create a channel to synchronize processUsers with handleUpdate
	processUsersChan := make(chan bool)

	// Create a context and cancel function for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up a goroutine to listen for OS signals and trigger shutdown
	go func() {
		// Channel for receiving OS termination signals (e.g., CTRL+C or kill command)
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		<-signalChan // Block until a signal is received
		log.Println("Received shutdown signal, shutting down gracefully...")
		cancel() // Trigger the shutdown process
	}()

	// Start a goroutine to handle periodic user processing every second
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				processUsersChan <- true // Send signal to process users every second
			case <-ctx.Done():
				log.Println("Stopping user processing due to shutdown signal.")
				return
			}
		}
	}()

	// Main loop: process incoming updates and handle periodic user processing
	for {
		select {
		case update := <-updates:
			// Handle incoming updates (messages and callback queries)
			handleUpdate(bot, update, db)
		case <-processUsersChan:
			// This block is triggered every 1 second to process users
			processUsers(db)
		case <-ctx.Done():
			// Graceful shutdown of the main loop
			log.Println("Shutting down the bot.")
			return
		}
	}
}
