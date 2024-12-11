package telegram_bot

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
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
	HandleUpdateModeRemove
)

// String method for Color type to print meaningful names
func (c HandleUpdateMode) String() string {
	return [...]string{"Basic", "Search"}[c]
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var usersMapHandleUpdMode = make(map[int64]HandleUpdateMode)

var usersMapLastAnimeIDList = make(map[int64][]int64)
var usersMapLastAnimeNameList = make(map[int64][]string)
var usersMapLastAnimeLastEpisodeList = make(map[int64][]int)

func RemoveAnimeIdAndLastEpisode(ctx context.Context, db *sql.DB, userID int64, animeID int, subscriptionNum int, msg_str string) string {
	animeResp, _ := search_anime.SearchAnimeById(ctx, int64(animeID))
	animeName := animeResp.Data.Animes[0].English

	msg_str += fmt.Sprintf("Removing anime %d. %s\n", subscriptionNum, animeName)
	pql.RemoveAnimeIdAndLastEpisode(db, userID, animeID)
	return msg_str
}

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
			tgbotapi.NewInlineKeyboardButtonData("Remove subscriptions", "remove"),
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

func checkRangeFormat(s string) (bool, int, int) {
	// Define the regex pattern for the "a-b" format
	re := regexp.MustCompile(`^(\d+)-(\d+)$`)
	matches := re.FindStringSubmatch(s)

	// If no match found, return false
	if matches == nil {
		return false, 0, 0
	}

	// Parse the integers
	a, errA := strconv.Atoi(matches[1])
	b, errB := strconv.Atoi(matches[2])

	// Check for any errors during conversion
	if errA != nil || errB != nil {
		return false, 0, 0
	}

	// Ensure that a <= b
	if a >= b {
		return false, 0, 0
	}

	// Return true if the format is valid
	return true, a, b
}

// checkCommaSeparatedIntegers checks if the string consists of comma-separated integers,
// removes duplicates, and sorts the output slice.
func checkCommaSeparatedIntegers(s string) (bool, []int) {
	// Define the regex pattern for comma-separated integers
	re := regexp.MustCompile(`^(\d+)(,\s*\d+)*$`)

	// Check if the string matches the pattern
	if !re.MatchString(s) {
		return false, nil
	}

	// Split the string by commas and parse each part into an integer
	parts := strings.Split(s, ",")
	uniqueInts := make(map[int]struct{}) // Map to store unique integers

	for _, part := range parts {
		// Trim spaces around each number
		part = strings.TrimSpace(part)
		// Convert the string part to an integer
		num, err := strconv.Atoi(part)
		if err != nil {
			// If any part is not a valid integer, return false
			return false, nil
		}
		// Add the integer to the map (duplicates are automatically removed)
		uniqueInts[num] = struct{}{}
	}

	// Convert the map keys to a slice
	var result []int
	for num := range uniqueInts {
		result = append(result, num)
	}

	// Sort the slice of integers
	sort.Ints(result)

	// Return true if all parts are valid integers, along with the sorted and deduplicated slice
	return true, result
}

// Unified function to handle both messages and inline button callbacks
func handleUpdate(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, db *sql.DB) {
	// Retrieve the logger from the context
	logger, ok := ctx.Value("logger").(*slog.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
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
	pql.SetChatID(db, user_and_msg.UserID, user_and_msg.ChatID)

	if !skip {

		if pql.UserExists(db, user_and_msg.UserID) {
			logger.Info("Found user in DB", "User ID", user_and_msg.UserID)
		} else {
			enabled := false
			// animeIDs := "{}"

			_, err = db.Exec("INSERT INTO users (id, enabled) VALUES ($1, $2)", user_and_msg.UserID, enabled)
			if err != nil {
				logger.Error("Failed INSERT", "error", err)
			}

			logger.Info("User inserted successfully", "User ID", user_and_msg.UserID)
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
					logger.Error("Error reading your list of subscritions", "error", err)

				} else {
					// if len(slice_anime_id) == 0 {
					// slice_anime_id = append(slice_anime_id, 5081)
					// }
					if len(slice_anime_id_and_last_episode) == 0 {
						msg_str += "You are not subscribed to any anime notifications.\n"
					} else {
						msg_str += "You are subscribed to notifications about following titles:\n"
						for i, id_and_last_episode := range slice_anime_id_and_last_episode {
							anime, _ := search_anime.SearchAnimeById(ctx, int64(id_and_last_episode.AnimeID))
							msg_str += strconv.Itoa(i+1) + ". "
							msg_str += anime.Data.Animes[0].English
							msg_str += "\n"
						}
					}

				}

			} else if user_and_msg.Text == "search" {
				usersMapHandleUpdMode[user_and_msg.UserID] = HandleUpdateModeSearch
				msg_str += "Enter a name of anime in english to search it.\n"
			} else if user_and_msg.Text == "remove" {
				slice_anime_id_and_last_episode, _ := pql.GetSliceAnimeIdAndLastEpisode(db, user_and_msg.UserID)
				if len(slice_anime_id_and_last_episode) == 0 {
					msg_str += "You are not subscribed to any anime notifications.\n"
				} else {
					msg_str += "You are subscribed to notifications about following titles:\n"
					var list_button_text []string
					for i, id_and_last_episode := range slice_anime_id_and_last_episode {
						anime, _ := search_anime.SearchAnimeById(ctx, int64(id_and_last_episode.AnimeID))
						msg_str += strconv.Itoa(i+1) + ". "
						msg_str += anime.Data.Animes[0].English
						msg_str += "\n"
						list_button_text = append(list_button_text, strconv.Itoa(i+1))
					}
					list_button_text = append(list_button_text, "All")
					msg_str += "\n"
					msg_str += "Choose subscriptions that you want to remove.\n"
					msg_str += "You can press a button with a number to remove a corresponding subsciption\n"
					msg_str += "You can press a button \"All\" to remove all subscriptions\n"
					msg_str += "You can send a message with a number to remove a corresponding subsciption\n"
					msg_str += "You can send a message with a range to remove corresponding subscriptions\n"
					msg_str += "For example, message 2-4 will remove subscriptions 2, 3, 4\n"
					msg_str += "You can send a message with a comma separated set of numbers to remove corresponding subscriptions\n"
					msg_str += "For example, message 1,2,5 will remove subscriptions 1, 2, 5\n"
					usersMapHandleUpdMode[user_and_msg.UserID] = HandleUpdateModeRemove

					keyboard = CreateInlineKeyboard(list_button_text, 5)
					msg.ReplyMarkup = keyboard

				}

			}
			if usersMapHandleUpdMode[user_and_msg.UserID] == HandleUpdateModeBasic {
				msg_str, keyboard, msg = GeneralMessage(msg_str, keyboard, msg)
			}

		} else if handle_update_mode == HandleUpdateModeSearch {
			animeResp, _ := search_anime.SearchAnimeByName(ctx, user_and_msg.Text)

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
		} else if usersMapHandleUpdMode[user_and_msg.UserID] == HandleUpdateModeRemove {
			userID := user_and_msg.UserID
			text := user_and_msg.Text
			list, _ := pql.GetSliceAnimeIdAndLastEpisode(db, userID)

			if text == "All" {
				for i, anime := range list {
					msg_str = RemoveAnimeIdAndLastEpisode(ctx, db, userID, anime.AnimeID, i+1, msg_str)

				}
			} else {
				v, a, b := checkRangeFormat(text)
				if v {
					if a-1 < len(list) {
						i_0 := a - 1
						i_1 := min(len(list)-1, b-1)
						for i := i_0; i <= i_1; i++ {
							msg_str = RemoveAnimeIdAndLastEpisode(ctx, db, userID, list[i].AnimeID, i+1, msg_str)
						}
					}

				} else {
					v, list_of_indices := checkCommaSeparatedIntegers(text)
					if v {
						for _, j := range list_of_indices {
							if j-1 < len(list) {
								msg_str = RemoveAnimeIdAndLastEpisode(ctx, db, userID, list[j-1].AnimeID, j, msg_str)
							} else {
								msg_str += fmt.Sprintf("Error: %d is greater than the largerst index of your subcription\n", j)

								break
							}
						}
					} else {
						val, err := strconv.Atoi(text)
						if err == nil {
							if val-1 < len(list) {
								msg_str = RemoveAnimeIdAndLastEpisode(ctx, db, userID, list[val-1].AnimeID, val, msg_str)

							} else {
								msg_str += fmt.Sprintf("Error: %d is greater than the largerst index of your subcription\n", val)
							}

						} else {
							msg_str += "Couldn't parse your message to get the necessary indices of subscriptions to remove\n"
						}
					}
				}

			}
			list, _ = pql.GetSliceAnimeIdAndLastEpisode(db, userID)

			if len(list) == 0 {
				msg_str += "You are not subscribed to any anime notifications.\n"
			} else {
				msg_str += "You are subscribed to notifications about following titles:\n"
				for i, id_and_last_episode := range list {
					anime, _ := search_anime.SearchAnimeById(ctx, int64(id_and_last_episode.AnimeID))
					msg_str += strconv.Itoa(i+1) + ". "
					msg_str += anime.Data.Animes[0].English
					msg_str += "\n"
				}
			}

			usersMapHandleUpdMode[user_and_msg.UserID] = HandleUpdateModeBasic
			msg_str, keyboard, msg = GeneralMessage(msg_str, keyboard, msg)
		}

		msg.Text = msg_str
		msg.DisableWebPagePreview = true

		// Send the response message
		bot.Send(msg)
	}
}

