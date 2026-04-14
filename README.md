# Slack Bug Bot with AI Diagnosis

A Slack bot that automatically creates bug tickets in Notion with AI-powered diagnosis. When users report bugs in Slack threads, the bot analyzes the issue using OpenAI, provides intelligent diagnosis, and creates a structured ticket in your Notion database.

## Features

- 🤖 **AI-Powered Diagnosis**: Uses OpenAI GPT-3.5 to analyze bug reports and provide:
  - Severity assessment (Critical/High/Medium/Low)
  - Category classification (Backend/Frontend/Database/API/etc.)
  - Root cause analysis
  - Suggested fixes
  - Affected components
  - Priority level (P0-P3)
  - Platform detection (iOS/Android/Web/Backend)
  - Team assignment

- 🏷️ **Smart App Detection**: Automatically detects and tags bugs by app:
  - **Jago App** - Detects "jago app" mentions in bug description or thread comments
  - **Jagoan App** - Detects "jagoan app" mentions or iOS/Android platform
  - **Depot Portal** - Detects "depot portal" or "depot" mentions
  - **Service** - Detects "service", "backend", or "api" mentions
  - Creates separate log files per app for better tracking

- 💬 **Slack Integration**: 
  - React with 🐞/🐛 emoji on any message to create bug ticket
  - Responds to @mentions with bug keywords
  - Supports `/bug` slash command
  - Captures full thread context and comments
  - Provides real-time status updates with reactions
  - Sends notifications to bug tracking channel

- 📝 **Notion Integration**:
  - Creates structured bug tickets automatically
  - Includes AI diagnosis and recommendations
  - Links back to Slack thread
  - Captures thread context and reporter info
  - Polls Notion for manually created bugs (every 2 minutes)
  - Skips automation-created bugs to avoid duplicates

- 📊 **Comprehensive Logging**:
  - Daily log files in `logs/` directory
  - Separate logs per app (jago-app-YYYY-MM-DD.log, etc.)
  - Error logs separated (error-YYYY-MM-DD.log)
  - Automatic cleanup of logs older than 7 days
  - Tracks all triggers, flows, API calls, and errors with timestamps

## Prerequisites

- Node.js 18+ 
- Slack workspace with admin access
- Notion workspace with integration access
- OpenAI API key

## Setup

### 1. Clone and Install

```bash
cd /Users/rizkajuliant20/Documents/BotBugWithAI
npm install
```

### 2. Slack App Configuration

