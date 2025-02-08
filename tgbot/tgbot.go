package tgbot

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"smOwd/logs"

	"strconv"
	"syscall"
	"time"

	"fmt"
	"smOwd/animes"
	"smOwd/misc"
	"smOwd/subscriptions"
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
	handleUpdateModeField handleUpdateMode
	sliceAnime            []animes.Anime
	lastTgMsg             tgbotapi.MessageConfig
	sliceSubscriptions    []subscriptions.Subscription
	test                  bool
}

type userHandle struct {
	sessionDataField sessionData
}

var mapIdUserHandle = make(map[int]*userHandle)

func clearSessionData(userID int) {
	userHandlePtr := mapIdUserHandle[userID]
	userHandlePtr.sessionDataField.lastTgMsg = tgbotapi.MessageConfig{}
	userHandlePtr.sessionDataField.sliceAnime = []animes.Anime{}
	userHandlePtr.sessionDataField.sliceSubscriptions = []subscriptions.Subscription{}
}

func checkAndAddUserToMap(ctx context.Context, userID int) {
	logger := logs.DefaultFromCtx(ctx)

	_, exists := mapIdUserHandle[userID]

	if !exists {
		logger.Info("Adding user handle to map", "ID", userID)
		sessionDataObj := sessionData{handleUpdateModeField: handleUpdateModeInit,
			sliceAnime: nil}
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
				tgbotapi.NewInlineKeyboardButtonData("Disable notifications", "disable"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Show subscriptions", "subscriptions"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Remove subscriptions", "remove"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Search anime by name", "search"),
			),
			// tgbotapi.NewInlineKeyboardRow(
			// 	tgbotapi.NewInlineKeyboardButtonData("Test", "test"),
			// ),
		)
	} else {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Enable notifications", "enable"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Show subscriptions", "subscriptions"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Remove subscriptions", "remove"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Search anime by name", "search"),
			),
			// tgbotapi.NewInlineKeyboardRow(
			// 	tgbotapi.NewInlineKeyboardButtonData("Test", "test"),
			// ),
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
	var chatID int
	var tgbotUser *tgbotapi.User

	var messageText string

	skip := true
	if update.Message != nil {

		tgbotUser = update.Message.From

		chatID = int(update.Message.Chat.ID)
		messageText = misc.RemoveFirstCharIfPresent(update.Message.Text, '/')

		skip = false

	} else if update.CallbackQuery != nil { // Handle inline button callback queries
		tgbotUser = update.CallbackQuery.Message.From

		chatID = int(update.CallbackQuery.Message.Chat.ID)
		messageText = update.CallbackQuery.Data
		skip = false
		defer bot.AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, "Done"))
	}
	if !skip {
		user = users.FindByChatID(ctx, db, chatID)

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
			user_id, err := users.Add(ctx, db, user)

			if err != nil {
				logger.Fatal("Error adding user to db",
					"Telegram ID", user.TelegramID,
					"error", err)
			}

			user.ID = user_id
		} else {
			logger.Info("Found user in db", "tg_name", tgbotUser.UserName)
		}

		checkAndAddUserToMap(ctx, user.ID)
	}

	userHandlePtr := mapIdUserHandle[user.ID]

	session := &userHandlePtr.sessionDataField
	updateMode := &session.handleUpdateModeField

	if *updateMode == handleUpdateModeInit {
		logger.Info("Update handle mode Initial", "tgname", user.UserName)

		msg := generalMessage(chatID, user.Enabled)
		bot.Send(tgbotapi.NewMessage(int64(chatID), "Started!"))

		bot.Send(msg)

		*updateMode = handleUpdateModeBasic

	} else if *updateMode == handleUpdateModeBasic {
		clearSessionData(user.ID)

		logger.Info("Update handle mode Basic", "tgname", user.UserName)

		if messageText == "enable" {
			err := users.Enable(ctx, db, user.ID)

			if err == nil {
				logger.Info("Enabled notifications",
					"Telegram username", user.UserName)

				bot.Send(tgbotapi.NewMessage(int64(chatID), "Enabled notifications"))
				bot.Send(generalMessage(chatID, true))
			} else {
				logger.Error("Failed to enable notifications",
					"Telegram username", user.UserName,
					"error", err)

				bot.Send(generalMessage(chatID, user.Enabled))
			}

		} else if messageText == "disable" {
			err := users.Disable(ctx, db, user.ID)

			if err == nil {
				logger.Info("Disabled notifications",
					"Telegram username", user.UserName)

				bot.Send(tgbotapi.NewMessage(int64(chatID), "Disabled notifications"))
				bot.Send(generalMessage(chatID, false))

			} else {
				logger.Error("Failed to disable notifications",
					"Telegram username", user.UserName,
					"error", err)

				bot.Send(generalMessage(chatID, user.Enabled))
			}
		} else if messageText == "search" {
			bot.Send(tgbotapi.NewMessage(int64(chatID), "Enter the name of the anime"))
			*updateMode = handleUpdateModeSearch
		} else if messageText == "subscriptions" {
			sliceSubscriptions := subscriptions.FindAll(ctx, db, user.TelegramID)

			if len(sliceSubscriptions) == 0 {
				bot.Send(tgbotapi.NewMessage(int64(user.ChatID),
					"You have no subscriptions"))
			} else {
				outputMsgText := "You are subscribed to these animes:\n\n"

				var shikiIDs []string

				for _, s := range sliceSubscriptions {
					shikiIDs = append(shikiIDs, s.ShikiID)
				}

				sliceAnime, err := animes.SearchAnimeByShikiIDs(ctx, shikiIDs)

				if err != nil {
					logger.Error("Error searching animes by ids",
						"IDs", shikiIDs,
						"error", err)
				} else {
					for i, a := range sliceAnime {
						line := strconv.Itoa(i+1) + ". " + a.English + " / " + a.URL + "\n"
						outputMsgText += line
						outputMsgText += fmt.Sprintf("Last episode aired: %d\n", a.EpisodesAired)

						var lastNotification int

						for _, s := range sliceSubscriptions {
							if user.TelegramID == s.TelegramID && s.ShikiID == a.ShikiID {
								lastNotification = s.LastEpisodeNotified
								break
							}
						}

						outputMsgText += fmt.Sprintf("Last episode notified of: %d\n\n", lastNotification)
					}
					outputMsg := tgbotapi.NewMessage(int64(chatID), outputMsgText)
					outputMsg.DisableWebPagePreview = true

					bot.Send(outputMsg)
				}
			}
			*updateMode = handleUpdateModeBasic
			bot.Send(generalMessage(chatID, user.Enabled))

		} else if messageText == "remove" {
			outputMsgText := "Choose an anime to unscubscribe:\n\n"

			var shikiIDs []string

			sliceSubscriptions := subscriptions.FindAll(ctx, db, user.TelegramID)

			if sliceSubscriptions == nil {
				logger.Error("Error getting subscriptions from DB",
					"Telegram ID", user.TelegramID)

				bot.Send(generalMessage(chatID, user.Enabled))
			} else if len(sliceSubscriptions) == 0 {
				bot.Send(tgbotapi.NewMessage(int64(user.ChatID),
					"You have no subscriptions"))
				bot.Send(generalMessage(chatID, user.Enabled))

			} else {
				var err error
				for _, s := range sliceSubscriptions {
					shikiIDs = append(shikiIDs, s.ShikiID)
				}

				session.sliceSubscriptions = sliceSubscriptions

				sliceAnime, err := animes.SearchAnimeByShikiIDs(ctx, shikiIDs)

				col := 0

				var buttons []tgbotapi.InlineKeyboardButton
				var keyboard [][]tgbotapi.InlineKeyboardButton

				for i, s := range session.sliceSubscriptions {
					found := false
					for _, a := range sliceAnime {
						if a.ShikiID == s.ShikiID {
							found = true
							session.sliceSubscriptions[i].Anime = &a
						}
					}
					if !found {
						logger.Fatal("Error: subscription anime mismatch")
					}
				}
				session.sliceAnime = []animes.Anime{}

				for _, s := range session.sliceSubscriptions {
					session.sliceAnime = append(session.sliceAnime, *s.Anime)
				}

				if err != nil {
					logger.Error("Error searching animes by ids",
						"IDs", shikiIDs,
						"error", err)
				} else {
					for i, a := range session.sliceAnime {
						line := strconv.Itoa(i+1) + ". " + a.English + " / " + a.URL + "\n"
						outputMsgText += line

						buttons = append(buttons,
							tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(i+1), strconv.Itoa(i)))

						col++
						if col > 4 {
							col = 0
							keyboard = append(keyboard, buttons)
							buttons = []tgbotapi.InlineKeyboardButton{}
						}
					}
					if len(buttons) > 0 {
						keyboard = append(keyboard, buttons)
					}

					buttons = []tgbotapi.InlineKeyboardButton{}
					buttons = append(buttons,
						tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel"))

					keyboard = append(keyboard, buttons)

					outputMsg := tgbotapi.NewMessage(int64(chatID), outputMsgText)
					outputMsg.DisableWebPagePreview = true
					outputMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

					bot.Send(outputMsg)
					*updateMode = handleUpdateModeRemove
					session.lastTgMsg = outputMsg
				}
			}

		}
	} else if *updateMode == handleUpdateModeSearch {
		var err error
		session.sliceAnime, err = animes.SearchAnimeByName(ctx, messageText)

		if err != nil {
			logger.Error("Error searching for anime",
				"Anime name", messageText,
				"error", err)

			*updateMode = handleUpdateModeBasic
		} else if len(session.sliceAnime) == 0 {

			logger.Warn("No animes found",
				"Anime name", messageText)
			tgMsg := tgbotapi.NewMessage(int64(chatID), "No animes found")

			bot.Send(tgMsg)

			bot.Send(generalMessage(chatID, user.Enabled))

			*updateMode = handleUpdateModeBasic

		} else {
			var keyboard [][]tgbotapi.InlineKeyboardButton
			var buttons []tgbotapi.InlineKeyboardButton

			col := 0

			var tgMsgText string
			var tgMsg tgbotapi.MessageConfig

			multipleMessages := false

			for i, anime := range session.sliceAnime {
				animeStr := strconv.Itoa(i+1) + ". " + anime.English + " / " + anime.URL + "\n"

				if anime.Status == "released" {
					animeStr += " / RELEASED!\n\n"
				} else {
					animeStr += "\n"

					buttons = append(buttons,
						tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(i+1), strconv.Itoa(i)))

					col++
					if col > 4 {
						col = 0
						keyboard = append(keyboard, buttons)
						buttons = []tgbotapi.InlineKeyboardButton{}
					}
				}

				tgMsgText += animeStr

				if len(tgMsgText) > 3000 {
					if len(buttons) > 0 {
						keyboard = append(keyboard, buttons)
					}
					buttons = []tgbotapi.InlineKeyboardButton{}
					buttons = append(buttons,
						tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel"))

					keyboard = append(keyboard, buttons)

					tgMsg = tgbotapi.NewMessage(int64(chatID), tgMsgText)
					tgMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
					tgMsg.DisableWebPagePreview = true
					bot.Send(tgMsg)

					keyboard = [][]tgbotapi.InlineKeyboardButton{}
					buttons = []tgbotapi.InlineKeyboardButton{}
					tgMsgText = ""

					if i < len(session.sliceAnime) {
						multipleMessages = true
					} else {
						multipleMessages = false
					}
				}
			}

			if multipleMessages {
				if len(buttons) > 0 {
					keyboard = append(keyboard, buttons)
				}
				buttons = []tgbotapi.InlineKeyboardButton{}
				buttons = append(buttons,
					tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel"))

				keyboard = append(keyboard, buttons)

				tgMsg = tgbotapi.NewMessage(int64(chatID), tgMsgText)
				tgMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
				tgMsg.DisableWebPagePreview = true
				bot.Send(tgMsg)
			}

			*updateMode = handleUpdateModeSubscribe
			session.lastTgMsg = tgMsg
		}
	} else if *updateMode == handleUpdateModeSubscribe {
		if update.CallbackQuery == nil {
			logger.Warn("No button pressed")

			bot.Send(tgbotapi.NewMessage(int64(user.ChatID),
				"Don't text, press a button"))

			bot.Send(session.lastTgMsg)
		} else if messageText == "cancel" {
			bot.Send(generalMessage(chatID, user.Enabled))
			*updateMode = handleUpdateModeBasic
		} else {
			i, _ := strconv.Atoi(messageText)

			anime := &session.sliceAnime[i]

			*updateMode = handleUpdateModeBasic

			logger.Info("Selected anime",
				"Anime name", anime.English)

			subscription := subscriptions.Find(ctx, db,
				user.TelegramID, anime.ShikiID)

			if subscription != nil {
				logger.Warn("Subscription already exists",
					"Telegram ID", user.TelegramID,
					"Shiki ID", anime.ShikiID)

				bot.Send(tgbotapi.NewMessage(int64(chatID),
					fmt.Sprintf("You are already subscribed to %s", anime.English)))

			} else {
				subscription = &subscriptions.Subscription{
					ID:                  -1,
					TelegramID:          user.TelegramID,
					ShikiID:             anime.ShikiID,
					LastEpisodeNotified: anime.EpisodesAired,
				}

				logger.Info("Adding subscription to db",
					"Telegram ID", user.TelegramID,
					"Anime name", anime.English)

				subscription_id, err := subscriptions.Add(ctx, db, *subscription)

				if err != nil {
					logger.Fatal("Error adding subscription to db",
						"Telegram ID", user.TelegramID,
						"Anime name", anime.English,
						"error", err)
				}

				subscription.ID = subscription_id

				logger.Info("Added subscriptions to db",
					"Telegram ID", user.TelegramID,
					"Anime name", anime.English,
					"Subscription ID", subscription.ID)

				bot.Send(tgbotapi.NewMessage(int64(chatID),
					fmt.Sprintf("You are now subscribed to %s", anime.English)))
			}

			*updateMode = handleUpdateModeBasic

			bot.Send(generalMessage(chatID, user.Enabled))
		}
	} else if *updateMode == handleUpdateModeRemove {
		if update.CallbackQuery == nil {
			logger.Warn("No button pressed")

			bot.Send(tgbotapi.NewMessage(int64(user.ChatID),
				"Don't text, press a button"))

			bot.Send(session.lastTgMsg)
		} else if messageText == "cancel" {
			*updateMode = handleUpdateModeBasic
			bot.Send(generalMessage(chatID, user.Enabled))
		} else {
			i, _ := strconv.Atoi(messageText)

			s := session.sliceSubscriptions[i]

			err := subscriptions.Remove(ctx, db, s.ID)

			if err != nil {
				logger.Error("Error removing subscription",
					"Telegram ID", s.TelegramID,
					"Shiki ID", s.ShikiID,
					"error", err)

			} else {
				logger.Info("Removed subscription",
					"Telegram ID", s.TelegramID,
					"Shiki ID", s.ShikiID)

				outputMsg := tgbotapi.NewMessage(int64(chatID),
					fmt.Sprintf("You are unsubscribed from %s", s.Anime.English))

				bot.Send(outputMsg)
			}

			*updateMode = handleUpdateModeBasic
			bot.Send(generalMessage(chatID, user.Enabled))
		}

	}
}

