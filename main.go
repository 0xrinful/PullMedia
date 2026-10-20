package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/lmittmann/tint"
)

var downloadSemaphore = make(chan struct{}, 5)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	logger := slog.New(
		tint.NewTextHandler(os.Stdout, &tint.Options{TimeFormat: "2006/01/02 15:04:05"}),
	)
	slog.SetDefault(logger)

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("missing TELEGRAM_BOT_TOKEN")
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, bot *bot.Bot, update *models.Update) {}),
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		log.Fatal(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, handleStart)
	b.RegisterHandlerRegexp(bot.HandlerTypeMessageText, urlPattern, handleMediaURL)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "dl|",
		bot.MatchTypePrefix, handleDownloadCallback)

	slog.Info("PullMedia is running...")
	b.Start(ctx)
}
