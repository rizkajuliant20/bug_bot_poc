# Slack Bug Bot with AI Diagnosis

A Slack bot that automatically creates bug tickets in Notion with AI-powered diagnosis. When users report bugs in Slack threads, the bot analyzes the issue using OpenAI, provides intelligent diagnosis, and creates a structured ticket in your Notion database.

## Features

- 🤖 **AI-Powered Diagnosis**: Uses OpenAI GPT-4 to analyze bug reports and provide:
  - Severity assessment (Critical/High/Medium/Low)
  - Category classification (Backend/Frontend/Database/API/etc.)
  - Root cause analysis
  - Suggested fixes
  - Affected components
  - Priority level (P0-P3)

- 💬 **Slack Integration**: 
  - Monitors messages containing bug-related keywords
  - Responds to @mentions
  - Supports `/bug` slash command
  - Captures full thread context
  - Provides real-time status updates with reactions

- 📝 **Notion Integration**:
  - Creates structured bug tickets automatically
  - Includes AI diagnosis and recommendations
  - Links back to Slack thread
  - Captures thread context and reporter info

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

#### Method 1: Automatic Detection
Simply mention keywords in any message:
```
"We have a bug in the login page - users can't authenticate"
"There's an error when submitting the form"
"The API is returning 500 errors"
```

The bot automatically detects messages containing: `bug`, `issue`, `error`, `problem`

#### Method 2: @Mention
```
@BugBot There's a critical bug in the payment processing
```

#### Method 3: Slash Command
```
/bug Users are experiencing timeout errors on checkout
```

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
│   └── services/
│       ├── aiService.js         # OpenAI integration
│       ├── notionService.js     # Notion API integration
│       └── slackService.js      # Slack helper functions
├── .env                         # Environment variables (create this)
├── .env.example                 # Environment template
├── .gitignore
├── package.json
└── README.md
```

## Example Output

When a bug is reported, the bot creates a Notion ticket like this:

**Title**: `[HIGH] Login authentication fails with 401 error`

**Properties**:
- Status: To Do
- Priority: P1
- Severity: High
- Category: Backend
- Reporter: John Doe

**Content**:
- Bug Description
- 🤖 AI Diagnosis with root cause analysis
- 💡 Suggested fix
- Thread context from Slack

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
