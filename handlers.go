package main

import (
	"context"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/lrstanley/go-ytdlp"
)

func handleStart(
	ctx context.Context,
	b *bot.Bot,
	update *models.Update,
) {
	if update.Message == nil {
		return
	}

	welcomeText := "⚡ <b>Welcome to PullMedia</b>\n\n" +
		"Send a link to download video or audio in your preferred quality.\n\n" +
		"<blockquote>🎬 144p – 1080p MP4  •  🎧 Audio Only</blockquote>\n\n" +
		"🔗 <i>Drop a URL below to start.</i>"

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      welcomeText,
		ParseMode: models.ParseModeHTML,
	})
}

func handleMediaURL(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	rawText := strings.TrimSpace(update.Message.Text)
	chatID := update.Message.Chat.ID

	url, err := ParseMediaURL(rawText)
	if err != nil {
		sendMessage(ctx, b, update, "❌ This link isn't supported.")
	}

	statusMsg, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      "🔍 <i>Fetching video formats...</i>",
		ParseMode: models.ParseModeHTML,
	})

	info, err := fetchVideoInfo(ctx, url.Value)
	if err != nil {
		editMessage(ctx, b, chatID, statusMsg, "❌ Failed to fetch video info.")
		return
	}

	options := buildCleanOptions(info, false)
	if len(options) == 0 {
		editMessage(ctx, b, chatID, statusMsg, "❌ No compatible formats found.")
		return

	}

	// Save options in memory and get the 8-char jobKey
	title := safeFilename(valOr(info.Title, "Untitled"))
	jobKey := SaveJob(url.Value, title, info.ID, options)

	deleteMessage(ctx, b, chatID, statusMsg)
	sendMediaCard(ctx, b, chatID, jobKey, info, options)
}

func handleDownloadCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	downloadSemaphore <- struct{}{}
	defer func() { <-downloadSemaphore }()

	cb := update.CallbackQuery
	if cb == nil || cb.Message.Message == nil {
		return
	}

	// 1. Parse "dl|<jobKey>|<optionIndex>"
	parts := strings.Split(cb.Data, "|")
	if len(parts) != 3 || parts[0] != "dl" {
		return
	}
	jobKey := parts[1]
	optIdx, err := strconv.Atoi(parts[2])
	if err != nil {
		return
	}

	// 2. Look up the job in memory
	job, ok := GetJob(jobKey)
	if !ok || optIdx < 0 || optIdx >= len(job.Options) {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: cb.ID,
			Text:            "⚠️ Session expired. Please send the link again.",
			ShowAlert:       true,
		})
		return
	}

	DeleteJob(jobKey)

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: cb.ID,
		Text:            "Starting download...",
	})

	selectedOpt := job.Options[optIdx]
	chatID := cb.Message.Message.Chat.ID
	cardMsgID := cb.Message.Message.ID

	// 3. Update the card caption to lock the buttons while downloading
	_, _ = b.EditMessageCaption(ctx, &bot.EditMessageCaptionParams{
		ChatID:    chatID,
		MessageID: cardMsgID,
		Caption:   fmt.Sprintf("⏳ <b>Downloading %s...</b>", selectedOpt.Label),
		ParseMode: models.ParseModeHTML,
	})

	// 4. Download the file to outputDir
	filePath, err := downloadMediaOption(ctx, job.URL, job.VideoID, selectedOpt, "/tmp")
	if err != nil {
		_, _ = b.EditMessageCaption(ctx, &bot.EditMessageCaptionParams{
			ChatID:    chatID,
			MessageID: cardMsgID,
			Caption:   "❌ <b>Download failed.</b> Please try again.",
			ParseMode: models.ParseModeHTML,
		})
		return
	}
	defer os.Remove(filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	// 5. Update card status & trigger Telegram's native "uploading..." header animation
	_, _ = b.EditMessageCaption(ctx, &bot.EditMessageCaptionParams{
		ChatID:    chatID,
		MessageID: cardMsgID,
		Caption:   "⬆️ <b>Uploading to Telegram...</b>",
		ParseMode: models.ParseModeHTML,
	})

	isAudioOnly := selectedOpt.Label == "Audio Only"
	inputFile := &models.InputFileUpload{
		Filename: filepath.Base(job.Title),
		Data:     file,
	}

	// 6. Send the new message (SendAudio or SendVideo)
	if isAudioOnly {
		_, _ = b.SendChatAction(ctx, &bot.SendChatActionParams{
			ChatID: chatID,
			Action: models.ChatActionUploadVoice,
		})
		_, err = b.SendAudio(ctx, &bot.SendAudioParams{
			ChatID: chatID,
			Audio:  inputFile,
		})
	} else {
		_, _ = b.SendChatAction(ctx, &bot.SendChatActionParams{
			ChatID: chatID,
			Action: models.ChatActionUploadVideo,
		})
		_, err = b.SendVideo(ctx, &bot.SendVideoParams{
			ChatID:            chatID,
			Video:             inputFile,
			SupportsStreaming: true,
		})
	}

	if err != nil {
		_, _ = b.EditMessageCaption(ctx, &bot.EditMessageCaptionParams{
			ChatID:    chatID,
			MessageID: cardMsgID,
			Caption:   fmt.Sprintf("❌ <b>Upload failed:</b> %s", html.EscapeString(err.Error())),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	// 7. Upload succeeded! Delete the temporary thumbnail status card
	_, _ = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: cardMsgID,
	})
}

func sendMediaCard(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	jobKey string,
	info *ytdlp.ExtractedInfo,
	options []MediaOption,
) error {
	title := html.EscapeString(valOr(info.Title, "Untitled Video"))
	duration := fmtDuration(info.Duration)
	thumbnailURL := valOr(info.Thumbnail, "")

	// Clean HTML Caption
	caption := fmt.Sprintf(
		"🎬 <b>%s</b>\n\n"+
			"<blockquote>⏱ <b>Duration:</b> <b>%s</b></blockquote>\n\n"+
			"<b>✦ <i>Select your preferred quality below</i></b>",
		title,
		duration,
	)

	kb := buildFormatKeyboard(jobKey, options)

	_, err := b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID: chatID,
		Photo: &models.InputFileString{
			Data: thumbnailURL,
		}, // Telegram fetches the URL directly
		Caption:     caption,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: kb,
	})
	return err
}

