# Slack Bug Bot

A high-performance Slack bot written in Go that automatically creates bug tickets in Notion with **GPT-4o-powered diagnosis**, **multi-issue detection**, and **auto-assignment**.

## Why Go?

Built with Go for optimal performance and deployment simplicity:
- ✅ **Easy Deployment**: Single compiled binary (9.8MB), no runtime dependencies
- ✅ **Better Performance**: Lower memory usage (~30MB) and faster execution
- ✅ **Smaller Docker Images**: ~25MB vs ~300MB for Node.js
- ✅ **Production Ready**: Built-in concurrency and robust error handling

## ✨ Key Features

### 🎯 Multi-Issue Detection with Interactive UI
Bot automatically analyzes thread + all replies to detect **multiple distinct issues** with a clean, interactive interface.

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
- **Processing indicators**: "⏳ Creating ticket... Please wait"
- **Message updates**: Summary replaced with success message + "View in Notion" button
- **Clean UX**: No duplicate messages, single message updates in place

### 🤖 AI-Powered Analysis (GPT-4o)
- **Model**: GPT-4o for superior context understanding and accuracy
- **Smart Diagnosis**: Severity, category, priority, root cause analysis
- **App Detection**: Auto-detects Jago App, Jagoan App, Depot Portal, Service
- **Thread Summarization**: Condenses long discussions into key points
- **Multi-Issue Recognition**: Identifies distinct problems in conversations
- **Accurate Titles**: Extracts exact keywords from bug description (no hallucination)

### 👥 Auto-Assignment
- **Jago App** bugs → Auto-assigned to **Janaka Jati Lasmana**
- **Jagoan App** bugs → Auto-assigned to **Santo Malau**
- Fetches and caches all Notion users on startup
- Maps assignee names to Notion user IDs automatically

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

## 🚀 Infrastructure Deployment

### Option 1: VPS/Server (Recommended for Production)

#### 1.1 Single Binary Deployment

```bash
# Build for Linux (from local machine)
GOOS=linux GOARCH=amd64 go build -o bug-bot ./cmd/bot

# Copy to server
scp bug-bot user@server:/opt/bug-bot/
scp .env user@server:/opt/bug-bot/

# Run on server
ssh user@server
cd /opt/bug-bot
./bug-bot
```

#### 1.2 Systemd Service (Auto-restart on failure)

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
StandardOutput=append:/var/log/bug-bot/stdout.log
StandardError=append:/var/log/bug-bot/stderr.log

# Environment variables (or use EnvironmentFile=/opt/bug-bot/.env)
Environment="SLACK_BOT_TOKEN=xoxb-..."
Environment="NOTION_API_KEY=secret_..."
Environment="OPENAI_API_KEY=sk-..."

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
# Create log directory
sudo mkdir -p /var/log/bug-bot
sudo chown bugbot:bugbot /var/log/bug-bot

# Enable and start service
sudo systemctl enable bug-bot
sudo systemctl start bug-bot
sudo systemctl status bug-bot

# View logs
sudo journalctl -u bug-bot -f
```

#### 1.3 Docker on VPS

```bash
# Build image
docker build -t bug-bot:latest .

# Run container with auto-restart
docker run -d \
  --name bug-bot \
  --restart unless-stopped \
  --env-file .env \
  -v $(pwd)/logs:/root/logs \
  bug-bot:latest

# View logs
docker logs -f bug-bot

# Update deployment
docker stop bug-bot
docker rm bug-bot
docker build -t bug-bot:latest .
docker run -d --name bug-bot --restart unless-stopped --env-file .env -v $(pwd)/logs:/root/logs bug-bot:latest
```

### Option 2: Cloud Platforms (PaaS)

#### 2.1 Railway.app (Easiest)

**Pros:** Free tier, auto-deploy from GitHub, simple setup
**Cons:** Limited free hours

```bash
# Install Railway CLI
npm install -g @railway/cli

# Login and deploy
railway login
railway init
railway up

# Set environment variables
railway variables set SLACK_BOT_TOKEN=xoxb-...
railway variables set NOTION_API_KEY=secret_...
railway variables set OPENAI_API_KEY=sk-...

# View logs
railway logs
```

Or use GitHub integration:
1. Connect GitHub repo to Railway
2. Set environment variables in Railway dashboard
3. Auto-deploys on every push to main

#### 2.2 Fly.io (Recommended for Production)

**Pros:** Global edge deployment, generous free tier, persistent volumes
**Cons:** Slightly more complex setup

```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Login
fly auth login

# Initialize (creates fly.toml)
fly launch --name bug-bot --region sin

# Set secrets
fly secrets set SLACK_BOT_TOKEN=xoxb-...
fly secrets set NOTION_API_KEY=secret_...
fly secrets set OPENAI_API_KEY=sk-...

# Deploy
fly deploy

# View logs
fly logs

# Scale (if needed)
fly scale count 1
fly scale vm shared-cpu-1x