func SignalAnimeComplete(bot *tgbotapi.BotAPI, chatID int64, animeName string) {
	var msg_str string
	var keyboard tgbotapi.InlineKeyboardMarkup
	var msg tgbotapi.MessageConfig

	msg = tgbotapi.NewMessage(chatID, "")
	msg_str = animeName + " is Complete!\nSubscription will be removed.\n"
	msg_str, keyboard, msg = GeneralMessage(msg_str, keyboard, msg)

	msg.Text = msg_str
	bot.Send(msg)
}

func SignalAnimeNewEpisodes(bot *tgbotapi.BotAPI, chatID int64, animeName string, newEpisode int) {
	var msg_str string
	var keyboard tgbotapi.InlineKeyboardMarkup
	var msg tgbotapi.MessageConfig

	msg = tgbotapi.NewMessage(chatID, "")
	msg_str = animeName + " episode " + strconv.Itoa(newEpisode) + " is released!\n"
	msg_str, keyboard, msg = GeneralMessage(msg_str, keyboard, msg)

	msg.Text = msg_str
	bot.Send(msg)
}

func TestSignalUpdate(bot *tgbotapi.BotAPI, chatID int64) {
	animeName := "TestAnimeName"
	newEpisode := 777
	SignalAnimeComplete(bot, chatID, animeName)
	SignalAnimeNewEpisodes(bot, chatID, animeName, newEpisode)
}