func buildFormatKeyboard(jobKey string, options []MediaOption) *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton
	var currentRow []models.InlineKeyboardButton
	var audioBtn *models.InlineKeyboardButton

	for idx, opt := range options {
		// ⚠️ Telegram bots have a strict 50 MB upload limit on standard Bot API!
		// Skip options over 49.5 MB so the upload doesn't fail.
		if float64(opt.TotalSize)/(1024*1024) > 49.5 {
			continue
		}

		// Pack everything needed to download into CallbackData (max 64 bytes)
		// Format: "dl|<jobKey>|<index>"
		// Example: "dl|f4a91c02|3" (14 bytes)
		cbData := fmt.Sprintf("dl|%s|%d", jobKey, idx)

		sizeText := ""
		if opt.TotalSize > 0 {
			sizeText = fmt.Sprintf(" • %s", fmtSize(opt.TotalSize))
		}

		if opt.Label == "Audio Only" {
			btn := models.InlineKeyboardButton{
				Text:         fmt.Sprintf("🎧 Audio Only%s", sizeText),
				CallbackData: cbData,
			}
			audioBtn = &btn
			continue
		}

		btn := models.InlineKeyboardButton{
			Text:         fmt.Sprintf("🎬 %s%s", opt.Label, sizeText),
			CallbackData: cbData,
		}
		currentRow = append(currentRow, btn)

		// Push every 2 buttons as a row (2-column grid)
		if len(currentRow) == 2 {
			rows = append(rows, currentRow)
			currentRow = nil
		}
	}

	// If there's 1 leftover video button, push it as its own row
	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	// Put Audio Only on a full-width row at the very bottom
	if audioBtn != nil {
		rows = append(rows, []models.InlineKeyboardButton{*audioBtn})
	}

	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func sendMessage(ctx context.Context, b *bot.Bot, update *models.Update, text string) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
	})
}

func editMessage(ctx context.Context, b *bot.Bot, chatID int64,
	message *models.Message, text string,
) {
	if message != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: message.ID,
			Text:      "❌ Failed to fetch video info.",
		})
	}
}

func deleteMessage(ctx context.Context, b *bot.Bot, chatID int64, message *models.Message) {
	if message != nil {
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    chatID,
			MessageID: message.ID,
		})
	}
}
