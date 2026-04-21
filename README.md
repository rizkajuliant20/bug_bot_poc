# Slack Bug Bot

A high-performance Slack bot written in Go that automatically creates bug tickets in Notion with AI-powered diagnosis and **multi-issue detection**.

## Why Go?

Built with Go for optimal performance and deployment simplicity:
- ✅ **Easy Deployment**: Single compiled binary (9.8MB), no runtime dependencies
- ✅ **Better Performance**: Lower memory usage (~30MB) and faster execution
- ✅ **Smaller Docker Images**: ~25MB vs ~300MB for Node.js
- ✅ **Production Ready**: Built-in concurrency and robust error handling

## ✨ Key Features

### 🎯 Multi-Issue Detection (NEW!)
Bot automatically analyzes thread + all replies to detect **multiple distinct issues** and lets you choose which ones to create tickets for.

**Example:**
```
User: "Ada 3 masalah:
1. Spin tidak bisa diklik
2. 🚀 Payment gagal tapi saldo terpotong  
3. Notifikasi tidak muncul"

Bot: 
🔍 Detected 3 separate issues in this thread:

Issue 1: Spin tidak bisa diklik
└ Description...
└ Severity: high | Category: Frontend

Issue 2: Payment gagal tapi saldo terpotong
└ Description...
└ Severity: critical | Category: Backend

Issue 3: Notifikasi tidak muncul
└ Description...
└ Severity: medium | Category: Backend

Which issues would you like to create tickets for?

[✅ Create Issue 1] [✅ Create Issue 2] [✅ Create Issue 3]
[✅ Create All Issues] [❌ Cancel]
```

**Interactive Buttons:**
- Click specific issue to create only that ticket
- Click "Create All" to create all tickets at once
- Click "Cancel" to abort
- Buttons auto-disable after click to prevent double submission

### 🤖 AI-Powered Analysis
- **Smart Diagnosis**: Severity, category, priority, root cause analysis
- **App Detection**: Auto-detects Jago App, Jagoan App, Depot Portal, Service
- **Thread Summarization**: Condenses long discussions into key points
- **Multi-Issue Recognition**: Identifies distinct problems in conversations

### 💬 Slack Integration
- **Triggers**: 
  - 🐞 Emoji reaction (ladybug, bug, beetle)
  - @mention with bug keywords
  - `/bug` slash command
- **Smart Emoji Management**:
  - 👀 Processing indicator (auto-removed on completion/error)
  - ✅ Success indicator
  - ❌ Error indicator
  - Auto-remove old status emojis on re-trigger

### 📝 Notion Integration
- Auto-creates structured tickets with full context
- Properties: Status, Severity, Category, Priority, Platform, Team
- Includes: Description, Diagnosis, Reporter, Slack Thread URL, Thread Summary
- Polling for manually created bugs (every 2 minutes)

### 📊 Comprehensive Logging
- Daily log rotation with automatic cleanup
- App-specific log files (Jago App, Jagoan App, Depot Portal, Service)
- Separate error logs
- All logs in `logs/` directory

## 📖 Usage Examples

### Single Issue
```
1. User posts bug report in Slack
2. Add 🐞 emoji reaction to the message
3. Bot analyzes with AI
4. Bot creates Notion ticket automatically
5. Bot replies with ticket link
```

### Multiple Issues
```
1. User posts message with multiple problems
2. Add 🐞 emoji reaction
3. Bot detects 3 separate issues
4. Bot shows interactive buttons
5. User clicks "Create All Issues"
6. Bot creates 3 Notion tickets
7. Bot replies with all ticket links
```

### Re-trigger
```
1. Bug report already processed (has ✅ or ❌)
2. Add 🐞 emoji again to re-process
3. Bot removes old status emoji
4. Bot re-analyzes and creates new ticket
```

## 🚀 Quick Start

### 1. Install Go

```bash
# macOS
brew install go

# Linux
wget https://go.dev/dl/go1.26.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.26.linux-amd64.tar.gz
```

### 2. Setup

```bash
cd go-bug-bot

# Install dependencies
make install

# Copy environment file
cp .env.example .env

# Edit .env with your credentials
nano .env
```

### 3. Run

```bash
# Development
make run

# Or build and run
make build
./bug-bot
```

## Docker Deployment

### Build Image

```bash
make docker-build
```

### Run Container

```bash
docker run -d \
  --name bug-bot \
  --env-file .env \
  -v $(pwd)/logs:/root/logs \
  bug-bot:latest
```

### Docker Compose

```yaml
version: '3.8'

services:
  bug-bot:
    build: .
    env_file: .env
    volumes:
      - ./logs:/root/logs
      - ./.notion-tracking.json:/root/.notion-tracking.json
    restart: unless-stopped
```

