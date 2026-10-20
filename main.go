package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/lmittmann/tint"
	"github.com/lrstanley/go-ytdlp"
)

var downloadSemaphore = make(chan struct{}, getEnvInt("DOWNLOAD_CONCURRENCY", 5))

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	ytdlp.MustInstall(ctx, nil)
	ytdlp.MustInstallFFmpeg(ctx, nil)

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

func getEnvInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
