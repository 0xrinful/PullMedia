# Media Downloader Telegram Bot

A Telegram bot written in Go to download videos, Shorts, Reels, and audio from YouTube, Instagram, and Facebook with interactive quality selection using `yt-dlp` and `ffmpeg`.

---

## Features

- **Multi-Platform Support:**
  - **YouTube:** Standard videos, Shorts, and pure audio tracks.
  - **Instagram:** Reels and standard video posts.
  - **Facebook:** Reels, public videos, and mobile share links.
- **Interactive Quality Selection:** Generates inline buttons for available resolutions (from 144p up to 1080p+) and an "Audio Only" option.
- **Direct Stream Merging:** Combines video and audio streams into `.mp4` using `ffmpeg` when separate DASH streams are provided.
- **Clean Audio Extraction:** Downloads raw audio streams directly when available (such as YouTube's `.m4a` or `.webm`) or extracts audio from combined video streams when necessary.

---

## Prerequisites

Before running the bot, ensure the following dependencies are installed and accessible in your system's `PATH`:

1. **[Go](https://go.dev/dl/)** (v1.21 or later)
2. **[yt-dlp](https://github.com/yt-dlp/yt-dlp)** (Keep this updated regularly)
3. **[ffmpeg](https://ffmpeg.org/download.html)**

---

## Configuration

Set the required environment variables in your shell or `.env` file:

```bash
# Telegram Bot API token from @BotFather
export TELEGRAM_BOT_TOKEN="your_bot_token_here"

# Maximum number of concurrent media downloads
export DOWNLOAD_CONCURRENCY=5

# (Optional but recommended) Path to your Netscape-formatted cookies file
# Helps avoid bot detection and rate limits
export COOKIES_FILE="./cookies.txt"
```

---

## Installation & Running

1. **Clone the repository:**
  
  ```bash
  git clone https://github.com/0xrinful/PullMedia.git
  cd PullMedia
  ```
  
2. **Download dependencies:**
  
  ```bash
  go mod download
  ```
  
3. **Run the bot:**
  
  ```bash
  make run
  ```
  
4. **Build a binary (production):**
  
  ```bash
  make build
  ```
  

---

## How It Works

1. **URL Validation (`ParseMediaURL`):** Validates the incoming message against platform-specific regex patterns and labels the platform (YouTube, Instagram, or Facebook) and content type (Regular or Short/Reel).
2. **Metadata Extraction (`fetchVideoInfo`):** Calls `yt-dlp` to extract available stream formats, resolution options, and codecs without downloading the media payload upfront.
3. **Stream Filtering (`buildCleanOptions`):**
  - Applies strict HTTPS and filesize requirements for YouTube DASH formats.
  - Handles Instagram and Facebook formats where exact byte sizes are omitted by Meta's CDN and supports both DASH and pre-merged progressive MP4s.
4. **Download & Delivery (`downloadMediaOption`):**
  - Merges streams or extracts audio based on the user's inline button selection.
  - Cleans up temporary disk storage once the file is sent to Telegram.

---

## License

This project is licensed under the [MIT License](LICENSE).
