# MemoBot

A simple memo reminder application that allows users to create memos and get reminded at specified times.

README is generated with LLM.

## Prerequisites

- Go 1.16 or later
- PostgreSQL
- sqlc

## Setup

1. Install sqlc:
```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

2. Create a PostgreSQL database:
```bash
createdb memodb
```

3. Create the database schema:
```bash
psql memodb < internal/db/schema.sql
```

4. Configure the application:
   - Copy `.env.example` to `.env`
   - Edit `.env` with your database credentials and Discord bot token
   - Never commit the `.env` file to version control

5. Generate the database code:
```bash
sqlc generate
```

6. Install dependencies:
```bash
go mod tidy
```

## Running the Bot

```bash
go run cmd/main.go
```


## Usage

The CLI application provides the following commands:

1. Add memo: Create a new memo with content and reminder time
2. List pending memos: View all your pending memos
3. Delete memo: Delete a specific memo by ID
4. Exit: Close the application

When adding a memo:
- Enter the memo content (press Enter twice to finish)
- Enter the reminder time in format: in natural language, like `in 5 min`, `today at 3pm`, or `YYYY-MM-DD HH:MM`

The backend service will:
- Check for pending reminders every minute
- Display reminders in the terminal when their time comes
- Process any missed reminders when started

## Database Schema

The application uses two tables:
- `users`: Stores user information (ID, username, preferred Discord channel)
- `memos`: Stores memo content and reminder times (content, user ID, channel ID, reminder time)

## Configuration

The application uses environment variables for configuration. Copy `.env.example` to `.env` and set the following variables:

### Database Configuration
- `DB_HOST`: Database host (default: localhost)
- `DB_PORT`: Database port (default: 5432)
- `DB_USER`: Database user (default: postgres)
- `DB_PASSWORD`: Database password (required)
- `DB_NAME`: Database name (default: memodb)
- `DB_SSLMODE`: Database SSL mode (default: disable)

### Application Configuration
- `SCAN_INTERVAL`: How often to check for pending reminders (default: 60s)
- `TIMEZONE`: Application timezone (default: UTC)

### Discord Configuration
- `DISCORD_BOT_TOKEN`: Your Discord bot token (required)

**Note:** Never commit your `.env` file to version control as it contains sensitive information.
