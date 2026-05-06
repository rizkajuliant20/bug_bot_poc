package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rizkajuliant20/bug-bot/pkg/logger"
	openai "github.com/sashabaranov/go-openai"
	"github.com/slack-go/slack"
)

type OpenAIService struct {
	client *openai.Client
	logger *logger.Logger
}

type BugDiagnosis struct {
	Severity           string   `json:"severity"`
	Category           string   `json:"category"`
	Priority           string   `json:"priority"`
	Platform           []string `json:"platform"`
	Team               string   `json:"team"`
	Precondition       string   `json:"precondition"`
	StepsToReproduce   string   `json:"stepsToReproduce"`
	ActualResult       string   `json:"actualResult"`
	ExpectedResult     string   `json:"expectedResult"`
	RootCause          string   `json:"rootCause"`
	SuggestedFix       string   `json:"suggestedFix"`
	AffectedComponents []string `json:"affectedComponents"`
	Tags               []string `json:"tags"`
}

type BugSummaryResult struct {
	Title   string
	AppName string
}

func NewOpenAIService(apiKey string, log *logger.Logger) *OpenAIService {
	client := openai.NewClient(apiKey)
	return &OpenAIService{
		client: client,
		logger: log,
	}
}

func (s *OpenAIService) DiagnoseBug(bugDescription string, threadMessages []slack.Message) (*BugDiagnosis, error) {
	s.logger.Flow("AI_SERVICE", "Starting bug diagnosis", map[string]interface{}{
		"descriptionLength": len(bugDescription),
		"threadMessages":    len(threadMessages),
	})

	conversation := formatThreadMessages(threadMessages)

	prompt := fmt.Sprintf(`Analyze this bug report and provide a structured diagnosis in JSON format.

Bug Description: %s

Thread Discussion:
%s

Provide your analysis in this exact JSON structure with DETAILED, CONTEXT-AWARE information:
{
  "severity": "critical|high|medium|low",
  "category": "Backend|Frontend|Database|API|UI/UX|Performance|Security|Other",
  "priority": "Low|Medium|High",
  "platform": ["Android", "iOS", "Website", "Backend"],
  "team": "Eng|Data|Design|Product",
  "precondition": "DETAILED conditions - include: user state, data state, environment, time/date if relevant",
  "stepsToReproduce": "SPECIFIC numbered steps with actual values/actions - e.g., '1. Customer selects Jago Party on 30 May 2026, 13:30-14:30\n2. Customer tries to book but unable to do so\n3. Customer uninstalls and reinstalls the app'",
  "actualResult": "PRECISE description of what happens - include error messages, UI state, data shown",
  "expectedResult": "CLEAR expected behavior - what should work correctly",
  "rootCause": "RCA as structured text: [Confirmed/Suspected/Unknown] Category: [FE/BE/API/etc]. Analysis: [details]. User Impact: [what user sees]. Mitigation: [if any]. Investigation needed: [if unknown/suspected]",
  "suggestedFix": "ACTIONABLE fix with technical details - what to investigate/fix and where. Include: Fix recommendation, Prevention idea",
  "affectedComponents": ["component1", "component2"],
  "tags": ["Bug", "Jago App"]
}

SEVERITY RULES (Impact Assessment):
CRITICAL: Crash/force close (reproducible), Data loss/corruption, Security/privacy issue, Payment/checkout/settlement gagal total
HIGH: Core flow bermasalah (not crash), Banyak user terdampak, Salah hitung yang berdampak transaksi/stock/settlement
MEDIUM: Flow sekunder rusak, Edge case, Dampak terbatas dengan workaround reasonable
LOW: Cosmetic/polish, tidak mempengaruhi hasil utama

ROOT CAUSE ANALYSIS (RCA) - CRITICAL RULES:
AI MUST NOT claim root cause as fact without explicit evidence!

Use "Confirmed Root Cause" ONLY if:
- Slack thread explicitly states "root cause confirmed..."
- Clear evidence from logs/investigation mentioned

Use "Suspected Root Cause (Hypothesis)" if:
- Strong signals but not confirmed
- Format: "Hypothesis: [reason]. What to confirm: [investigation steps]"

Use "Unknown (Needs Investigation)" if:
- Minimal info available
- Format: "Root cause: TBD. Investigation steps: check logs at [where], check payload, check feature flag, check data condition"

Root Cause Categories (pick one or more for rootCause field):
- FE state/logic, BE validation/business rule, API contract mismatch, Data issue/migration
- Feature flag/config, Race condition/concurrency, Network/timeout/retry
- 3rd party dependency, Permission/auth, UI layout/copy

RCA FORMAT (write as structured text in rootCause field):
"[Confirmed/Suspected/Unknown] Category: [FE/BE/API/etc]. Analysis: [technical details]. User Impact: [what user sees]. Mitigation: [workaround if any]. Investigation: [steps needed if not confirmed]. Prevention: [test idea]."

Example: "Suspected. Category: BE validation/business rule. Analysis: Possible issue with location distance verification or manual processing due to map discrepancy. User Impact: Customer unable to book Jago Party, sees 'location too far' error. Mitigation: Manual processing by team. Investigation: Check location validation logic, verify distance calculation on backend. Prevention: Add test for edge case locations."

ANALYSIS GUIDELINES - DETAILED & FEATURE-AWARE:

For JAGO APP bugs:
- Precondition: order type (on-demand/scheduled/bulk), location, timeslot, cart items, device, phone number
- Steps: exact flow with actual values (nearby map → select cart → items → checkout → payment)
- Root Cause: Consider location validation, distance calculation, timeslot availability, payment gateway, stock availability
- Suggested Fix: Reference APIs (order creation, location service, payment service, timeslot validation)

For JAGOAN APP bugs:
- Precondition: Jagoan status, shift state, stock level, order queue
- Steps: operational flow (login → dashboard → checkout/settlement → order handling)
- Root Cause: Consider auth flow, order assignment, stock sync, settlement calculation, offline/online sync
- Suggested Fix: Reference modules (auth, order management, checkout flow, settlement flow, sync service)

For DEPOT PORTAL bugs:
- Precondition: admin role, depot state, pending orders count, Jagoan availability
- Steps: admin workflow (view orders → assign Jagoan → approve checkout → approve settlement)
- Root Cause: Consider assignment algorithm, approval workflow, data sync, equipment tracking
- Suggested Fix: Reference modules (order management, assignment logic, approval system, equipment tracking)

MAKE IT SPECIFIC: Use actual data (dates, times, names, error messages, phone numbers, order IDs, build versions)

IMPORTANT RULES:
- Priority must be exactly: "Low", "Medium", or "High" (case-sensitive)
- Platform options: "Android", "iOS", "Website", "Backend"
- Team must be exactly: "Eng", "Data", "Design", or "Product"
- Tags MUST contain EXACTLY 2 items: "Bug" (mandatory) + ONE app name
- App name options: "Jago App", "Jagoan App", "Depot Portal", or "Service"
- NEVER include other tags like "Tech Debt", "Design System", "UI/UX", "Website", etc.
- Example valid tags: ["Bug", "Jago App"] or ["Bug", "Service"]

PRIORITY RULES (Urgency = P0/P1/P2/P3 mapped to High/Medium/Low):

FEATURE IMPACT MATRIX (Business × User Journey × Operational):
Use this to assess priority based on which feature is affected:

JAGO APP (Customer) - P0/P1 Features (High Priority):
- On-demand delivery: H/H/H (direct transaction, blocker if fail, ripple to depot/jagoan)
- Scheduled order: H/H/H (conversion driver, main fallback, complex fulfillment)
- Bulk order/Jago Party: H/MH/H (high AOV, critical for bulk segment, complex ops)
- Nearby map: H/H/M (discovery→transaction, main entry point, data dependent)
- Payment/Checkout/Transaction: H/H/H (revenue blocker)

JAGO APP (Customer) - P2 Features (Medium Priority):
- Earn Jago Energy: MH/M/M (retention driver, not first-order blocker, needs accuracy)
- Redeem Energy: M/M/M (perceived value, post-purchase, needs guardrails)
- Scan QR Energy: M/M/H (offline bridge, important for offline users, ops-heavy/risky)

JAGOAN APP (Operator) - P0/P1 Features (High Priority):
- Authentication/Login: H/H/H (cannot work without login, basic blocker, access control)
- Scheduled orders handling: H/H/H (fulfillment SLA, core execution, affects tracking)
- Manual orders (offline POS): H/H/H (main sales channel, core daily transaction, stock/payment/loyalty)
- Checkout (request stock): H/H/H (no stock = no sales, start selling, depot dependency)
- Settlement (EOD): H/H/H (finance/recon/close shift, mandatory EOD, high risk)
- Dashboard: M/M/H (monitoring, control center, gateway to Checkout/Settlement)

DEPOT PORTAL (Admin) - P0/P1 Features (High Priority):
- Scheduled/Urgent Orders page: H/H/H (order can fail, main work queue, choke point)
- Assign/Reassign order: H/H/H (determines fulfilled vs fail, core action, manual escalation risk)
- Jago Party page: H/MH/H (high value events, used during events, complex coordination)
- Checkout approval+code: H/H/H (stock availability, admin routine, physical dependency/bottleneck)
- Settlement approval: H/H/H (finance/recon, mandatory EOD, high risk if stuck)
- Jagoan management: H/M/H (supply side capacity, HR/ops workflow, day-to-day core)

DEPOT PORTAL (Admin) - P2 Features (Medium Priority):
- Omni-channel input: MH/M/H (revenue recording, not core assignment, manual daily process)
- Equipment tracking: M/M/H (uptime/indirect revenue, not direct assignment, maintenance/allocation)
- Rank progression: M/LM/M (incentive driver, not order blocker, governance/performance)

HIGH PRIORITY CONDITIONS (P0/P1):
P0 (use "High"): Blocks core functionality/critical path, Crash on main flow, Transaction/order/settlement blocked, Mass impact/security
P1 (use "High"): Severely disrupts main flow, Requires significant manual ops intervention, High frequency/exposure

CRITICAL KEYWORDS - If ANY of these appear, Priority MUST be "High":
- book, booking, pesan (e.g., "book Jago Party", "pesan jago party", "booking gagal")
- order, ordering (e.g., "order tidak bisa", "cannot order")
- delivery, payment, checkout, cart, transaction
- settlement, assignment, crash, gagal total
- auth, login, cannot access

EXAMPLES:
- "unable to book Jago Party" → Priority: High (contains "book")
- "pesan jago party gagal" → Priority: High (contains "pesan")
- "customer cannot order" → Priority: High (contains "order")
- "location too far error when booking" → Priority: High (contains "booking")

MEDIUM PRIORITY (P2 - Has Workaround):
- Clear workaround exists and impact is limited
- Edge case or affects small subset of users/ops
- Secondary features: Jago Energy, Redeem, QR scan, Dashboard monitoring, Omni Channel input, Rank tracking
- UX issues affecting flow but not blocking core transactions

LOW PRIORITY (P3 - Cosmetic):
- UI alignment, spacing, colors, icons, labels, copywriting
- Badge display, environment indicators
- Minor visual bugs that don't affect functionality
- Polish items with no business/ops impact

DECISION RULE: Priority can be higher than Severity if there's release/ops urgency. Focus Priority on "urgency to fix", Severity on "impact if happens".

APP DETECTION RULES (CRITICAL):
STEP 1 - Initial detection from bug description:
- If text mentions "jagoan", "aplikasi jagoan", "jagoan app" → Initially "Jagoan App"
- If text mentions "jago", "aplikasi jago", "jago app" (but NOT jagoan) → Initially "Jago App"
- If text mentions "depot", "depot portal" → Initially "Depot Portal"
- If text mentions "service", "backend", "api" → Initially "Service"

STEP 2 - RE-EVALUATE after Root Cause analysis (CRITICAL):
After you write the Root Cause, CHECK:
- Does Root Cause mention "backend", "API", "server", "database", "validation logic", "distance calculation", "location validation"?
- Does Suggested Fix mention "backend", "API", "server logic", "check API"?
If YES → OVERRIDE tags to ["Bug", "Service"] (even if initially detected as Jago App/Jagoan App/Depot Portal)

EXAMPLES:
- Bug: "unable to book Jago Party" + Root Cause: "backend validation logic" → Tags: ["Bug", "Service"] (NOT "Jago App")
- Bug: "settlement error" + Root Cause: "UI state issue" → Tags: ["Bug", "Jagoan App"] (stays as app)
- Bug: "order assignment" + Root Cause: "check location API" → Tags: ["Bug", "Service"] (NOT "Depot Portal")

CRITICAL: Backend/API/Server issues ALWAYS = "Service" tag, regardless of which app user was using

IMPORTANT: Extract information ONLY from the actual bug report and thread discussion. DO NOT make assumptions or use placeholders.
- If precondition is not mentioned → extract from context or leave as brief statement
- If steps are not clear → infer from description but use actual details
- If root cause is unclear → use "Unknown (Needs Investigation)" with specific investigation steps
- NEVER use generic placeholders like "TBD", "To be investigated", "Unknown" without context

Use ONLY actual data from the bug report. Be specific with what you know and honest about what needs investigation.`, bugDescription, conversation)

	startTime := time.Now()
	s.logger.Info("Calling OpenAI API for diagnosis", nil)

	resp, err := s.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a technical bug analyst. Always respond with valid JSON only.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature:    0.3,
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject},
		},
	)

	if err != nil {
		s.logger.Error("Bug diagnosis failed", err, nil)
		return nil, fmt.Errorf("failed to diagnose bug: %w", err)
	}

	duration := time.Since(startTime)

	var diagnosis BugDiagnosis
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &diagnosis); err != nil {
		s.logger.Error("Failed to parse diagnosis JSON", err, nil)
		return nil, fmt.Errorf("failed to parse diagnosis: %w", err)
	}

	// Validate and clean tags - only "Bug" + one app name
	validAppNames := map[string]bool{
		"Jago App":     true,
		"Jagoan App":   true,
		"Depot Portal": true,
		"Service":      true,
	}

	cleanedTags := []string{"Bug"} // Always include Bug
	for _, tag := range diagnosis.Tags {
		if validAppNames[tag] {
			cleanedTags = append(cleanedTags, tag)
			break // Only take first valid app name
		}
	}

	// If no valid app name found, default to "Jago App"
	if len(cleanedTags) == 1 {
		cleanedTags = append(cleanedTags, "Jago App")
	}

	diagnosis.Tags = cleanedTags

	s.logger.Success("Bug diagnosis completed", map[string]interface{}{
		"duration": duration.Milliseconds(),
		"severity": diagnosis.Severity,
		"category": diagnosis.Category,
		"tags":     diagnosis.Tags,
	})

	return &diagnosis, nil
}

