package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/rizkajuliant20/bug-bot/pkg/logger"
	"github.com/rizkajuliant20/bug-bot/pkg/services"
	"github.com/slack-go/slack"
)

type BugHandler struct {
	slackService       *services.SlackService
	notionService      *services.NotionService
	aiService          *services.OpenAIService
	logger             *logger.Logger
	bugTrackingChannel string
}

func NewBugHandler(
	slackService *services.SlackService,
	notionService *services.NotionService,
	aiService *services.OpenAIService,
	log *logger.Logger,
	bugTrackingChannel string,
) *BugHandler {
	return &BugHandler{
		slackService:       slackService,
		notionService:      notionService,
		aiService:          aiService,
		logger:             log,
		bugTrackingChannel: bugTrackingChannel,
	}
}

func (h *BugHandler) HandleBugReport(channel, ts, threadTS, user, text, teamID string) error {
	if threadTS == "" {
		threadTS = ts
	}

	h.logger.Flow("BUG_HANDLER", "Started processing bug report", map[string]interface{}{
		"user":           user,
		"channel":        channel,
		"ts":             ts,
		"threadTs":       threadTS,
		"messagePreview": truncateString(text, 100),
	})

	// Remove old status reactions to allow re-trigger
	h.slackService.RemoveReaction(channel, ts, "x")
	h.slackService.RemoveReaction(channel, ts, "white_check_mark")

	// Add eyes reaction
	if err := h.slackService.AddReaction(channel, ts, "eyes"); err != nil {
		h.logger.Error("Failed to add eyes reaction", err, nil)
	}

	// Ensure eyes reaction is removed at the end (success or error)
	defer h.slackService.RemoveReaction(channel, ts, "eyes")

	// Clean bug description
	bugDescription := cleanMentions(text)

	// Fetch thread context
	h.logger.Flow("BUG_HANDLER", "Fetching thread context", map[string]interface{}{
		"channel":  channel,
		"threadTs": threadTS,
	})

	threadMessages, err := h.slackService.GetThreadMessages(channel, threadTS)
	if err != nil {
		h.logger.Error("Failed to fetch thread messages", err, nil)
		threadMessages = []slack.Message{}
	}

	reporterName, err := h.slackService.GetUserInfo(user)
	if err != nil {
		h.logger.Error("Failed to get user info", err, nil)
		reporterName = "Unknown"
	}

	h.logger.Info("Thread context retrieved", map[string]interface{}{
		"messageCount": len(threadMessages),
		"reporter":     reporterName,
	})

	// Send analyzing message and store timestamp to delete later
	analyzingMsgTS := h.slackService.SendThreadReply(channel, threadTS, "🤖 Analyzing bug report with AI...", nil)

	// First, detect if there are multiple issues
	h.logger.Flow("BUG_HANDLER", "Detecting multiple issues", map[string]interface{}{
		"descriptionLength": len(bugDescription),
	})

	multiIssue, err := h.aiService.DetectMultipleIssues(bugDescription, threadMessages)
	if err != nil {
		h.logger.Error("Multi-issue detection failed, proceeding with single issue", err, nil)
		multiIssue = &services.MultiIssueAnalysis{IssueCount: 1}
	}

	// If multiple issues detected, ask user which ones to create
	if multiIssue.IssueCount > 1 {
		h.logger.Info("Multiple issues detected", map[string]interface{}{
			"count": multiIssue.IssueCount,
		})

		// Delete analyzing message
		if analyzingMsgTS != "" {
			h.slackService.DeleteMessage(channel, analyzingMsgTS)
		}

		// Store issue data for button interactions
		storeKey := fmt.Sprintf("%s_%s", channel, threadTS)
		GetIssueStore().Set(storeKey, &IssueData{
			Analysis:       multiIssue,
			BugDescription: bugDescription,
			ThreadMessages: threadMessages,
			Reporter:       reporterName,
			TeamID:         teamID,
			Channel:        channel,
			ThreadTS:       threadTS,
			TS:             ts,
		})

		// Send interactive message with buttons
		h.sendMultiIssueSelection(channel, threadTS, multiIssue, bugDescription, threadMessages, reporterName, teamID)
		h.logger.Success("Multi-issue selection sent to user", nil)
		return nil // Wait for user interaction
	}

	// Single issue flow (or fallback)
	h.logger.Flow("BUG_HANDLER", "Starting AI analysis for single issue", map[string]interface{}{
		"descriptionLength": len(bugDescription),
	})

	diagnosis, err := h.aiService.DiagnoseBug(bugDescription, threadMessages)

	// Delete analyzing message after AI completes
	if analyzingMsgTS != "" {
		h.slackService.DeleteMessage(channel, analyzingMsgTS)
	}

	if err != nil {
		h.logger.Error("AI diagnosis failed", err, nil)
		h.slackService.AddReaction(channel, ts, "x")
		h.slackService.SendThreadReply(channel, threadTS, fmt.Sprintf("❌ Error analyzing bug: %v", err), nil)
		return err
	}

	h.logger.Info("AI diagnosis completed", map[string]interface{}{
		"severity": diagnosis.Severity,
		"category": diagnosis.Category,
		"priority": diagnosis.Priority,
	})

	// Generate title
	summaryResult, err := h.aiService.GenerateBugSummary(bugDescription, diagnosis, threadMessages)
	if err != nil {
		h.logger.Error("Title generation failed", err, nil)
		summaryResult = &services.BugSummaryResult{
			Title:   "[Bug][App] " + truncateString(bugDescription, 50),
			AppName: "",
		}
	}

	h.logger.Info("Bug title generated", map[string]interface{}{
		"title":   summaryResult.Title,
		"appName": summaryResult.AppName,
	})

	// Summarize thread
	threadSummary, err := h.aiService.SummarizeThread(threadMessages)
	if err != nil {
		h.logger.Error("Thread summarization failed", err, nil)
		threadSummary = ""
	}

	h.logger.Info("Thread summary generated", map[string]interface{}{
		"hasSummary": threadSummary != "",
		"appName":    summaryResult.AppName,
	})

	// Generate Slack thread URL
	slackThreadURL := h.slackService.GetSlackThreadURL(teamID, channel, threadTS)
	h.logger.Info("Slack thread URL generated", map[string]interface{}{
		"url":     slackThreadURL,
		"appName": summaryResult.AppName,
	})

	// Store analysis data for confirmation
	confirmationID := fmt.Sprintf("confirm_%s_%s", channel, ts)
	StoreConfirmationData(confirmationID, &ConfirmationData{
		BugDescription: bugDescription,
		Diagnosis:      diagnosis,
		Title:          summaryResult.Title,
		AppName:        summaryResult.AppName,
		Reporter:       reporterName,
		SlackThreadURL: slackThreadURL,
		ThreadMessages: threadMessages,
		ThreadSummary:  threadSummary,
		TeamID:         teamID,
		Channel:        channel,
		ThreadTS:       threadTS,
		OriginalTS:     ts,
	})

	// Send confirmation message with preview
	h.sendConfirmationMessage(channel, threadTS, confirmationID, summaryResult.Title, diagnosis)

	h.logger.Success("Confirmation sent - waiting for user action", map[string]interface{}{
		"confirmationID": confirmationID,
	})

	return nil
}

