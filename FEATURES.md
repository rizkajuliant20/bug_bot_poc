# Bug Bot Features

## ✅ Completed Features

### 1. **Multi-Issue Detection with Interactive Buttons**

Bot automatically analyzes thread + all replies to detect multiple distinct issues.

**Flow:**
```
User adds 🐞 emoji to message
  ↓
Bot analyzes thread + all comments
  ↓
AI detects: Single issue or Multiple issues?
  ↓
┌─────────────────┬──────────────────────┐
│ Single Issue    │ Multiple Issues      │
├─────────────────┼──────────────────────┤
│ Auto-create     │ Show interactive     │
│ ticket          │ buttons for user     │
│                 │ to select which      │
│                 │ issues to create     │
└─────────────────┴──────────────────────┘
```

**Interactive Buttons:**
- ✅ **Create Issue 1, 2, 3...** - Create specific issue ticket
- ✅ **Create All Issues** - Create all tickets at once
- ❌ **Cancel** - Cancel ticket creation

**Example Output:**
```
🔍 Detected 3 separate issues in this thread:

Issue 1: Spin feature tidak bisa diklik
└ Customer tidak bisa menggunakan fitur spin setelah pembelian
└ Severity: high | Category: Frontend

Issue 2: Payment gagal tapi saldo terpotong
└ Transaksi gagal namun saldo customer tetap berkurang
└ Severity: critical | Category: Backend

Issue 3: Notifikasi tidak muncul
└ Push notification tidak terkirim ke user
└ Severity: medium | Category: Backend

Which issues would you like to create tickets for?

[✅ Create Issue 1] [✅ Create Issue 2] [✅ Create Issue 3]
[✅ Create All Issues] [❌ Cancel]
```

**After Button Click:**
```
✅ Created 3/3 bug tickets in Notion

• Issue 1 Ticket (clickable link)
• Issue 2 Ticket (clickable link)
• Issue 3 Ticket (clickable link)

[📝 View Issue 1] [📝 View Issue 2] [📝 View Issue 3]
```

### 2. **Smart Emoji Status Management**

- ✅ Auto-remove old ❌ or ✅ emojis when re-triggering
- 👀 Processing emoji auto-removed even on error (using defer)
- Allows re-trigger by adding 🐞 emoji again

### 3. **AI-Powered Bug Analysis**

- **Diagnosis**: Severity, category, priority, root cause
- **Title Generation**: Smart app detection (Jago App, Jagoan App, Depot Portal, Service)
- **Thread Summarization**: Condenses long discussions
- **Multi-Issue Detection**: Identifies distinct problems in conversations

### 4. **Comprehensive Logging**

- Daily log rotation
- App-specific log files (Jago App, Jagoan App, Depot Portal, Service)
- Separate error logs
- All logs in `logs/` directory

### 5. **Notion Integration**

- Auto-create structured tickets with full context
- Includes: Title, Description, Diagnosis, Reporter, Slack Thread URL, Thread Summary
- Properties: Status, Severity, Category, Priority, Platform, Team
- Polling for manually created bugs (every 2 minutes)

### 6. **Slack Integration**

- **Triggers**: 
  - 🐞 Emoji reaction
  - @mention with bug keywords
  - `/bug` slash command
- **Responses**:
  - Interactive buttons for multi-issue selection
  - Clickable Notion ticket links
  - Status emojis (👀, ✅, ❌)
  - Thread replies with full context

### 7. **Deployment Ready**

- Single binary (9.8MB)
- No runtime dependencies
- Docker support
- Systemd service file
- Multiple cloud platform guides (Railway, Fly.io, GCP, AWS)

## 🎯 Use Cases

### Single Issue
```
User: "Bug di Jago App, spin tidak bisa diklik setelah beli paket"
Bot: 
  - Analyzes with AI
  - Creates Notion ticket
  - Replies with ticket link
```

### Multiple Issues
```
User: "Ada beberapa masalah:
1. Spin tidak bisa diklik
2. Payment gagal tapi saldo terpotong
3. Notifikasi tidak muncul"

Bot:
  - Detects 3 separate issues
  - Shows interactive buttons
  - User selects which to create
  - Creates selected tickets
  - Replies with all ticket links
```

### Re-trigger
```
User: Adds 🐞 emoji again to same message
Bot:
  - Removes old ✅ or ❌
  - Re-analyzes
  - Creates new ticket or shows buttons
```

## 🔧 Technical Implementation

### Multi-Issue Detection
- Uses OpenAI GPT-3.5 with JSON mode
- Analyzes original message + all thread replies
- Returns structured data: issue count, titles, descriptions, severity, category
- Threshold: Shows buttons if >1 issue detected

### Button Interactions
- In-memory store for issue data (thread-scoped)
- Button actions: `create_issue_0`, `create_issue_1`, `create_all_issues`, `cancel_issues`
- Auto-cleanup after button click
- Supports partial creation (some succeed, some fail)

### Slack Responses
- Rich formatting with Slack blocks
- Clickable buttons with URLs
- Markdown support for formatting
- Thread-based replies (keeps context)

### Error Handling
- Graceful fallback to single issue if multi-detection fails
- Partial success handling (create what works, report failures)
- Emoji cleanup on all paths (success, error, cancel)
- User-friendly error messages

## 📊 Performance

- Multi-issue detection: ~3-5 seconds (OpenAI API call)
- Single ticket creation: ~5-7 seconds (AI diagnosis + Notion API)
- Multiple ticket creation: ~5-7 seconds per ticket (parallel possible)
- Memory usage: ~30MB (Go runtime + in-memory store)

## 🚀 Future Enhancements

Potential improvements:
- [ ] Persistent storage for issue data (Redis/DB)
- [ ] Edit/update existing tickets from Slack
- [ ] Bulk operations (close multiple tickets)
- [ ] Custom templates per app/team
- [ ] Analytics dashboard
- [ ] Slack shortcuts for quick actions
