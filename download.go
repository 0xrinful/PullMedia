package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lrstanley/go-ytdlp"
)

func downloadMediaOption(
	ctx context.Context,
	videoURL string,
	jobKey string,
	opt MediaOption,
	outputDir string,
) (string, error) {
	isAudioOnly := !strings.Contains(opt.FormatID, "+")

	// 1. Generate a unique session ID for this specific download task
	sessionID := fmt.Sprintf("pullmedia_%s", jobKey)

	// 2. Tell yt-dlp where to save the file using %(ext)s placeholder
	outputTemplate := filepath.Join(outputDir, fmt.Sprintf("%s.%%(ext)s", sessionID))

	dl := ytdlp.New().
		NoPlaylist().
		Format(opt.FormatID).
		Output(outputTemplate)

	// 3. Determine what the final file extension on disk will be
	finalExt := opt.Extension // For audio-only (e.g. "m4a" or "webm")
	if !isAudioOnly {
		dl.MergeOutputFormat("mp4")
		finalExt = "mp4"
	}

	if _, err := dl.Run(ctx, videoURL); err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	// 4. Return the exact path of the finished file on disk
	finalFilePath := filepath.Join(outputDir, fmt.Sprintf("%s.%s", sessionID, finalExt))
	return finalFilePath, nil
}