func (h *BugHandler) SendBugNotification(channel, title, reporter, notionURL, slackThreadURL string, diagnosis *services.BugDiagnosis, appName string) {
	h.logger.Flow("BUG_HANDLER", "Sending notification to bug tracking channel", map[string]interface{}{
		"channel": channel,
		"appName": appName,
	})

	platformStr := "N/A"
	if len(diagnosis.Platform) > 0 {
		platformStr = strings.Join(diagnosis.Platform, ", ")
	}

	blocks := []slack.Block{
		slack.NewHeaderBlock(&slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: title,
		}),
		slack.NewSectionBlock(
			nil,
			[]*slack.TextBlockObject{
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Reporter:*\n%s", reporter)},
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Priority:*\n%s", diagnosis.Priority)},
			},
			nil,
		),
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*Category:* %s\n*Platform:* %s", diagnosis.Category, platformStr),
			},
			nil,
			nil,
		),
		slack.NewActionBlock(
			"",
			slack.NewButtonBlockElement("", "", &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: "📝 View in Notion",
			}).WithURL(notionURL).WithStyle(slack.StylePrimary),
			slack.NewButtonBlockElement("", "", &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: "💬 View Thread",
			}).WithURL(slackThreadURL),
		),
	}

	if err := h.slackService.PostMessage(channel, fmt.Sprintf("🐛 New bug ticket created: %s", title), blocks); err != nil {
		h.logger.Error("Failed to send bug notification", err, map[string]interface{}{
			"channel": channel,
			"appName": appName,
		})
	} else {
		h.logger.Success("Bug notification sent to tracking channel", map[string]interface{}{
			"channel": channel,
			"appName": appName,
		})
	}
}

