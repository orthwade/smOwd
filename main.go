package main

import (
	"context"

	// "log"
	"log/slog"
	"os"
	"smOwd/logs"

	"database/sql"
	"smOwd/animes"
	"smOwd/pql"
	"smOwd/subscriptions"

	"smOwd/tgbot"
	"smOwd/users"
	"time"

	"github.com/joho/godotenv"
)

func LoadEnv(ctx context.Context) {
	logger := logs.DefaultFromCtx(ctx)
	if err := godotenv.Load(); err != nil {
		logger.Fatal("Error loading .env file", "error", err)
	} else {
		logger.Info("Load .env file successfull")
	}
}

func CreateTableIfNotExistAndPrintInfo(ctx context.Context,
	db *sql.DB, tableName string,
	createFunc func(context.Context, *sql.DB) error) {

	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}
	tableExists, _ := pql.CheckTable(ctx, db, tableName)

	if tableExists {
		logger.Info("Table " + tableName + " already created")
	} else {
		logger.Warn("Table " + tableName +
			" isn't created yet. Attempting create")
		createFunc(ctx, db)
	}

	pql.PrintTableColumnsNamesAndTypes(ctx, db, tableName)
}

func simulateFatal(ctx context.Context) {
	time.Sleep(2 * time.Second)

	logger := logs.DefaultFromCtx(ctx)

	logger.Fatal("Sim Fatal")
}

func testGracefulShutdown(cancel context.CancelFunc) {
	time.Sleep(2 * time.Second)
	cancel()
}

func main() {
	// return
	// Initialize logger
	logger := logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	// Create context with logger
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = context.WithValue(ctx, "logger", logger)

	LoadEnv(ctx)

	postgresDb := pql.ConnectToDatabasePostgres(ctx)

	db := pql.ConnectToDatabaseSubscriptions(ctx, postgresDb)
	defer db.Close()

	postgresDb.Close()

	CreateTableIfNotExistAndPrintInfo(ctx, db, "users", users.CreateTable)
	CreateTableIfNotExistAndPrintInfo(ctx, db, "animes", animes.CreateTable)
	CreateTableIfNotExistAndPrintInfo(ctx, db, "subscriptions",
		subscriptions.CreateTable)

	// animes_, _ := animes.SearchAnimeByName(ctx, "frieren")

	tgbot.StartBotAndHandleUpdates(ctx, cancel, db)

}