func (s *OpenAIService) SummarizeThread(threadMessages []slack.Message) (string, error) {
	if len(threadMessages) <= 1 {
		return "", nil
	}

	s.logger.Flow("AI_SERVICE", "Summarizing thread", map[string]interface{}{
		"messageCount": len(threadMessages),
	})

	conversation := formatThreadMessages(threadMessages)

	prompt := fmt.Sprintf(`Summarize this bug discussion thread concisely. Focus on:
1. Key points discussed
2. Additional context or symptoms mentioned
3. Any attempted solutions or workarounds

Thread:
%s

Provide a concise summary (max 200 words) focusing on technical details.`, conversation)

	startTime := time.Now()
	s.logger.Info("Calling OpenAI API for thread summary", nil)

	resp, err := s.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a technical writer. Summarize bug discussions concisely.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.3,
			MaxTokens:   300,
		},
	)

	if err != nil {
		s.logger.Error("Thread summarization failed", err, nil)
		return "", nil
	}

	duration := time.Since(startTime)
	summary := strings.TrimSpace(resp.Choices[0].Message.Content)

	s.logger.Success("Thread summary generated", map[string]interface{}{
		"duration": duration.Milliseconds(),
	})

	return summary, nil
}

func (s *OpenAIService) GenerateBugSummary(bugDescription string, diagnosis *BugDiagnosis, threadMessages []slack.Message) (*BugSummaryResult, error) {
	s.logger.Flow("AI_SERVICE", "Generating bug title", map[string]interface{}{
		"descriptionLength":  len(bugDescription),
		"threadMessageCount": len(threadMessages),
	})

	// Use app name from diagnosis tags (source of truth)
	appName := "App"
	var detectedAppName string

	// Extract app name from tags
	for _, tag := range diagnosis.Tags {
		if tag == "Jago App" || tag == "Jagoan App" || tag == "Depot Portal" || tag == "Service" {
			appName = tag
			detectedAppName = tag
			s.logger.Info("Using app name from diagnosis tags", map[string]interface{}{"appName": appName})
			break
		}
	}

	// Fallback: if no app name in tags, detect from text
	if appName == "App" {
		allText := strings.ToLower(bugDescription)
		for _, msg := range threadMessages {
			allText += " " + strings.ToLower(msg.Text)
		}

		// Detect app name - check Jagoan first (more specific)
		if strings.Contains(allText, "jagoan") || strings.Contains(allText, "aplikasi jagoan") {
			appName = "Jagoan App"
			detectedAppName = "Jagoan App"
		} else if strings.Contains(allText, "jago app") || strings.Contains(allText, "jagoapp") || strings.Contains(allText, "aplikasi jago") {
			appName = "Jago App"
			detectedAppName = "Jago App"
		} else if strings.Contains(allText, "depot portal") || strings.Contains(allText, "depot") {
			appName = "Depot Portal"
			detectedAppName = "Depot Portal"
		} else if strings.Contains(allText, "service") || strings.Contains(allText, "backend") || strings.Contains(allText, "api") {
			appName = "Service"
			detectedAppName = "Service"
		}
		s.logger.Info("Detected app name from text (fallback)", map[string]interface{}{"appName": appName})
	}

	prompt := fmt.Sprintf(`Create a concise bug ticket title in this exact format:
[Bug][%s] Brief description of the issue

Bug Description: %s
Category: %s

CRITICAL RULES:
1. Extract the EXACT issue from the bug description - DO NOT invent or guess
2. Use ACTUAL keywords from the description (e.g., if it says "create order", use "create order", NOT "login")
3. Keep it brief (max 8 words after app name)
4. Focus on the core technical issue mentioned

Example:
- If bug says "app loading after create order button" → "App loading after create order button"
- If bug says "settlement error" → "Settlement error"
- DO NOT use generic terms like "login" unless explicitly mentioned`, appName, bugDescription, diagnosis.Category)

	startTime := time.Now()
	s.logger.Info("Calling OpenAI API for bug title generation", map[string]interface{}{
		"appName":        appName,
		"bugDescription": bugDescription,
		"category":       diagnosis.Category,
	})

	resp, err := s.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a technical writer. Create clear, structured bug titles following the exact format provided.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.1, // Very low to prevent hallucination
			MaxTokens:   60,
		},
	)

	if err != nil {
		s.logger.Error("Bug title generation failed", err, nil)
		return &BugSummaryResult{
			Title:   fmt.Sprintf("[Bug][App] %s", truncate(bugDescription, 50)),
			AppName: detectedAppName,
		}, nil
	}

	duration := time.Since(startTime)
	title := strings.TrimSpace(resp.Choices[0].Message.Content)

	s.logger.Success("Bug title generated", map[string]interface{}{
		"duration": duration.Milliseconds(),
		"title":    title,
	})

	return &BugSummaryResult{
		Title:   title,
		AppName: detectedAppName,
	}, nil
}