func (h *BugHandler) buildSuccessBlocks(title string, diagnosis *services.BugDiagnosis, notionURL string) []slack.Block {
	return []slack.Block{
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: "✅ *Bug ticket created in Notion*",
			},
			nil,
			nil,
		),
		slack.NewSectionBlock(
			nil,
			[]*slack.TextBlockObject{
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Title:*\n%s", title)},
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Severity:*\n%s", strings.ToUpper(diagnosis.Severity))},
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Category:*\n%s", diagnosis.Category)},
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Priority:*\n%s", diagnosis.Priority)},
			},
			nil,
		),
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*🤖 AI Diagnosis:*\n%s", diagnosis.RootCause),
			},
			nil,
			nil,
		),
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*💡 Suggested Fix:*\n%s", diagnosis.SuggestedFix),
			},
			nil,
			nil,
		),
		slack.NewActionBlock(
			"",
			slack.NewButtonBlockElement("", "", &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: "View in Notion",
			}).WithURL(notionURL).WithStyle(slack.StylePrimary),
		),
	}
}

func cleanMentions(text string) string {
	// Remove @mentions
	cleaned := text
	for strings.Contains(cleaned, "<@") {
		start := strings.Index(cleaned, "<@")
		end := strings.Index(cleaned[start:], ">")
		if end == -1 {
			break
		}
		cleaned = cleaned[:start] + cleaned[start+end+1:]
	}
	return strings.TrimSpace(cleaned)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (h *BugHandler) CreateIssueTicket(issueIndex int, issueData *IssueData) (string, error) {
	issue := issueData.Analysis.Issues[issueIndex]

	h.logger.Flow("BUG_HANDLER", "Creating ticket for specific issue", map[string]interface{}{
		"issueIndex": issueIndex,
		"title":      issue.Title,
	})

	// Create diagnosis from the detected issue (now includes all fields)
	diagnosis := &services.BugDiagnosis{
		Severity:           issue.Severity,
		Category:           issue.Category,
		Priority:           issue.Priority,
		Platform:           issue.Platform,
		Team:               issue.Team,
		Precondition:       "",
		StepsToReproduce:   "",
		ActualResult:       issue.Description,
		ExpectedResult:     "",
		RootCause:          issue.Description,
		SuggestedFix:       "To be investigated",
		AffectedComponents: []string{},
		Tags:               issue.Tags, // Use tags from AI analysis
	}

	// Generate title with app detection
	summaryResult, err := h.aiService.GenerateBugSummary(issue.Description, diagnosis, issueData.ThreadMessages)
	if err != nil {
		h.logger.Error("Title generation failed", err, nil)
		summaryResult = &services.BugSummaryResult{
			Title:   issue.Title,
			AppName: "",
		}
	}

	// Summarize thread
	threadSummary, _ := h.aiService.SummarizeThread(issueData.ThreadMessages)

	// Generate Slack thread URL
	slackThreadURL := h.slackService.GetSlackThreadURL(issueData.TeamID, issueData.Channel, issueData.ThreadTS)

	// Create Notion ticket
	notionPage, err := h.notionService.CreateBugTicket(&services.BugTicketData{
		Title:          summaryResult.Title,
		Description:    issue.Description,
		Diagnosis:      diagnosis,
		Reporter:       issueData.Reporter,
		SlackThreadURL: slackThreadURL,
		ThreadMessages: issueData.ThreadMessages,
		ThreadSummary:  threadSummary,
		// Jago-specific fields
		CreatedBy:    issueData.Reporter,
		Assignee:     "",                 // Will be set manually in Notion
		Environment:  "Production",       // Default to Production
		Reproducible: true,               // Default to true
		Impact:       diagnosis.Severity, // Map from severity
		Urgency:      diagnosis.Severity, // Map from severity
		DueDate:      time.Time{},        // Empty for now
	})

	if err != nil {
		h.logger.Error("Notion ticket creation failed", err, map[string]interface{}{
			"issueIndex": issueIndex,
		})
		return "", err
	}

	notionURL := h.notionService.GetNotionPageURL(notionPage.ID.String())
	h.logger.Success("Notion ticket created for issue", map[string]interface{}{
		"notionUrl":  notionURL,
		"pageId":     notionPage.ID,
		"issueIndex": issueIndex,
	})

	// Send notification to bug tracking channel if configured
	if h.bugTrackingChannel != "" {
		h.SendBugNotification(h.bugTrackingChannel, summaryResult.Title, issueData.Reporter, notionURL, slackThreadURL, diagnosis, summaryResult.AppName)
	}

	return notionURL, nil
}

func getPriorityFromSeverity(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "P0"
	case "high":
		return "P1"
	case "medium":
		return "P2"
	default:
		return "P3"
	}
}

