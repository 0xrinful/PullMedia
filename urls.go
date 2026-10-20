package main

import (
	"errors"
	"regexp"
)

type Platform int

const (
	PlatformYouTube Platform = iota
	PlatformFacebook
)

type VideoType int

const (
	VideoTypeRegular VideoType = iota
	VideoTypeShort
)

var (
	urlPattern       = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	RegularYTPattern = regexp.MustCompile(
		`(?i)^https?://(?:www\.|m\.)?(?:youtube\.com/(?:watch\?.*v=|embed/|live/|v/)|youtu\.be/)([a-zA-Z0-9_-]{11})`,
	)
	ShortsYTPattern = regexp.MustCompile(
		`(?i)^https?://(?:www\.|m\.)?youtube\.com/shorts/([a-zA-Z0-9_-]{11})`,
	)
)

var ErrUnsupportedLink = errors.New("unsupported link")

type MediaURL struct {
	Value     string
	Platform  Platform
	VideoType VideoType
}

func ParseMediaURL(rawURL string) (*MediaURL, error) {
	url := MediaURL{Value: rawURL}
	switch {
	case RegularYTPattern.MatchString(rawURL):
		url.Platform = PlatformYouTube
		url.VideoType = VideoTypeRegular
	case ShortsYTPattern.MatchString(rawURL):
		url.Platform = PlatformYouTube
		url.VideoType = VideoTypeShort
	default:
		return nil, ErrUnsupportedLink
	}

	return &url, nil
}