var testReleased = false
var testNewEpisode = false

func processUsers(ctx context.Context, db *sql.DB, bot *tgbotapi.BotAPI) {
	logger := logs.DefaultFromCtx(ctx)

	sliceSubscriptions := subscriptions.SelectAll(ctx, db)

	if len(sliceSubscriptions) == 0 {
		logger.Info("No subscrtiptions in db")
	} else {
		for _, s := range sliceSubscriptions {
			user := users.FindByTelegramID(ctx, db, s.TelegramID)

			if !user.Enabled {
				continue
			}

			// userHandle, ok := mapIdUserHandle[user.ID]
			// session := &userHandle.sessionDataField
			// updateMode := &session.handleUpdateModeField

			sliceAnime, err := animes.SearchAnimeByShikiIDs(ctx, []string{s.ShikiID})

			var a animes.Anime

			if err != nil {
				logger.Error("Error searching anime by shiki ID",
					"Shiki ID", s.ShikiID,
					"error", err)

			} else if sliceAnime == nil {
				logger.Error("Error: no anime found",
					"Shiki ID", s.ShikiID)
			} else {
				a = sliceAnime[0]
				logger.Info("Found anime", "Anime name", a.English)

				chatID := user.ChatID

				if a.Status == "released" {
					logger.Info("Anime status RELEASED!", "Anime name", a.English)
					outputMsg := tgbotapi.NewMessage(int64(chatID),
						fmt.Sprintf("%s\n%s \nStatus Released!"+
							"\nYou are no longer subscribed to this anime",
							a.English, a.URL))

					outputMsg.DisableWebPagePreview = true

					err = subscriptions.Remove(ctx, db, s.ID)

					if err != nil {
						logger.Error("Error removing subscription",
							"Telegram ID", s.TelegramID,
							"Shiki ID", s.ShikiID)
					} else {
						bot.Send(outputMsg)
					}
				} else if a.EpisodesAired > s.LastEpisodeNotified {
					logger.Info("New Episode!",
						"Anime name", a.English,
						"Episode", a.EpisodesAired)

					outputMsg := tgbotapi.NewMessage(int64(chatID),
						fmt.Sprintf("%s\n%s \nNew Episode %d!",
							a.English, a.URL, a.EpisodesAired))

					outputMsg.DisableWebPagePreview = true

					bot.Send(outputMsg)

					subscriptions.SetLastEpisode(ctx, db, s.ID, a.EpisodesAired)
				} else if testReleased {
					logger.Info("Anime status RELEASED! ----TEST----", "Anime name", a.English)
					outputMsg := tgbotapi.NewMessage(int64(chatID),
						fmt.Sprintf("%s\n%s \nStatus Released!"+
							"\nYou are no longer subscribed to this anime",
							a.English, a.URL))
					outputMsg.DisableWebPagePreview = true

					// err = subscriptions.Remove(ctx, db, s.ID)

					if err != nil {
						logger.Error("Error removing subscription ----TEST----",
							"Telegram ID", s.TelegramID,
							"Shiki ID", s.ShikiID)
					} else {
						bot.Send(outputMsg)
					}

					ss := subscriptions.FindAll(ctx, db, user.TelegramID)

					for _, s := range ss {
						logger.Info("Subscrtiption",
							"Telegram ID", s.TelegramID,
							"Shiki ID", s.ShikiID)
					}

					testReleased = false

				} else if testNewEpisode {
					logger.Info("New Episode! ----TEST----",
						"Anime name", a.English,
						"Episode", a.EpisodesAired)

					outputMsg := tgbotapi.NewMessage(int64(chatID),
						fmt.Sprintf("%s\n%s \nNew Episode %d!",
							a.English, a.URL, a.EpisodesAired))
					outputMsg.DisableWebPagePreview = true

					bot.Send(outputMsg)

					// subscriptions.SetLastEpisode(ctx, db, s.ID, a.EpisodesAired)

					ss := subscriptions.FindAll(ctx, db, user.TelegramID)

					for _, s := range ss {
						logger.Info("Subscrtiption",
							"Telegram ID", s.TelegramID,
							"Shiki ID", s.ShikiID)
					}

					testNewEpisode = false
				}

			}
		}
	}

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
	u.Timeout = 160

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
		ticker := time.NewTicker(15 * time.Minute)
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
