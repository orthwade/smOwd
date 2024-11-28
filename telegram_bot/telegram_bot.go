package telegram_bot

import (
	"database/sql"
	"fmt"
	"log"
	"os"

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

		// If the user sends "/start", show the inline keyboard with subscribe/unsubscribe buttons
		msg_str := ""
		msg = tgbotapi.NewMessage(user_and_msg.ChatID, "")

		if user_and_msg.Text == "enable" {
			pql.SetEnabled(db, user_and_msg.UserID, true)
			pql.GetEnabled(db, user_and_msg.UserID)
			msg_str += "You have enabled subscription notifications!\n"
		} else if user_and_msg.Text == "disable" {
			pql.SetEnabled(db, user_and_msg.UserID, false)
			pql.GetEnabled(db, user_and_msg.UserID)
			msg_str += "You have disabled subscription notifications.\n"
		} else if user_and_msg.Text == "subscriptions" {
			slice_anime_id := pql.GetSliceAnimeId(db, user_and_msg.UserID)
			// if len(slice_anime_id) == 0 {
			// slice_anime_id = append(slice_anime_id, 5081)
			// }
			if len(slice_anime_id) == 0 {
				msg_str += "You are not subscribed to any anime notifications.\n"
			} else {
				msg_str += "You are subscribed to notificatins about following titles:\n"
			}
			for _, id := range slice_anime_id {
				anime := search_anime.SearchAnimeById(id)
				msg_str += "1. "
				msg_str += anime.Data.Animes[0].English
				msg_str += "\n"
			}
		}
		msg_str += "Please choose one of the options:\n"

		msg.Text = msg_str

		// Inline keyboard for subscription options
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Enbale\nnotifications", "enable"),
				tgbotapi.NewInlineKeyboardButtonData("Disable\nnotifications", "disable"),
				tgbotapi.NewInlineKeyboardButtonData("Show\nsubscriptions", "subscriptions"),
			),
		)
		msg.ReplyMarkup = keyboard

		// Send the response message
		bot.Send(msg)
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

	// Main loop: process incoming updates
	for update := range updates {
		handleUpdate(bot, update, db)
	}
}
