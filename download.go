package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lrstanley/go-ytdlp"
)

func downloadMediaOption(
	ctx context.Context,
	videoURL string,
	jobKey string,
	opt MediaOption,
	outputDir string,
) (string, error) {
	isAudioOnly := opt.Label == "Audio Only"

	// 1. Generate a unique session ID for this specific download task
	sessionID := fmt.Sprintf("pullmedia_%s", jobKey)

	// 2. Tell yt-dlp where to save the file using %(ext)s placeholder
	outputTemplate := filepath.Join(outputDir, fmt.Sprintf("%s.%%(ext)s", sessionID))

	dl := ytdlp.New().
		NoPlaylist().
		Format(opt.FormatID).
		Output(outputTemplate).
		Cookies(cookiesFile)

	// 3. Determine what the final file extension on disk will be
	finalExt := opt.Extension
	if isAudioOnly {
		// Only happens on IG/FB when no standalone audio stream exists:
		// tells ffmpeg to strip the audio out of the combined MP4 and save as .m4a
		if opt.FormatID == "bestaudio/best" {
			dl.ExtractAudio().AudioFormat("m4a")
			finalExt = "m4a"
		}
		// Note: For YouTube (or IG DASH audio), opt.FormatID is a real stream ID (e.g. "251" or "140"),
		// so it skips ExtractAudio() and downloads the raw .webm / .m4a directly just like before!
	} else {
		dl.MergeOutputFormat("mp4")
		finalExt = "mp4"
	}

	if _, err := dl.Run(ctx, videoURL); err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	// 5. Return the exact path of the finished file on disk
	finalFilePath := filepath.Join(outputDir, fmt.Sprintf("%s.%s", sessionID, finalExt))
	return finalFilePath, nil
}