func (h *BugHandler) sendMultiIssueSelection(channel, threadTS string, analysis *services.MultiIssueAnalysis, bugDescription string, threadMessages []slack.Message, reporter, teamID string) {
	// Build issue list text
	var issueList strings.Builder
	issueList.WriteString(fmt.Sprintf("🔍 *Detected %d separate issues in this thread:*\n\n", analysis.IssueCount))

	for i, issue := range analysis.Issues {
		issueList.WriteString(fmt.Sprintf("*Issue %d:* %s\n", i+1, issue.Title))
		issueList.WriteString(fmt.Sprintf("└ %s\n", issue.Description))
		issueList.WriteString(fmt.Sprintf("└ Severity: %s | Category: %s\n\n", issue.Severity, issue.Category))
	}

	// Create buttons for each issue
	var buttons []slack.BlockElement
	for i := range analysis.Issues {
		buttonValue := fmt.Sprintf("create_issue_%d", i)
		buttons = append(buttons, slack.NewButtonBlockElement(
			buttonValue,
			buttonValue,
			&slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: fmt.Sprintf("✅ Create Issue %d", i+1),
			},
		).WithStyle(slack.StylePrimary))
	}

	// Add "Create All" button
	buttons = append(buttons, slack.NewButtonBlockElement(
		"create_all_issues",
		"create_all_issues",
		&slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "✅ Create All Issues",
		},
	).WithStyle(slack.StylePrimary))

	// Add "Cancel" button
	buttons = append(buttons, slack.NewButtonBlockElement(
		"cancel_issues",
		"cancel_issues",
		&slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "❌ Cancel",
		},
	).WithStyle(slack.StyleDanger))

	blocks := []slack.Block{
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: issueList.String(),
			},
			nil,
			nil,
		),
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: "*Which issues would you like to create tickets for?*",
			},
			nil,
			nil,
		),
		slack.NewActionBlock(
			"multi_issue_actions",
			buttons...,
		),
	}

	h.slackService.SendThreadReply(channel, threadTS, "Multiple issues detected", blocks)
}