# SSH into instance
fly ssh console
```

**fly.toml** example:
```toml
app = "bug-bot"
primary_region = "sin"

[build]
  builder = "paketobuildpacks/builder:base"

[env]
  PORT = "8080"

[[services]]
  internal_port = 8080
  protocol = "tcp"

  [[services.ports]]
    port = 80
    handlers = ["http"]

  [[services.ports]]
    port = 443
    handlers = ["tls", "http"]
```

#### 2.3 Google Cloud Run (Serverless)

**Pros:** Auto-scaling, pay-per-use, managed infrastructure
**Cons:** Cold starts, more expensive for always-on services

```bash
# Build and push to Google Container Registry
gcloud builds submit --tag gcr.io/PROJECT_ID/bug-bot

# Deploy to Cloud Run
gcloud run deploy bug-bot \
  --image gcr.io/PROJECT_ID/bug-bot \
  --platform managed \
  --region asia-southeast1 \
  --allow-unauthenticated \
  --set-env-vars SLACK_BOT_TOKEN=xoxb-...,NOTION_API_KEY=secret_...,OPENAI_API_KEY=sk-... \
  --memory 512Mi \
  --cpu 1 \
  --min-instances 1 \
  --max-instances 1

# View logs
gcloud run logs tail bug-bot
```

#### 2.4 AWS ECS/Fargate

**Pros:** Full AWS ecosystem integration, highly scalable
**Cons:** More complex, higher cost

```bash
# Build and push to ECR
aws ecr create-repository --repository-name bug-bot
docker build -t bug-bot .
docker tag bug-bot:latest AWS_ACCOUNT.dkr.ecr.REGION.amazonaws.com/bug-bot:latest
aws ecr get-login-password --region REGION | docker login --username AWS --password-stdin AWS_ACCOUNT.dkr.ecr.REGION.amazonaws.com
docker push AWS_ACCOUNT.dkr.ecr.REGION.amazonaws.com/bug-bot:latest

# Create ECS task definition and service via AWS Console or CLI
# Set environment variables in task definition
```

#### 2.5 DigitalOcean App Platform

**Pros:** Simple, affordable, good for small teams
**Cons:** Less flexible than VPS

```bash
# Via web UI:
1. Connect GitHub repo
2. Select branch (main)
3. Set environment variables
4. Deploy

# Or use doctl CLI
doctl apps create --spec app.yaml
```

**app.yaml** example:
```yaml
name: bug-bot
services:
- name: bot
  github:
    repo: rizkajuliant20/bug_bot_poc
    branch: main
  envs:
  - key: SLACK_BOT_TOKEN
    value: xoxb-...
  - key: NOTION_API_KEY
    value: secret_...
  - key: OPENAI_API_KEY
    value: sk-...
  instance_count: 1
  instance_size_slug: basic-xxs
```

### Option 3: Kubernetes (Enterprise)

For large-scale deployments:

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bug-bot
spec:
  replicas: 1
  selector:
    matchLabels:
      app: bug-bot
  template:
    metadata:
      labels:
        app: bug-bot
    spec:
      containers:
      - name: bug-bot
        image: bug-bot:latest
        env:
        - name: SLACK_BOT_TOKEN
          valueFrom:
            secretKeyRef:
              name: bug-bot-secrets
              key: slack-bot-token
        - name: NOTION_API_KEY
          valueFrom:
            secretKeyRef:
              name: bug-bot-secrets
              key: notion-api-key
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: bug-bot-secrets
              key: openai-api-key
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
```

Deploy:
```bash
kubectl apply -f deployment.yaml
kubectl apply -f secrets.yaml
```

### Deployment Comparison

| Platform | Cost | Ease | Scalability | Best For |
|----------|------|------|-------------|----------|
| VPS + Systemd | $5-20/mo | Medium | Manual | Production, full control |
| Railway | Free-$20/mo | Easy | Auto | Quick start, small teams |
| Fly.io | Free-$10/mo | Medium | Auto | Production, global edge |
| Cloud Run | Pay-per-use | Medium | Auto | Serverless, variable load |
| DigitalOcean | $5-12/mo | Easy | Manual | Simple production |
| Kubernetes | $50+/mo | Hard | High | Enterprise, multi-service |

### Monitoring & Maintenance

```bash
# Health check endpoint (add to main.go if needed)
curl http://your-server:3000/health

# View logs
# VPS: tail -f /var/log/bug-bot/stdout.log
# Railway: railway logs
# Fly.io: fly logs
# Cloud Run: gcloud run logs tail bug-bot

# Restart service
# VPS: sudo systemctl restart bug-bot
# Railway: railway restart
# Fly.io: fly apps restart bug-bot
# Cloud Run: gcloud run services update bug-bot --image gcr.io/PROJECT_ID/bug-bot

# Update deployment
git push origin main  # Auto-deploys on Railway, Fly.io (with GitHub Actions)
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