func processUsers(ctx context.Context, db *sql.DB, bot *tgbotapi.BotAPI) {
	// Query all users from the users table
	logger, ok := ctx.Value("logger").(*slog.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	rows, err := db.Query("SELECT id, enabled FROM users")
	if err != nil {
		logger.Error("Failed to query users", "error", err)
	}
	defer rows.Close()

	// Process each user
	for rows.Next() {
		var userID int64
		var enabled bool
		if err := rows.Scan(&userID, &enabled); err != nil {
			logger.Error("Failed to scan row", "error", err)
		}
		if !enabled {
			return
		}

		// Example processing: Print the user ID and their enabled status

		logger.Info("Processing User ", "User ID", userID)

		list, err := pql.GetSliceAnimeIdAndLastEpisode(db, userID)
		if err != nil {
			logger.Error("Error reading subscription for user", "User ID", userID, "error", err)
		}

		if len(list) == 0 {
			logger.Info("Used is not subscribed to any anime notifications.\n", "User ID", userID)
		} else {
			for _, id_and_last_episode := range list {

				animeID := id_and_last_episode.AnimeID

				anime, _ := search_anime.SearchAnimeById(ctx, int64(animeID))
				animeName := anime.Data.Animes[0].English
				storedAnimeLastEpisode := id_and_last_episode.LastEpisode
				actualAnimeLastEpisode := anime.Data.Animes[0].EpisodesAired
				animeStatus := anime.Data.Animes[0].Status
				chatID := pql.GetChatID(db, userID)
				if animeStatus == "released" {
					SignalAnimeComplete(bot, chatID, animeName)
					pql.RemoveAnimeIdAndLastEpisode(db, userID, animeID)
				} else if actualAnimeLastEpisode > storedAnimeLastEpisode {
					SignalAnimeNewEpisodes(bot, chatID, animeName, actualAnimeLastEpisode)
					pql.UpdateAnimeIdAndLastEpisode(db, userID, animeID, actualAnimeLastEpisode)
				} else {
					logger.Info("Nothing new for User", "User ID", userID)
				}
				// TestSignalUpdate(bot, chatID)
			}
		}

		// You can add your custom processing logic here
	}

	// Check for errors in the row iteration
	if err := rows.Err(); err != nil {
		logger.Error("Error reading rows", "error", err)
	}
}

func StartBotAndHandleUpdates(ctx context.Context, db *sql.DB) {
	// Get the logger from the context, or use a default logger if not available
	logger, ok := ctx.Value("logger").(*slog.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	// Get the Telegram bot token from an environment variable
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		logger.Error("TELEGRAM_BOT_TOKEN is not set")
		panic("TELEGRAM_BOT_TOKEN is not set") // Panic instead of log.Fatal
	}

	// Initialize the bot
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logger.Error("Failed to initialize bot", "error", err)
		panic("Failed to initialize bot") // Panic instead of log.Fatal
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
		logger.Error("Failed to get updates", "error", err)
		panic("Failed to get updates") // Panic instead of log.Fatal
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