func (h *BugHandler) SendMultiIssueResults(channel, threadTS, ts string, createdTickets []string, failedIssues []int, totalIssues int) {
	h.slackService.RemoveReaction(channel, ts, "eyes")

	if len(createdTickets) == 0 {
		h.slackService.AddReaction(channel, ts, "x")
		h.slackService.SendThreadReply(channel, threadTS, "❌ Failed to create any tickets", nil)
		return
	}

	h.slackService.AddReaction(channel, ts, "white_check_mark")

	var message strings.Builder
	message.WriteString(fmt.Sprintf("✅ *Created %d/%d bug tickets in Notion*", len(createdTickets), totalIssues))

	if len(failedIssues) > 0 {
		message.WriteString(fmt.Sprintf("\n\n⚠️ Failed to create tickets for issues: %v", failedIssues))
	}

	blocks := []slack.Block{
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: message.String(),
			},
			nil,
			nil,
		),
	}

	// Add button for each created ticket
	var buttons []slack.BlockElement
	for i, url := range createdTickets {
		buttons = append(buttons, slack.NewButtonBlockElement(
			fmt.Sprintf("view_ticket_%d", i),
			url,
			&slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: fmt.Sprintf("📝 View Issue %d", i+1),
			},
		).WithURL(url).WithStyle(slack.StylePrimary))
	}

	if len(buttons) > 0 {
		blocks = append(blocks, slack.NewActionBlock("view_tickets", buttons...))
	}

	h.slackService.SendThreadReply(channel, threadTS, "Bug tickets created", blocks)
	h.logger.Success("Multi-issue results sent to Slack", map[string]interface{}{
		"created": len(createdTickets),
		"failed":  len(failedIssues),
	})
}

func (h *BugHandler) SendSingleIssueSuccess(channel, threadTS, ts string, issueNumber int, notionURL string) {
	h.slackService.RemoveReaction(channel, ts, "eyes")
	h.slackService.AddReaction(channel, ts, "white_check_mark")

	blocks := []slack.Block{
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("✅ *Bug ticket created for Issue %d*", issueNumber),
			},
			nil,
			nil,
		),
		slack.NewActionBlock(
			"view_ticket",
			slack.NewButtonBlockElement(
				"view_notion",
				notionURL,
				&slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "📝 View in Notion",
				},
			).WithURL(notionURL).WithStyle(slack.StylePrimary),
		),
	}

	h.slackService.SendThreadReply(channel, threadTS, fmt.Sprintf("Ticket created: %s", notionURL), blocks)
	h.logger.Success("Single issue ticket created and sent to Slack", map[string]interface{}{
		"issueNumber": issueNumber,
		"notionURL":   notionURL,
	})
}

func (h *BugHandler) SendSingleIssueError(channel, threadTS string, issueNumber int, err error) {
	h.slackService.SendThreadReply(channel, threadTS, fmt.Sprintf("❌ Failed to create ticket for Issue %d: %v", issueNumber, err), nil)
	h.logger.Error("Failed to send single issue error", err, map[string]interface{}{
		"issueNumber": issueNumber,
	})
}

func (h *BugHandler) SendCancellationMessage(channel, threadTS string) {
	h.slackService.SendThreadReply(channel, threadTS, "❌ Bug ticket creation cancelled", nil)
	h.logger.Info("Cancellation message sent", nil)
}

func (h *BugHandler) UpdateMessageToProcessing(channel, messageTS, actionID string, analysis *services.MultiIssueAnalysis) {
	// Build issue list text
	var issueList strings.Builder
	issueList.WriteString(fmt.Sprintf("🔍 *Detected %d separate issues in this thread:*\n\n", analysis.IssueCount))

	for i, issue := range analysis.Issues {
		issueList.WriteString(fmt.Sprintf("*Issue %d:* %s\n", i+1, issue.Title))
		issueList.WriteString(fmt.Sprintf("└ %s\n", issue.Description))
		issueList.WriteString(fmt.Sprintf("└ Severity: %s | Category: %s\n\n", issue.Severity, issue.Category))
	}

	// Determine processing message
	var processingMsg string
	if actionID == "create_all_issues" {
		processingMsg = "⏳ *Creating all tickets...* Please wait."
	} else if actionID == "cancel_issues" {
		processingMsg = "❌ *Cancelled*"
	} else if strings.HasPrefix(actionID, "create_issue_") {
		issueIndexStr := strings.TrimPrefix(actionID, "create_issue_")
		var issueIndex int
		fmt.Sscanf(issueIndexStr, "%d", &issueIndex)
		processingMsg = fmt.Sprintf("⏳ *Creating ticket for Issue %d...* Please wait.", issueIndex+1)
	}

	issueList.WriteString(processingMsg)

	blocks := []slack.Block{
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: issueList.String(),
			},
			nil,
			nil,
		),
	}

	// Update the original message (this disables buttons)
	_, _, _, err := h.slackService.GetClient().UpdateMessage(channel, messageTS, slack.MsgOptionBlocks(blocks...))
	if err != nil {
		h.logger.Error("Failed to update message", err, nil)
	}
}

