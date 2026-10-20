package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/lrstanley/go-ytdlp"
)

type MediaOption struct {
	Label      string // "Audio Only", "144p", "360p", "480p", "720p", "1080p"
	FormatID   string // e.g., "251" for audio, or "398+251" for video+audio
	RawVideoID string
	Extension  string
	Codec      string
	TotalSize  int
}

func fetchVideoInfo(ctx context.Context, videoURL string) (*ytdlp.ExtractedInfo, error) {
	dl := ytdlp.New().NoPlaylist()

	infoList, _, err := dl.ExtractInfo(ctx, videoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch info: %w", err)
	}

	if len(infoList) == 0 {
		return nil, fmt.Errorf("no video info found")
	}

	return infoList[0], nil
}

func getResolutionTier(f *ytdlp.ExtractedFormat) int {
	// 1. Check if yt-dlp already labeled it in FormatNote (e.g., "720p60", "480p")
	note := valOr(f.FormatNote, "")
	for _, tier := range []int{144, 240, 360, 480, 720, 1080, 1440, 2160} {
		if strings.HasPrefix(note, fmt.Sprintf("%dp", tier)) {
			return tier
		}
	}

	// 2. Fallback: Use the smaller side of Width x Height (handles 9:16 and 1:1)
	if f.Width == nil || f.Height == nil {
		return 0
	}
	shortSide := int(*f.Width)
	if int(*f.Height) < shortSide {
		shortSide = int(*f.Height)
	}

	switch {
	case shortSide <= 160:
		return 144
	case shortSide <= 270:
		return 240
	case shortSide <= 400:
		return 360
	case shortSide <= 640:
		return 480
	case shortSide <= 768:
		return 720
	case shortSide <= 1152:
		return 1080
	default:
		return 0
	}
}

func buildCleanOptions(info *ytdlp.ExtractedInfo, preferCompatibleMP4 bool) []MediaOption {
	var bestAudio *ytdlp.ExtractedFormat
	bestVideoByTier := make(map[int]*ytdlp.ExtractedFormat)

	for _, f := range info.Formats {
		proto := valOr(f.Protocol, "")
		vcodec := valOr(f.VCodec, "none")
		acodec := valOr(f.ACodec, "none")
		formatID := valOr(f.FormatID, "")

		// 1. Only keep direct https streams with a known file size (drops 0.00 MiB m3u8)
		if proto != "https" || sizeBytes(f) == 0 {
			continue
		}

		// 2. Skip -drc audio tracks
		if strings.HasSuffix(formatID, "-drc") {
			continue
		}

		hasVideo := vcodec != "none" && vcodec != "images"
		hasAudio := acodec != "none" && acodec != "unknown"

		// --- AUDIO ONLY ---
		if hasAudio && !hasVideo {
			// If preferCompatibleMP4 is true, prefer m4a (AAC) over webm (Opus)
			if preferCompatibleMP4 && valOr(f.Extension, "") != "m4a" && bestAudio != nil {
				continue
			}

			if bestAudio != nil && isOriginalAudio(bestAudio) && !isOriginalAudio(f) {
				continue
			}

			bestAudio = f // Later formats in slice are higher quality
			continue
		}

		// --- VIDEO ONLY ---
		if hasVideo && !hasAudio {
			tier := getResolutionTier(f)
			if tier == 0 {
				continue
			}

			// Optional: If sending to Telegram/iOS, you might ONLY want avc1 (H.264)
			// because av01/vp9 sometimes won't play inline on older iPhones.
			if preferCompatibleMP4 && !strings.HasPrefix(vcodec, "avc1") {
				// Only skip if we already found an avc1 stream for this tier
				if existing, ok := bestVideoByTier[tier]; ok &&
					strings.HasPrefix(valOr(existing.VCodec, ""), "avc1") {
					continue
				}
			}

			// Overwrite with the current format (since yt-dlp orders worst -> best)
			bestVideoByTier[tier] = f
		}
	}

	var options []MediaOption
	var audioSize int
	var audioID string

	// 1. Add Audio Option First
	if bestAudio != nil {
		audioSize = sizeBytes(bestAudio)
		audioID = valOr(bestAudio.FormatID, "")
		options = append(options, MediaOption{
			Label:     "Audio Only",
			FormatID:  audioID,
			Extension: valOr(bestAudio.Extension, ""),
			Codec:     valOr(bestAudio.ACodec, ""),
			TotalSize: audioSize,
		})
	}

	// 2. Add Video Options in fixed order
	targetTiers := []int{144, 240, 360, 480, 720, 1080}
	for _, tier := range targetTiers {
		vf, exists := bestVideoByTier[tier]
		if !exists {
			continue
		}

		vidID := valOr(vf.FormatID, "")
		combinedID := vidID
		if audioID != "" {
			// Combine video + best audio for yt-dlp (e.g., "398+251" or "298+140")
			combinedID = fmt.Sprintf("%s+%s", vidID, audioID)
		}

		totalBytes := sizeBytes(vf) + audioSize

		options = append(options, MediaOption{
			Label:      fmt.Sprintf("%dp", tier),
			FormatID:   combinedID,
			RawVideoID: vidID,
			Extension:  "mp4", // When merging with ffmpeg, you usually output mp4
			Codec:      valOr(vf.VCodec, ""),
			TotalSize:  totalBytes,
		})
	}

	return options
}
