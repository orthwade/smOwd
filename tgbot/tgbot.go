package tgbot

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"smOwd/logs"

	// "strconv"
	"syscall"
	"time"

	// "smOwd/pql"
	"smOwd/users"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

type handleUpdateMode int

const (
	handleUpdateModeInit handleUpdateMode = iota
	handleUpdateModeBasic
	handleUpdateModeSearch
	handleUpdateModeSubscribe
	handleUpdateModeRemove
)

func (c handleUpdateMode) String() string {
	return [...]string{"Basic", "Search"}[c]
}

type sessionData struct {
	handleUpdateMode
}

type userHandle struct {
	sessionDataField sessionData
}

var mapIdUserHandle = make(map[int]*userHandle)

func checkAndAddUserToMap(ctx context.Context, userID int) {
	logger := logs.DefaultFromCtx(ctx)

	_, exists := mapIdUserHandle[userID]

	if !exists {
		logger.Info("Adding user handle to map", "ID", userID)
		sessionDataObj := sessionData{handleUpdateMode: handleUpdateModeInit}
		userHandleObj := userHandle{sessionDataField: sessionDataObj}
		mapIdUserHandle[userID] = &userHandleObj
	}
}

func createInlineKeyboard(listText []string,
	maxCols int) tgbotapi.InlineKeyboardMarkup {
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

func generalMessage(chatID int, notificationsEnabled bool) *tgbotapi.MessageConfig {
	msgStr := "Please choose one of the options:\n"
	msg := tgbotapi.NewMessage(int64(chatID), msgStr)

	var keyboard tgbotapi.InlineKeyboardMarkup

	if notificationsEnabled {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Disable\nnotifications", "disable"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Show\nsubscriptions", "subscriptions"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Remove subscriptions", "remove"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Search anime by name", "search"),
			),
		)
	} else {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Enable\nnotifications", "enable"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Show\nsubscriptions", "subscriptions"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Remove subscriptions", "remove"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Search anime by name", "search"),
			),
		)
	}

	msg.ReplyMarkup = keyboard

	return &msg
}

// Unified function to handle both messages and inline button callbacks
func handleUpdate(ctx context.Context, bot *tgbotapi.BotAPI,
	update tgbotapi.Update, db *sql.DB) {
	// Retrieve the logger from the context
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	var user *users.User
	var telegramID int
	var chatID int
	var tgbotUser *tgbotapi.User

	var messageText string

	skip := true
	if update.Message != nil {
		tgbotUser = update.Message.From
		telegramID = update.Message.From.ID
		chatID = int(update.Message.Chat.ID)
		messageText = misc.RemoveFirstCharIfPresent(update.Message.Text, '/')
		skip = false

	} else if update.CallbackQuery != nil { // Handle inline button callback queries
		tgbotUser = update.Message.From
		telegramID = update.CallbackQuery.From.ID
		chatID = int(update.CallbackQuery.Message.Chat.ID)
		messageText = update.CallbackQuery.Data
		skip = false
		defer bot.AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, "Done"))
	}
	if !skip {
		user = users.FindByTelegramID(ctx, db, telegramID)

		if user == nil {
			logger.Info("New user", "tg_name", tgbotUser.UserName)
			user = &users.User{
				TelegramID:   tgbotUser.ID,
				ChatID:       chatID,
				FirstName:    tgbotUser.FirstName,
				LastName:     tgbotUser.LastName,
				UserName:     tgbotUser.UserName,
				LanguageCode: tgbotUser.LanguageCode,
				IsBot:        tgbotUser.IsBot,
				Enabled:      true, // Default to enabled, or adjust as needed
			}
			users.Add(ctx, db, user)

			user = users.FindByTelegramID(ctx, db, telegramID)

			if user == nil {
				logger.Fatal("Fatal error processing user", "Telegram ID", telegramID)
			}

		} else {
			logger.Info("Found user in db", "tg_name", tgbotUser.UserName)
		}

		checkAndAddUserToMap(ctx, user.ID)
	}

	userHandlePtr := mapIdUserHandle[user.ID]

	session := &userHandlePtr.sessionDataField
	updateMode := &session.handleUpdateMode

	if *updateMode == handleUpdateModeInit {
		logger.Info("Update handle mode Initial", "tgname", user.UserName)

		msg := generalMessage(chatID, user.Enabled)
		bot.Send(msg)

		*updateMode = handleUpdateModeBasic
	} else if *updateMode == handleUpdateModeBasic {
		logger.Info("Update handle mode Basic", "tgname", user.UserName)
		
		if messageText == "Enable"
	}
}

func processUsers(ctx context.Context, db *sql.DB, bot *tgbotapi.BotAPI) {

}

func StartBotAndHandleUpdates(ctx context.Context, cancel context.CancelFunc,
	db *sql.DB) {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		logger.Fatal("TELEGRAM_BOT_TOKEN is not set")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logger.Fatal("Failed to initialize bot", "error", err)
	}

	// Set bot to debug mode (optional)
	bot.Debug = true
	logger.Info("Authorized on account", "UserName", bot.Self.UserName)

	// Configure the update channel (long polling)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Get updates (messages and callback queries) from Telegram
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		logger.Fatal("Failed to get updates", "error", err)
	}
	// Create a channel to synchronize processUsers with handleUpdate
	processUsersChan := make(chan bool)

	// Set up a goroutine to listen for OS signals and trigger shutdown
	go func() {
		// Channel for receiving OS termination signals (e.g., CTRL+C or kill command)
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		<-signalChan // Block until a signal is received
		logger.Info("Received shutdown signal, shutting down gracefully...")
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
				logger.Info("Stopping user processing due to shutdown signal.")
				return
			}
		}
	}()

	// Main loop: process incoming updates and handle periodic user processing
	for {
		select {
		case update := <-updates:
			// Handle incoming updates (messages and callback queries)
			handleUpdate(ctx, bot, update, db)
		case <-processUsersChan:
			// This block is triggered every 1 second to process users
			processUsers(ctx, db, bot)
		case <-ctx.Done():
			// Graceful shutdown of the main loop
			logger.Info("Shutting down the bot.")
			return
		}
	}
}