// Getter methods
func (h *BugHandler) GetSlackService() *services.SlackService {
	return h.slackService
}

func (h *BugHandler) GetNotionService() *services.NotionService {
	return h.notionService
}

func (h *BugHandler) GetBugTrackingChannel() string {
	return h.bugTrackingChannel
}

func (h *BugHandler) sendConfirmationMessage(channel, threadTS, confirmationID, title string, diagnosis *services.BugDiagnosis) {
	platformStr := "N/A"
	if len(diagnosis.Platform) > 0 {
		platformStr = strings.Join(diagnosis.Platform, ", ")
	}

	tagsStr := "N/A"
	if len(diagnosis.Tags) > 0 {
		tagsStr = strings.Join(diagnosis.Tags, ", ")
	}

	blocks := []slack.Block{
		slack.NewHeaderBlock(&slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "🔍 Bug Analysis Complete",
		}),
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*Title:*\n%s", title),
			},
			nil,
			nil,
		),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(
			nil,
			[]*slack.TextBlockObject{
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Priority:*\n%s", diagnosis.Priority)},
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Team:*\n%s", diagnosis.Team)},
			},
			nil,
		),
		slack.NewSectionBlock(
			nil,
			[]*slack.TextBlockObject{
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Platform:*\n%s", platformStr)},
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Category:*\n%s", diagnosis.Category)},
			},
			nil,
		),
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*Tags:* %s", tagsStr),
			},
			nil,
			nil,
		),
	}

	// Add Root Cause if available
	if diagnosis.RootCause != "" {
		blocks = append(blocks, slack.NewDividerBlock())
		blocks = append(blocks, slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*🔍 Root Cause:*\n%s", truncateString(diagnosis.RootCause, 200)),
			},
			nil,
			nil,
		))
	}

	// Add Suggested Fix if available
	if diagnosis.SuggestedFix != "" {
		blocks = append(blocks, slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*💡 Suggested Fix:*\n%s", truncateString(diagnosis.SuggestedFix, 200)),
			},
			nil,
			nil,
		))
	}

	// Add Steps to Reproduce if available
	if diagnosis.StepsToReproduce != "" {
		blocks = append(blocks, slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*📝 Steps to Reproduce:*\n%s", truncateString(diagnosis.StepsToReproduce, 150)),
			},
			nil,
			nil,
		))
	}

	// Final confirmation prompt
	blocks = append(blocks,
		slack.NewDividerBlock(),
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: "✨ *Please review the analysis above.*\nCreate the ticket or cancel?",
			},
			nil,
			nil,
		),
		slack.NewActionBlock(
			"",
			slack.NewButtonBlockElement(
				fmt.Sprintf("create_ticket_%s", confirmationID),
				confirmationID,
				&slack.TextBlockObject{Type: slack.PlainTextType, Text: "✅ Create Ticket"},
			).WithStyle(slack.StylePrimary),
			slack.NewButtonBlockElement(
				fmt.Sprintf("cancel_ticket_%s", confirmationID),
				confirmationID,
				&slack.TextBlockObject{Type: slack.PlainTextType, Text: "❌ Cancel"},
			).WithStyle(slack.StyleDanger),
		),
	)

	h.slackService.SendThreadReply(channel, threadTS, "Bug analysis complete! Please confirm:", blocks)
}
