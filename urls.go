package main

import (
	"errors"
	"regexp"
)

type Platform int

const (
	PlatformYouTube Platform = iota
	PlatformFacebook
	PlatformInstagram
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
	InstaReelPattern = regexp.MustCompile(
		`(?i)^https?://(?:www\.)?instagram\.com/(?:reel|reels|p|tv)/([a-zA-Z0-9_-]+)`,
	)

	// Facebook Reels, Watch, and mobile share links
	FBReelPattern = regexp.MustCompile(
		`(?i)^https?://(?:www\.|m\.|web\.)?facebook\.com/(?:reel/|share/r/|share/v/|watch/?\?v=|.*?\bvideos/)([a-zA-Z0-9_-]+)`,
	)
	FBWatchShortPattern = regexp.MustCompile(
		`(?i)^https?://fb\.watch/([a-zA-Z0-9_-]+)`,
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
	case InstaReelPattern.MatchString(rawURL):
		url.Platform = PlatformInstagram
		url.VideoType = VideoTypeShort
	case FBReelPattern.MatchString(rawURL), FBWatchShortPattern.MatchString(rawURL):
		url.Platform = PlatformFacebook
		url.VideoType = VideoTypeShort
	default:
		return nil, ErrUnsupportedLink
	}

	return &url, nil
}
