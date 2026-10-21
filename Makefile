-include .env
export

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ -/'

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #
## run: run the application
.PHONY: run
run:
	@go run .

# ==================================================================================== #
# BUILD
# ==================================================================================== #
.PHONY: build
build:
	@go build -ldflags="-s -w" -trimpath -o pullmedia .
	@echo "Binary built successfully."

# ==================================================================================== #
# DEPLOY
# ==================================================================================== #
.PHONY: deploy
deploy: build
	@echo "Stopping pullmedia service..."
	@ssh $(VM_HOST) "systemctl stop pullmedia || true"
	@echo "Uploading binary to VM..."
	@ssh $(VM_HOST) "mkdir -p $(VM_DIR)"
	@scp pullmedia $(VM_HOST):$(VM_DIR)/pullmedia
	@scp cookies.txt $(VM_HOST):$(VM_DIR)/cookies.txt
	@echo "Starting pullmedia service..."
	@ssh $(VM_HOST) "systemctl start pullmedia"
	@echo "Deployment complete!"

# ==================================================================================== #
# MAINTENANCE
# ==================================================================================== #
.PHONY: clean
clean:
	@rm -rf /tmp/pullmedia_*
	@echo "Cleaned up residual temp files."


.PHONY: cookies
cookies:
	@rm cookies.txt
	@yt-dlp --cookies-from-browser firefox --cookies cookies.txt "https://www.youtube.com/" --flat-playlist --playlist-items 1 >/dev/null 2>&1
	@echo "Exported Firefox cookies to cookies.txt."
