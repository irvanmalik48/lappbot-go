# Lappbot

A better Lappbot. Built upon the legacy of the thing I made with Typescript in the past.

## Features

Whatever Rose has, Lappbot is trying to reimplement it, so yeah.

## Requirements

- Go 1.25+
- PostgreSQL
- Valkey
- Local Telegram Bot API (aiogram/telegram-bot-api or something idk)

1.  `git clone https://github.com/irvanmalik48/lappbot-go`

2.  `cd lappbot-go`

3.  Copy `.env.example` to `.env` (or create it) and fill in your details.
    - **Note**: For Local Bot API, ensure you provide `TELEGRAM_API_ID` and `TELEGRAM_API_HASH`.
    - Set `BOT_API_URL` correctly (e.g., `http://127.0.0.1:8081` or whatever port you configured).
    - **Long Polling**: To use Long Polling (default), ensure `WEBHOOK_URL` is empty. The bot will automatically delete any existing webhook on startup.

4.  The bot handles migrations automatically on startup using `golang-migrate`.

5.  ```bash
    go mod download
    go run cmd/bot/main.go
    ```

## Deployment

### Docker Compose

1.  Ensure Docker and Docker Compose are installed.
2.  Run the stack:
    ```bash
    docker-compose up -d --build
    ```

## Usage

Use `/help` to see all commands, duh.

## License

RCCL 2.0