func formatThreadMessages(messages []slack.Message) string {
	var parts []string
	messageNum := 1
	for _, msg := range messages {
		// Skip bot messages to avoid AI confusion
		if msg.BotID != "" || msg.Username == "bug-bot" {
			continue
		}

		// Skip messages containing bot-specific patterns
		lowerText := strings.ToLower(msg.Text)
		botPatterns := []string{
			"ticket created:",
			"error analyzing bug",
			"failed to diagnose bug",
			"failed to parse diagnosis",
			"status code:",
			"does not have access to model",
			"gpt-4o",
			"gpt-3.5",
			"proj_",
			"detected 2 separate issues",
			"detected multiple issues",
			"which issues would you like",
			"create issue",
			"create all issues",
			"view in notion",
		}

		skipMessage := false
		for _, pattern := range botPatterns {
			if strings.Contains(lowerText, pattern) {
				skipMessage = true
				break
			}
		}

		if skipMessage {
			continue
		}

		// Only include actual user messages
		if msg.Text != "" {
			parts = append(parts, fmt.Sprintf("Message %d: %s", messageNum, msg.Text))
			messageNum++
		}
	}
	return strings.Join(parts, "\n")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

type DetectedIssue struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"`
	Category    string   `json:"category"`
	Priority    string   `json:"priority"`
	Platform    []string `json:"platform"`
	Team        string   `json:"team"`
	Tags        []string `json:"tags"`
}