1. Go to [api.slack.com/apps](https://api.slack.com/apps)
2. Click **Create New App** → **From scratch**
3. Name it "Bug Bot" and select your workspace

#### Enable Socket Mode:
- Go to **Socket Mode** in the sidebar
- Enable Socket Mode
- Generate an App-Level Token with `connections:write` scope
- Save the token (starts with `xapp-`)

#### Bot Token Scopes:
Go to **OAuth & Permissions** and add these scopes:
- `app_mentions:read`
- `chat:write`
- `channels:history`
- `channels:read`
- `groups:history`
- `groups:read`
- `im:history`
- `mpim:history`
- `reactions:write`
- `users:read`

#### Event Subscriptions:
Go to **Event Subscriptions** and subscribe to:
- `app_mention`
- `reaction_added`
- `message.channels`
- `message.groups`
- `message.im`
- `message.mpim`

#### Slash Commands:
Go to **Slash Commands** and create:
- Command: `/bug`
- Description: "Create a bug ticket with AI diagnosis"
- Usage Hint: "[bug description]"

#### Install to Workspace:
- Go to **Install App**
- Click **Install to Workspace**
- Copy the **Bot User OAuth Token** (starts with `xoxb-`)

### 3. Notion Database Setup

1. Create a new Notion database with these properties:
   - **Title** (Title)
   - **Status** (Select): Options: To Do, In Progress, Done
   - **Priority** (Select): Options: P0, P1, P2, P3
   - **Severity** (Select): Options: Critical, High, Medium, Low
   - **Category** (Select): Options: Backend, Frontend, Database, API, UI/UX, Performance, Security, Other
   - **Reporter** (Text)
   - **Slack Thread** (URL)

2. Create a Notion Integration:
   - Go to [notion.so/my-integrations](https://www.notion.so/my-integrations)
   - Click **New integration**
   - Name it "Bug Bot"
   - Copy the **Internal Integration Token** (starts with `secret_`)

3. Share your database with the integration:
   - Open your Notion database
   - Click **Share** → Add your integration

4. Get your Database ID:
   - Open the database in Notion
   - Copy the ID from the URL: `notion.so/[workspace]/[DATABASE_ID]?v=...`

### 4. OpenAI API Key

1. Go to [platform.openai.com/api-keys](https://platform.openai.com/api-keys)
2. Create a new API key
3. Copy the key (starts with `sk-`)

### 5. Environment Configuration

Create a `.env` file:

```bash
cp .env.example .env
```

Edit `.env` with your credentials:

```env
SLACK_BOT_TOKEN=xoxb-your-bot-token
SLACK_SIGNING_SECRET=your-signing-secret
SLACK_APP_TOKEN=xapp-your-app-token
SLACK_BUG_TRACKING_CHANNEL=C0AT3CEB5ED  # Optional: Channel ID for bug notifications

NOTION_API_KEY=secret_your-notion-integration-token
NOTION_DATABASE_ID=your-database-id

OPENAI_API_KEY=sk-your-openai-api-key

PORT=3000
```

## Usage

### Start the Bot

```bash
npm start
```

For development with auto-reload:

```bash
npm run dev
```

### Using the Bot in Slack

#### Method 1: React with Bug Emoji
React to any message with 🐞 (lady_beetle), 🐛 (bug), or 🪲 (beetle) emoji:
```
User: "Login is broken on the app"
[React with 🐞 emoji] → Bot creates ticket
```

#### Method 2: @Mention
```
@BugBot There's a critical bug in the payment processing on Jago App
```

#### Method 3: Slash Command
```
/bug Users are experiencing timeout errors on checkout in Jagoan App
```

#### App Detection Examples
The bot automatically detects which app the bug is for:

```
"This bug happens in Jago App" → Tagged as [Bug][Jago App]
"Jagoan app crashes on startup" → Tagged as [Bug][Jagoan App]  
"Depot portal login issue" → Tagged as [Bug][Depot Portal]
"Backend service is down" → Tagged as [Bug][Service]
```

You can mention the app name in:
- The original bug report
- Any reply/comment in the thread
- Thread discussions

### What Happens Next

1. 👀 Bot adds "eyes" reaction to show it's processing
2. 🤖 AI analyzes the bug and thread context
3. 📝 Creates a Notion ticket with diagnosis
4. ✅ Replies in thread with:
   - Bug ticket summary
   - AI diagnosis and root cause
   - Suggested fix
   - Link to Notion ticket

## Project Structure

```
BotBugWithAI/
├── src/
│   ├── index.js                 # Main bot entry point
│   ├── config.js                # Configuration and validation
│   ├── handlers/
│   │   └── bugHandler.js        # Bug report processing logic
│   ├── services/
│   │   ├── aiService.js         # OpenAI integration & app detection
│   │   ├── notionService.js     # Notion API integration
│   │   ├── notionPolling.js     # Notion polling for manual bugs
│   │   └── slackService.js      # Slack helper functions
│   └── utils/
│       └── logger.js            # Logging utility with file output
├── logs/                        # Log files (auto-created)
│   ├── app-YYYY-MM-DD.log       # Main application logs
│   ├── jago-app-YYYY-MM-DD.log  # Jago App specific logs
│   ├── jagoan-app-YYYY-MM-DD.log # Jagoan App specific logs
│   ├── depot-portal-YYYY-MM-DD.log # Depot Portal specific logs
│   ├── service-YYYY-MM-DD.log   # Service specific logs
│   └── error-YYYY-MM-DD.log     # Error logs
├── .env                         # Environment variables (create this)
├── .env.example                 # Environment template
├── .gitignore
├── .notion-tracking.json        # Tracked Notion bugs (auto-created)
├── package.json
└── README.md
```

## Example Output

When a bug is reported, the bot creates a Notion ticket like this:

**Title**: `[Bug][Jago App] Login authentication fails with 401 error`

**Properties**:
- Status: Not started
- Priority: P1
- Severity: High
- Category: Backend
- Platform: iOS, Android
- Team: Eng
- Reporter: John Doe
- Slack Thread: [Link to thread]

**Content**:
- **Bug Description**: Original bug report
- **Precondition**: Conditions before bug occurs
- **Steps to Reproduce**: Numbered steps
- **Actual Result**: What actually happens
- **Expected Result**: What should happen
- **🔍 QA Diagnosis**: AI-generated root cause and suggested fix
- **💬 Thread Summary**: Summary of thread discussion (if any)

## Logging & Monitoring

### Log Files
All logs are stored in the `logs/` directory with daily rotation:

- **`app-YYYY-MM-DD.log`**: All application logs
- **`jago-app-YYYY-MM-DD.log`**: Jago App specific bugs
- **`jagoan-app-YYYY-MM-DD.log`**: Jagoan App specific bugs
- **`depot-portal-YYYY-MM-DD.log`**: Depot Portal specific bugs
- **`service-YYYY-MM-DD.log`**: Service/Backend specific bugs
- **`error-YYYY-MM-DD.log`**: All errors across all apps

### Log Format
```
[2026-04-14T09:15:47.802Z] [SUCCESS] ✅ Notion ticket created {"notionUrl":"https://notion.so/...","pageId":"...","appName":"Jago App"}
```

### Log Retention
- Logs older than 7 days are automatically deleted
- Each log entry includes timestamp, level, message, and metadata
- Error logs include full stack traces

### Viewing Logs
```bash
# View today's main log
tail -f logs/app-$(date +%Y-%m-%d).log

# View Jago App logs
tail -f logs/jago-app-$(date +%Y-%m-%d).log

# View errors only
tail -f logs/error-$(date +%Y-%m-%d).log

# Search for specific bug
grep "Bug ticket created" logs/app-*.log
```

## Advanced Features

### Notion Polling
The bot polls your Notion database every 2 minutes to detect manually created bugs:
- Only notifies for bugs created manually (without Slack Thread URL)
- Skips automation-created bugs to avoid duplicate notifications
- Sends notifications to configured bug tracking channel
- Tracks processed bugs in `.notion-tracking.json`

### Bug Tracking Channel
Configure `SLACK_BUG_TRACKING_CHANNEL` in `.env` to receive notifications for:
- All automation-created bugs (from Slack)
- Manually created bugs in Notion
- Includes bug details, severity, priority, and links

### Thread Context Analysis
The bot analyzes the entire Slack thread:
- Original bug report message
- All replies and comments in the thread
- Detects app name from any message in thread
- Summarizes discussion for Notion ticket
- Uses context for better AI diagnosis

## Troubleshooting

### Bot doesn't respond
- Check that Socket Mode is enabled
- Verify the app is installed to your workspace
- Ensure event subscriptions are configured
- Check console logs for errors

### Notion ticket creation fails
- Verify database properties match exactly
- Ensure integration has access to the database
- Check that Database ID is correct

### AI diagnosis not working
- Verify OpenAI API key is valid
- Check you have sufficient API credits
- Review console logs for API errors

## Development

The bot uses:
- **@slack/bolt** for Slack integration
- **@notionhq/client** for Notion API
- **openai** for AI diagnosis
- **dotenv** for environment management

## License

MIT
