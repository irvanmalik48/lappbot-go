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

## BotFather Commands

```
start - Start the bot
help - Get help
ping - Check latency
version - Check version
report - Report a message
actiontopic - Get action topic
setactiontopic - Set action topic
newtopic - Create topic
renametopic - Rename topic
closetopic - Close topic
reopentopic - Reopen topic
deletetopic - Delete topic
get - Get note
save - Save note
clear - Delete note
notes - List notes
clearall - Delete all notes
privatenotes - Toggle private mode
connect - Connect to Chat
disconnect - Disconnect
reconnect - Reconnect
connection - Check Connection
kick - Kick (Reply)
ban - Ban (Reply)
tban - Timed Ban (Reply)
mute - Mute (Reply)
tmute - Timed Mute (Reply)
skick - Silent Kick (Reply)
sban - Silent Ban (Reply)
smute - Silent Mute (Reply)
unban - Unban (Reply)
unmute - Unmute (Reply)
pin - Pin (Reply)
lock - Lock Group
unlock - Unlock Group
purge - Purge messages
spurge - Silent purge
del - Delete message
purgefrom - Mark start
purgeto - Purge range
warn - Warn (Reply)
dwarn - Warn & Delete
swarn - Silent Warn
rmwarn - Remove Last Warn (Reply)
resetwarn - Reset Warns (Reply)
resetallwarns - Reset Chat Warns
warns - Check Warns
warnings - Check Settings
warnlimit - Set Limit
warnmode - Set Action
warntime - Set Duration
antiraid - Toggle Anti-Raid
raidtime - Set Anti-Raid duration
raidactiontime - Ban duration
autoantiraid - Auto-enable Anti-Raid
flood - Flood Settings
setflood - Consecutive limit
setfloodtimer - Timed limit
floodmode - Flood Action
clearflood - Delete flood
welcome - Welcome Msg
goodbye - Goodbye Msg
captcha - CAPTCHA
filter - Add filter (reply)
stop - Remove filter
filters - List filters
promote - Promote
demote - Demote
approve - Exempt User
unapprove - Revoke Exemption
bl - Blacklist
unbl - Unblacklist
blacklist - List Rules
rban - Realm Ban (Reply)
rmute - Realm Mute (Reply)
zalgo - Zalgo text generator
uwuify - UwU text transformation
emojify - Emojify text
leetify - Leetify text
```

## License

RCCL 2.0
