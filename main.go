package main

import (
	"context"

	// "log"
	"log/slog"
	"os"
	"smOwd/logs"

	"smOwd/pql"
	// "smOwd/telegram_bot"

	"github.com/joho/godotenv"
)

func LoadEnv(ctx context.Context) {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}
	if err := godotenv.Load(); err != nil {
		logger.Fatal("Error loading .env file", "error", err)
	} else {
		logger.Info("Load .env file successfull")
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	ctx := context.WithValue(context.Background(), "logger", logger)

	LoadEnv(ctx)

	db := pql.ConnectToDatabaseSubscriptions(ctx)
	defer db.Close()

	// db := pql.ConnectToDatabaseSubscriptions(ctx)
	// defer db.Close()

	// pql.PrintTableColumnsNamesAndTypes(ctx, db, "users")
	// telegram_bot.StartBotAndHandleUpdates(ctx, db)
}
