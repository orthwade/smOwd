package telegram_bot

import (
	"fmt"
	"log"
	"os"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

var subscribedUsers = make(map[int]bool)

// Handle incoming text messages like "/start", "/subscribe", and "/unsubscribe"
func handleUpdates(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.Message == nil { // Ignore non-message updates
		return
	}

	// userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	// If the user sends "/start", show the inline keyboard with subscribe/unsubscribe buttons
	if text == "/start" {
		msg := tgbotapi.NewMessage(chatID, "Welcome! Please choose to subscribe or unsubscribe.")

		// Inline keyboard for subscription options
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Subscribe", "subscribe"),
				tgbotapi.NewInlineKeyboardButtonData("Unsubscribe", "unsubscribe"),
			),
		)
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
	}
}

// Handle callback queries (button clicks)
func handleCallbackQuery(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.CallbackQuery == nil { // Ignore non-callback queries
		return
	}

	callbackData := update.CallbackQuery.Data
	userID := update.CallbackQuery.From.ID
	chatID := update.CallbackQuery.Message.Chat.ID

	var msg tgbotapi.MessageConfig

	// Based on the button clicked, subscribe or unsubscribe the user
	switch callbackData {
	case "subscribe":
		subscribedUsers[userID] = true
		msg = tgbotapi.NewMessage(chatID, "You are now subscribed!")
	case "unsubscribe":
		delete(subscribedUsers, userID)
		msg = tgbotapi.NewMessage(chatID, "You have unsubscribed.")
	}

	// Acknowledge the callback to remove the "loading" spinner
	bot.AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, "Done"))

	// Send the response message
	bot.Send(msg)
}

func StartBotAndHandleUpdates() {
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
		// Handle callback query (button clicks)
		if update.CallbackQuery != nil {
			handleCallbackQuery(bot, update)
		}

		// Handle regular message updates (like /start, /subscribe)
		if update.Message != nil {
			handleUpdates(bot, update)
		}
	}
}