## Deployment Options

### 1. Single Binary Deployment

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 go build -o bug-bot ./cmd/bot

# Copy to server
scp bug-bot user@server:/opt/bug-bot/
scp .env user@server:/opt/bug-bot/

# Run on server
ssh user@server
cd /opt/bug-bot
./bug-bot
```

### 2. Systemd Service

Create `/etc/systemd/system/bug-bot.service`:

```ini
[Unit]
Description=Slack Bug Bot
After=network.target

[Service]
Type=simple
User=bugbot
WorkingDirectory=/opt/bug-bot
ExecStart=/opt/bug-bot/bug-bot
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable bug-bot
sudo systemctl start bug-bot
sudo systemctl status bug-bot
```

### 3. Cloud Platforms

#### Railway
```bash
# Install Railway CLI
npm install -g @railway/cli

# Deploy
railway login
railway init
railway up
```

#### Fly.io
```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Deploy
fly launch
fly deploy
```

#### Google Cloud Run
```bash
# Build and push
gcloud builds submit --tag gcr.io/PROJECT_ID/bug-bot

# Deploy
gcloud run deploy bug-bot \
  --image gcr.io/PROJECT_ID/bug-bot \
  --platform managed \
  --region us-central1 \
  --set-env-vars-file .env
```

## Project Structure

```
go-bug-bot/
├── cmd/
│   └── bot/
│       └── main.go              # Application entry point
├── pkg/
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── logger/
│   │   └── logger.go            # Logging utility
│   ├── services/
│   │   ├── slack.go             # Slack API integration
│   │   ├── notion.go            # Notion API integration
│   │   └── openai.go            # OpenAI API integration
│   └── handlers/
│       └── bug_handler.go       # Bug report handling logic
├── logs/                        # Log files (auto-created)
├── Dockerfile                   # Docker configuration
├── Makefile                     # Build automation
├── go.mod                       # Go dependencies
└── README.md
```

## Environment Variables

```env
# Slack Configuration
SLACK_BOT_TOKEN=xoxb-...
SLACK_SIGNING_SECRET=...
SLACK_APP_TOKEN=xapp-...
SLACK_BUG_TRACKING_CHANNEL=C0AT3CEB5ED  # Optional

# Notion Configuration
NOTION_API_KEY=secret_...
NOTION_DATABASE_ID=...

# OpenAI Configuration
OPENAI_API_KEY=sk-...

# Server Configuration
PORT=3000
```

## Performance Comparison

| Metric | Node.js | Go |
|--------|---------|-----|
| Binary Size | ~200MB (with node_modules) | ~15MB |
| Memory Usage | ~150MB | ~30MB |
| Startup Time | ~2s | ~100ms |
| Docker Image | ~300MB | ~25MB |

## Development

### Install Development Tools

```bash
# Air for hot reload
go install github.com/cosmtrek/air@latest

# Run with auto-reload
make dev
```

### Run Tests

```bash
make test
```

### Format Code

```bash
make fmt
```

## Logging

Logs are stored in `logs/` directory:

```bash
# View main log
tail -f logs/app-$(date +%Y-%m-%d).log

# View app-specific logs
tail -f logs/jago-app-$(date +%Y-%m-%d).log
tail -f logs/jagoan-app-$(date +%Y-%m-%d).log

# View errors
tail -f logs/error-$(date +%Y-%m-%d).log
```

## 🔧 Troubleshooting

### Multi-issue detection not working
```bash
# Check if thread has multiple distinct issues
# Bot only shows buttons if >1 issue detected

# View logs to see detection result
tail -f logs/app-$(date +%Y-%m-%d).log | grep "Multi-issue"

# If detection fails, bot falls back to single issue mode
```

### Buttons not appearing
```bash
# Buttons only appear when multiple issues detected
# Check logs for "Multiple issues detected"

# If only 1 issue found, bot creates ticket automatically
# No buttons needed for single issue
```

### Buttons still clickable after click
```bash
# This should not happen with latest version
# Buttons auto-disable immediately after click

# If issue persists, restart bot:
pkill bug-bot
./bug-bot
```

### Bot doesn't start
```bash
# Check logs
./bug-bot 2>&1 | tee startup.log

# Verify environment variables
env | grep SLACK
env | grep NOTION
env | grep OPENAI
```

### Connection issues
```bash
# Test Slack connection
curl -H "Authorization: Bearer $SLACK_BOT_TOKEN" \
  https://slack.com/api/auth.test

# Test Notion connection
curl -H "Authorization: Bearer $NOTION_API_KEY" \
  https://api.notion.com/v1/users/me
```

## License

MIT
