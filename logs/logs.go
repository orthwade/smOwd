// logs/logs.go
package logs

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

type Logger struct {
	*slog.Logger
}

func New(logs *slog.Logger) *Logger {
	return &Logger{Logger: logs}
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	// Log the fatal message
	l.Error(msg, args...)

	// Trigger a SIGTERM signal to terminate the program
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGTERM)

	// Send the SIGTERM signal
	sigTerm <- syscall.SIGTERM

	// Optionally, wait for the signal handler to run (for graceful shutdown)
	<-sigTerm
	os.Exit(1)
}

func DefaultFromCtx(ctx context.Context) *Logger {
	logger, ok := ctx.Value("logger").(*Logger)
	if !ok {
		logger = New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	return logger
}
