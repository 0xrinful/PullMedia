package main

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type DownloadJob struct {
	URL       string
	Title     string
	VideoID   string
	Options   []MediaOption
	CreatedAt time.Time
}

var jobCache sync.Map

func SaveJob(url string, title string, videoID string, options []MediaOption) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	key := hex.EncodeToString(b)

	jobCache.Store(key, DownloadJob{
		URL:       url,
		Title:     title,
		VideoID:   videoID,
		Options:   options,
		CreatedAt: time.Now(),
	})
	return key
}

func GetJob(key string) (DownloadJob, bool) {
	val, ok := jobCache.Load(key)
	if !ok {
		return DownloadJob{}, false
	}
	return val.(DownloadJob), true
}

func DeleteJob(key string) {
	jobCache.Delete(key)
}

func startJobCleaner() {
	ticker := time.NewTicker(1 * time.Hour)

	go func() {
		defer ticker.Stop()

		for range ticker.C {
			threshold := time.Now().Add(-12 * time.Hour)
			jobCache.Range(func(key, value interface{}) bool {
				job, ok := value.(DownloadJob)
				if ok && job.CreatedAt.Before(threshold) {
					jobCache.Delete(key)
				}
				return true
			})
		}
	}()
}
