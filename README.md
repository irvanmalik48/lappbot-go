# Lappbot

A better Lappbot. Built upon the legacy of the thing I made with Typescript in the past.

## Features

### Moderation

- **Kick/Ban/Mute**: Standard moderation commands.
- **Timed Actions**: Temporarily ban or mute users (`/tban`, `/tmute`).
- **Silent Actions**: Perform actions without deleting the user's message or notifying (`/skick`, `/smute`, `/sban`).
- **Purge**: Multithreaded message deletion for cleaning up chat history.
- **Pin**: Pin and unpin messages easily.

### Blacklist System

A comprehensive blacklist system to automatically filter unwanted content.

- **Types**:
  - `regex`: Block patterns in text and captions.
  - `sticker_set`: Block entire sticker packs.
  - `emoji`: Block specific custom emojis.
- **Actions**: Configurable per rule (`delete`, `soft_warn`, `hard_warn`, `kick`, `mute`, `ban`).
- **Exemptions**: Admins and "Approved Users" are exempt from checks.

### Administration

- **Promote/Demote**: Manage admin storage directly via the bot. `/promote` grants safe admin rights (cannot add new admins).
- **Approve**: Whitelist trusted users to bypass filters and blacklists.
- **Realm Actions**: (Bot Owner Only) Global ban/mute across all groups the bot manages.

### Utilities

- **Welcome**: Customizable welcome messages.
- **CAPTCHA**: Verification for new members to prevent bots.
- **Filters**: Custom text triggers and responses.
- **Report**: User reporting system.

## Requirements

- Go 1.25+
- PostgreSQL

  git clone https://github.com/irvanmalik48/lappbot
  cd lappbot

3.  **Configuration**
    Copy `.env.example` to `.env` (or create it) and fill in your details.

4.  **Run Migrations**
    The bot handles migrations automatically on startup using `golang-migrate`.

5.  **Build and Run**
    ```bash
    go mod download
    go run cmd/bot/main.go
    ```

## Usage

### Basic Commands

- `/help`: Show all commands.
- `/ping`: Check latency.
- `/report`: Report a message to admins.

### Moderation Commands

- `/warn`: Warn a user (3 warns = kick).
- `/ban`, `/mute`, `/kick`: Standard actions.
- `/bl <type> <value> [action]`: Add a blacklist rule.
  - Example: `/bl regex "bad word" ban`

### Admin Commands

- `/approve`: Approve a user.
- `/promote`: Promote a user to admin.

## License

RCCL 2.0
