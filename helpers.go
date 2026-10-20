package main

import (
	"fmt"
	"strings"

	"github.com/lrstanley/go-ytdlp"
)

func valOr(s *string, fallback string) string {
	if s == nil || *s == "" {
		return fallback
	}
	return *s
}

func sizeBytes(f *ytdlp.ExtractedFormat) int {
	if f.FileSize != nil && *f.FileSize > 0 {
		return *f.FileSize
	}
	if f.FileSizeApprox != nil && *f.FileSizeApprox > 0 {
		return *f.FileSizeApprox
	}
	return 0
}

func isOriginalAudio(f *ytdlp.ExtractedFormat) bool {
	note := strings.ToLower(valOr(f.FormatNote, ""))
	return strings.Contains(note, "original") || strings.Contains(note, "default")
}

func fmtSize(bytes int) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(bytes)
	cur := 0

	for size >= 1024 && cur < len(units)-1 {
		size /= 1024
		cur++
	}

	return fmt.Sprintf("%.2f %s", size, units[cur])
}

func fmtDuration(seconds *float64) string {
	if seconds == nil || *seconds <= 0 {
		return "00:00"
	}

	total := int(*seconds)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