type MultiIssueAnalysis struct {
	IssueCount int             `json:"issueCount"`
	Issues     []DetectedIssue `json:"issues"`
}

func (s *OpenAIService) DetectMultipleIssues(bugDescription string, threadMessages []slack.Message) (*MultiIssueAnalysis, error) {
	s.logger.Flow("AI_SERVICE", "Detecting multiple issues from thread", map[string]interface{}{
		"descriptionLength": len(bugDescription),
		"threadMessages":    len(threadMessages),
	})

	conversation := formatThreadMessages(threadMessages)

	prompt := fmt.Sprintf(`Analyze this bug report and thread discussion to detect if there are MULTIPLE DISTINCT issues mentioned.

Bug Description: %s

Thread Discussion:
%s

CRITICAL: Be consistent and thorough in your analysis.
- If the thread discusses ONE problem with multiple symptoms/aspects → issueCount: 1
- If the thread discusses MULTIPLE SEPARATE problems → issueCount: 2 or more
- Don't split one issue into multiple parts just because it has multiple steps
- Only count as multiple issues if they are truly independent problems

Carefully identify:
1. How many DISTINCT issues/bugs are mentioned?
2. For each issue, provide a DETAILED title and description

DESCRIPTION GUIDELINES - Make each issue description SPECIFIC and DETAILED:
- Include actual data: dates, times, error messages, user IDs, phone numbers, order IDs
- Specify the feature/flow affected (e.g., "Jago Party booking", "Scheduled order assignment", "Checkout approval")
- Include preconditions if mentioned (e.g., "Customer on iOS", "Jagoan during settlement", "Admin assigning order")
- Mention actual vs expected behavior clearly
- Reference specific user actions or system states

Example GOOD descriptions:
- "Customer unable to book Jago Party for 30 May 2026, 13:30-14:30. Error: 'location too far'. Customer tried reinstalling app but issue persists. Device: iOS, Phone: 089663125432"
- "Jagoan cannot complete settlement at end of day. Settlement approval stuck in pending state. Affects depot operations and shift closure."
- "Depot admin unable to reassign order #12345 from Jagoan A to Jagoan B. Assignment button not responding on Depot Portal."

Example BAD descriptions (too vague):
- "Booking not working"
- "Settlement error"
- "Cannot assign order"

Provide your response in this exact JSON format:
{
  "issueCount": <number of distinct issues found>,
  "issues": [
    {
      "title": "Brief title of issue 1",
      "description": "Short description of what the issue is",
      "severity": "critical|high|medium|low",
      "category": "Backend|Frontend|Database|API|UI/UX|Performance|Security|Other",
      "priority": "Low|Medium|High",
      "platform": ["Android", "iOS", "Website", "Backend"],
      "team": "Eng|Data|Design|Product",
      "tags": ["Bug", "Jago App"]
    }
  ]
}

IMPORTANT RULES:
- If only ONE issue is discussed, return issueCount: 1 with that single issue
- If multiple SEPARATE issues are mentioned, list all of them
- Don't split one issue into multiple parts
- Focus on truly distinct problems
- Severity must be lowercase: "critical", "high", "medium", or "low"
- Category options: "Backend", "Frontend", "Database", "API", "UI/UX", "Performance", "Security", "Other"
- Priority must be exactly: "Low", "Medium", or "High" (case-sensitive)
- Platform options: "Android", "iOS", "Website", "Backend"
- Team must be exactly: "Eng", "Data", "Design", or "Product"
- Tags MUST contain EXACTLY 2 items: "Bug" (mandatory) + ONE app name
- App name options: "Jago App", "Jagoan App", "Depot Portal", or "Service"
APP DETECTION (2-STEP PROCESS):
STEP 1 - Initial: "jagoan" → Jagoan App, "jago" → Jago App, "depot" → Depot Portal
STEP 2 - RE-EVALUATE based on category:
- If category is "Backend", "API", "Database" → OVERRIDE tags to ["Bug", "Service"]
- If category is "Frontend", "UI/UX" → Keep app tag (Jago App/Jagoan App/Depot Portal)
- If description mentions "backend", "API", "server", "database", "validation logic" → OVERRIDE to ["Bug", "Service"]

EXAMPLES:
- Issue: "unable to book" + Category: Backend → tags: ["Bug", "Service"] (NOT "Jago App")
- Issue: "settlement error" + Category: Frontend → tags: ["Bug", "Jagoan App"]
- Issue: "map location wrong" + Category: Backend → tags: ["Bug", "Service"] (NOT "Jago App")

CRITICAL: Backend/Database/API category = "Service" tag ALWAYS, regardless of which app user was using

SEVERITY RULES:
CRITICAL: Crash/force close (reproducible), Data loss/corruption, Security/privacy, Payment/checkout/settlement gagal total
HIGH: Core flow bermasalah, Banyak user terdampak, Salah hitung transaksi/stock/settlement
MEDIUM: Flow sekunder rusak, Edge case, Dampak terbatas dengan workaround
LOW: Cosmetic/polish, tidak mempengaruhi hasil utama

PRIORITY RULES (P0/P1/P2/P3 → High/Medium/Low):

FEATURE IMPACT (Business/Journey/Ops):
JAGO APP P0/P1 (High): On-demand delivery (H/H/H), Scheduled order (H/H/H), Bulk/Jago Party (H/MH/H), Nearby map (H/H/M), Payment/Checkout (H/H/H)
JAGO APP P2 (Medium): Earn Energy (MH/M/M), Redeem (M/M/M), QR scan (M/M/H)
JAGOAN APP P0/P1 (High): Auth/Login (H/H/H), Scheduled orders (H/H/H), Manual orders (H/H/H), Checkout (H/H/H), Settlement (H/H/H), Dashboard (M/M/H)
DEPOT PORTAL P0/P1 (High): Orders page (H/H/H), Assign/Reassign (H/H/H), Jago Party (H/MH/H), Checkout approval (H/H/H), Settlement approval (H/H/H), Jagoan mgmt (H/M/H)
DEPOT PORTAL P2 (Medium): Omni-channel (MH/M/H), Equipment (M/M/H), Rank (M/LM/M)

HIGH (P0/P1): Blocks core functionality/critical path, Crash on main flow, Transaction/order/settlement blocked, Mass impact, Manual ops intervention
CRITICAL KEYWORDS - If ANY appear, Priority MUST be "High":
book, booking, pesan, order, ordering, delivery, payment, checkout, cart, transaction, settlement, assignment, crash, gagal total, auth, login
EXAMPLES: "unable to book" → High, "pesan gagal" → High, "cannot order" → High

MEDIUM (P2): Has workaround, Edge case, Small subset, Secondary features

LOW (P3): Cosmetic only

DECISION RULE: Priority = urgency to fix, Severity = impact if happens`, bugDescription, conversation)

	startTime := time.Now()
	s.logger.Info("Calling OpenAI API for multi-issue detection", nil)

	resp, err := s.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a technical bug analyst. Detect if there are multiple distinct issues in the bug report. Always respond with valid JSON only.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature:    0.1, // Lower for more consistent results
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject},
		},
	)

	if err != nil {
		s.logger.Error("Multi-issue detection failed", err, nil)
		return nil, fmt.Errorf("failed to detect multiple issues: %w", err)
	}

	duration := time.Since(startTime)

	var analysis MultiIssueAnalysis
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &analysis); err != nil {
		s.logger.Error("Failed to parse multi-issue JSON", err, nil)
		return nil, fmt.Errorf("failed to parse multi-issue analysis: %w", err)
	}

	// Validate and clean tags for each issue
	validAppNames := map[string]bool{
		"Jago App":     true,
		"Jagoan App":   true,
		"Depot Portal": true,
		"Service":      true,
	}

	for i := range analysis.Issues {
		cleanedTags := []string{"Bug"} // Always include Bug
		for _, tag := range analysis.Issues[i].Tags {
			if validAppNames[tag] {
				cleanedTags = append(cleanedTags, tag)
				break // Only take first valid app name
			}
		}

		// If no valid app name found, default to "Jago App"
		if len(cleanedTags) == 1 {
			cleanedTags = append(cleanedTags, "Jago App")
		}

		analysis.Issues[i].Tags = cleanedTags
	}

	s.logger.Success("Multi-issue detection completed", map[string]interface{}{
		"duration":   duration.Milliseconds(),
		"issueCount": analysis.IssueCount,
	})

	return &analysis, nil
}
