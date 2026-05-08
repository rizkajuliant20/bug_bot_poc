package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rizkajuliant20/bug-bot/pkg/config"
	"github.com/rizkajuliant20/bug-bot/pkg/handlers"
	"github.com/rizkajuliant20/bug-bot/pkg/logger"
	"github.com/rizkajuliant20/bug-bot/pkg/services"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := logger.New()

	// Initialize services
	slackService := services.NewSlackService(cfg.Slack.BotToken, cfg.Slack.AppToken)
	notionService := services.NewNotionService(cfg.Notion.APIKey, cfg.Notion.DatabaseID, logger)
	aiService := services.NewOpenAIService(cfg.OpenAI.APIKey, logger)

	// Initialize bug handler
	bugHandler := handlers.NewBugHandler(slackService, notionService, aiService, logger, cfg.Slack.BugTrackingChannel)

	// Clear stores on startup (cleanup orphaned messages from previous session)
	handlers.GetIssueStore().Clear()
	handlers.ClearConfirmationStore()
	logger.Info("Issue and confirmation stores cleared on startup", nil)

	// Get socket mode client
	client := slackService.GetSocket()

	// Handle events
	go func() {
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					continue
				}

				client.Ack(*evt.Request)

				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					handleCallbackEvent(eventsAPIEvent.InnerEvent, bugHandler, slackService, cfg, logger)
				}

			case socketmode.EventTypeSlashCommand:
				cmd, ok := evt.Data.(slack.SlashCommand)
				if !ok {
					continue
				}

				client.Ack(*evt.Request)

				if cmd.Command == "/bug" {
					handleSlashCommand(cmd, bugHandler, logger)
				}

			case socketmode.EventTypeInteractive:
				callback, ok := evt.Data.(slack.InteractionCallback)
				if !ok {
					continue
				}

				client.Ack(*evt.Request)

				// Handle button interactions
				if callback.Type == slack.InteractionTypeBlockActions {
					handleButtonInteraction(callback, bugHandler, logger)
				}
			}
		}
	}()

	logger.Success("⚡️ Slack Bug Bot is running!", nil)
	logger.Info("Listening for bug reports...", nil)

	// Start Notion polling if bug tracking channel is configured
	// DISABLED for Jago: Too many existing bugs in database would cause spam
	// Only Slack → Notion flow is active (via emoji reactions)
	if cfg.Slack.BugTrackingChannel != "" {
		logger.Info("Bug tracking channel configured", map[string]interface{}{
			"channel": cfg.Slack.BugTrackingChannel,
		})
		logger.Info("Notion polling DISABLED - only Slack reactions will create tickets", nil)
		// notionService.StartPolling(2*time.Minute, cfg.Slack.BugTrackingChannel, slackService)

		// Start weekly bug report scheduler
		weeklyReportService := services.NewWeeklyReportService(notionService, slackService, logger, cfg.Slack.BugTrackingChannel)
		weeklyReportService.StartWeeklyReports()
		logger.Success("Weekly bug report scheduler started", map[string]interface{}{
			"schedule": "Every Friday at 5 PM",
		})
	} else {
		logger.Info("Bug tracking channel not configured - polling and weekly reports disabled", nil)
	}

	// Run socket mode
	if err := client.Run(); err != nil {
		logger.Error("Failed to start socket mode", err, nil)
		os.Exit(1)
	}
}

func handleCallbackEvent(innerEvent slackevents.EventsAPIInnerEvent, bugHandler *handlers.BugHandler, slackService *services.SlackService, cfg *config.Config, log *logger.Logger) {
	switch ev := innerEvent.Data.(type) {
	case *slackevents.ReactionAddedEvent:
		handleReactionAdded(ev, bugHandler, slackService, log)

	case *slackevents.AppMentionEvent:
		handleAppMention(ev, bugHandler, slackService, cfg, log)
	}
}

func handleReactionAdded(event *slackevents.ReactionAddedEvent, bugHandler *handlers.BugHandler, slackService *services.SlackService, log *logger.Logger) {
	log.Flow("SLACK_EVENT", "Reaction detected", map[string]interface{}{
		"reaction": event.Reaction,
		"channel":  event.Item.Channel,
	})

	// Ignore old reactions (more than 5 minutes old) to avoid processing stale events when bot restarts
	if event.EventTimestamp != "" {
		eventTime, err := strconv.ParseFloat(event.EventTimestamp, 64)
		if err == nil {
			currentTime := float64(time.Now().Unix())
			ageSeconds := currentTime - eventTime

			if ageSeconds > 300 { // 5 minutes = 300 seconds
				log.Info("Ignoring old reaction event", map[string]interface{}{
					"reaction":   event.Reaction,
					"ageSeconds": int(ageSeconds),
				})
				return
			}
		}
	}

	// Check if it's a bug-related emoji
	bugEmojis := []string{"lady_beetle", "ladybug", "bug", "beetle"}
	isBugEmoji := false
	for _, emoji := range bugEmojis {
		if event.Reaction == emoji {
			isBugEmoji = true
			break
		}
	}

	if !isBugEmoji {
		return
	}

	// Get the message
	message, err := slackService.GetMessage(event.Item.Channel, event.Item.Timestamp)
	if err != nil {
		log.Error("Failed to get message", err, map[string]interface{}{
			"reaction": event.Reaction,
		})
		return
	}

	log.Success("Bug reaction detected - triggering bug handler", map[string]interface{}{
		"reaction":        event.Reaction,
		"channel":         event.Item.Channel,
		"messageTS":       event.Item.Timestamp,
		"messageText":     truncate(message.Text, 100),
		"messageThreadTS": message.ThreadTimestamp,
		"messageUser":     message.User,
	})

	// Validate message timestamp matches event
	if message.Timestamp != event.Item.Timestamp {
		log.Error("WARNING: Message timestamp mismatch! Slack API may have returned wrong message", nil, map[string]interface{}{
			"eventTS":   event.Item.Timestamp,
			"messageTS": message.Timestamp,
		})
		// Continue anyway, but this indicates a Slack API issue
	}

	// Extract team ID (not critical for functionality)
	teamID := ""

	// Determine thread timestamp
	// If message has ThreadTimestamp, it's a reply in a thread
	// Otherwise, the message itself is the thread root
	threadTS := message.ThreadTimestamp
	if threadTS == "" {
		threadTS = event.Item.Timestamp
	}

	log.Info("Thread context determined", map[string]interface{}{
		"messageTS": event.Item.Timestamp,
		"threadTS":  threadTS,
	})

	// Handle bug report
	err = bugHandler.HandleBugReport(
		event.Item.Channel,
		event.Item.Timestamp,
		threadTS,
		message.User,
		message.Text,
		teamID,
	)

	if err != nil {
		log.Error("Failed to handle bug report from reaction", err, nil)
	}
}

func handleAppMention(event *slackevents.AppMentionEvent, bugHandler *handlers.BugHandler, slackService *services.SlackService, cfg *config.Config, log *logger.Logger) {
	log.Flow("SLACK_EVENT", "App mention detected", map[string]interface{}{
		"user":    event.User,
		"channel": event.Channel,
	})

	text := strings.ToLower(event.Text)

	// Check if it contains bug-related keywords
	keywords := []string{"bug", "issue", "error", "problem"}
	hasBugKeyword := false
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			hasBugKeyword = true
			break
		}
	}

	if hasBugKeyword {
		log.Success("Bug mention detected - triggering bug handler", map[string]interface{}{
			"user":           event.User,
			"channel":        event.Channel,
			"messagePreview": truncate(event.Text, 50),
		})

		err := bugHandler.HandleBugReport(
			event.Channel,
			event.TimeStamp,
			event.ThreadTimeStamp,
			event.User,
			event.Text,
			"", // Team ID not available in this event
		)

		if err != nil {
			log.Error("Failed to handle bug report from mention", err, nil)
		}
	} else {
		log.Info("App mention without bug keywords - sending help message", map[string]interface{}{
			"user": event.User,
		})

		helpText := "👋 Hi! I help create bug tickets in Notion with AI diagnosis. Mention me with a bug report containing keywords like \"bug\", \"issue\", \"error\", or \"problem\" and I'll analyze it and create a ticket for you!"
		slackService.SendThreadReply(event.Channel, event.TimeStamp, helpText, nil)
	}
}

func handleSlashCommand(cmd slack.SlashCommand, bugHandler *handlers.BugHandler, log *logger.Logger) {
	log.Flow("SLACK_EVENT", "Slash command /bug received", map[string]interface{}{
		"user":    cmd.UserID,
		"channel": cmd.ChannelID,
	})

	if cmd.Text == "" {
		log.Info("Slash command /bug called without description", map[string]interface{}{
			"user": cmd.UserID,
		})
		return
	}

	log.Success("Slash command /bug processing", map[string]interface{}{
		"user":              cmd.UserID,
		"descriptionLength": len(cmd.Text),
	})

	// Create a synthetic timestamp
	ts := fmt.Sprintf("%d.000000", time.Now().Unix())

	err := bugHandler.HandleBugReport(
		cmd.ChannelID,
		ts,
		"",
		cmd.UserID,
		cmd.Text,
		cmd.TeamID,
	)

	if err != nil {
		log.Error("Failed to handle bug report from slash command", err, nil)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func handleButtonInteraction(callback slack.InteractionCallback, bugHandler *handlers.BugHandler, log *logger.Logger) {
	if len(callback.ActionCallback.BlockActions) == 0 {
		return
	}

	action := callback.ActionCallback.BlockActions[0]
	channel := callback.Channel.ID
	messageTS := callback.Container.MessageTs    // The message with buttons
	threadTS := callback.Message.ThreadTimestamp // The thread timestamp

	log.Flow("SLACK_EVENT", "Button interaction received", map[string]interface{}{
		"actionID":  action.ActionID,
		"user":      callback.User.ID,
		"channel":   channel,
		"messageTS": messageTS,
		"threadTS":  threadTS,
	})

	// Ignore view buttons (they're just URLs, not actions)
	if strings.HasPrefix(action.ActionID, "view_") {
		log.Info("Ignoring view button click (URL only)", map[string]interface{}{
			"actionID": action.ActionID,
		})
		return
	}

	// Handle confirmation buttons (create_ticket / cancel_ticket)
	if strings.HasPrefix(action.ActionID, "create_ticket_") || strings.HasPrefix(action.ActionID, "cancel_ticket_") {
		handleConfirmationButton(callback, action, bugHandler, log)
		return
	}

	// Get stored issue data (for multi-issue flow)
	storeKey := fmt.Sprintf("%s_%s", channel, threadTS)
	if threadTS == "" {
		storeKey = fmt.Sprintf("%s_%s", channel, messageTS)
	}

	issueData, ok := handlers.GetIssueStore().Get(storeKey)
	if !ok {
		log.Error("Issue data not found in store", nil, map[string]interface{}{
			"storeKey": storeKey,
		})

		// Just delete the old message silently (no warning needed)
		go bugHandler.GetSlackService().DeleteMessage(channel, messageTS)
		return
	}

	// Handle cancel first (before updating message)
	if action.ActionID == "cancel_issues" {
		log.Info("User cancelled multi-issue creation", map[string]interface{}{
			"user": callback.User.ID,
		})

		// Delete multi-issue selection message silently (no cancellation message needed)
		bugHandler.GetSlackService().DeleteMessage(channel, messageTS)

		// Clean up store
		handlers.GetIssueStore().Delete(storeKey)
		return
	}

	// Handle create all issues
	if action.ActionID == "create_all_issues" {
		log.Info("User requested to create all issues", map[string]interface{}{
			"user":  callback.User.ID,
			"count": issueData.Analysis.IssueCount,
		})

		// Update message to show processing
		processingBlocks := []slack.Block{
			slack.NewSectionBlock(
				&slack.TextBlockObject{
					Type: slack.MarkdownType,
					Text: fmt.Sprintf("⏳ *Creating %d tickets...* Please wait.", issueData.Analysis.IssueCount),
				},
				nil,
				nil,
			),
		}
		bugHandler.GetSlackService().GetClient().UpdateMessage(channel, messageTS, slack.MsgOptionBlocks(processingBlocks...))

		var createdTickets []handlers.TicketInfo
		var failedIssues []int

		for i := range issueData.Analysis.Issues {
			notionURL, title, err := bugHandler.CreateIssueTicket(i, issueData)
			if err != nil {
				log.Error("Failed to create ticket for issue", err, map[string]interface{}{
					"issueIndex": i,
				})
				failedIssues = append(failedIssues, i+1)
			} else {
				createdTickets = append(createdTickets, handlers.TicketInfo{
					URL:   notionURL,
					Title: title,
				})
			}
		}

		// Update message with results summary
		bugHandler.UpdateMultiIssueResults(channel, messageTS, issueData.TS, createdTickets, failedIssues, issueData.Analysis)

		// Clean up store
		handlers.GetIssueStore().Delete(storeKey)
		return
	}

	// Handle create specific issue
	if strings.HasPrefix(action.ActionID, "create_issue_") {
		issueIndexStr := strings.TrimPrefix(action.ActionID, "create_issue_")
		var issueIndex int
		fmt.Sscanf(issueIndexStr, "%d", &issueIndex)

		log.Info("User requested to create specific issue", map[string]interface{}{
			"user":       callback.User.ID,
			"issueIndex": issueIndex,
		})

		// Update message to show processing
		processingBlocks := []slack.Block{
			slack.NewSectionBlock(
				&slack.TextBlockObject{
					Type: slack.MarkdownType,
					Text: fmt.Sprintf("⏳ *Creating ticket for Issue %d...* Please wait.", issueIndex+1),
				},
				nil,
				nil,
			),
		}
		bugHandler.GetSlackService().GetClient().UpdateMessage(channel, messageTS, slack.MsgOptionBlocks(processingBlocks...))

		notionURL, title, err := bugHandler.CreateIssueTicket(issueIndex, issueData)
		if err != nil {
			log.Error("Failed to create ticket for issue", err, map[string]interface{}{
				"issueIndex": issueIndex,
			})
			// Update message with error
			errorBlocks := []slack.Block{
				slack.NewSectionBlock(
					&slack.TextBlockObject{
						Type: slack.MarkdownType,
						Text: fmt.Sprintf("❌ *Failed to create ticket for Issue %d*\n%v", issueIndex+1, err),
					},
					nil,
					nil,
				),
			}
			bugHandler.GetSlackService().GetClient().UpdateMessage(channel, messageTS, slack.MsgOptionBlocks(errorBlocks...))
		} else {
			// Update message with success
			successBlocks := []slack.Block{
				slack.NewSectionBlock(
					&slack.TextBlockObject{
						Type: slack.MarkdownType,
						Text: fmt.Sprintf("✅ *Ticket Created:* %s", title),
					},
					nil,
					nil,
				),
				slack.NewActionBlock(
					"",
					slack.NewButtonBlockElement("view_notion_single", "", &slack.TextBlockObject{
						Type: slack.PlainTextType,
						Text: "📝 View in Notion",
					}).WithURL(notionURL).WithStyle(slack.StylePrimary),
				),
			}
			bugHandler.GetSlackService().GetClient().UpdateMessage(channel, messageTS, slack.MsgOptionBlocks(successBlocks...))

			// Also remove eyes reaction
			bugHandler.GetSlackService().RemoveReaction(channel, issueData.TS, "eyes")
		}

		// Clean up store after creating single issue
		handlers.GetIssueStore().Delete(storeKey)
		return
	}
}

func handleConfirmationButton(callback slack.InteractionCallback, action *slack.BlockAction, bugHandler *handlers.BugHandler, log *logger.Logger) {
	confirmationID := action.Value
	channel := callback.Channel.ID
	messageTS := callback.Container.MessageTs

	// Get confirmation data
	data, ok := handlers.GetConfirmationData(confirmationID)
	if !ok {
		log.Error("Confirmation data not found", nil, map[string]interface{}{
			"confirmationID": confirmationID,
		})
		return
	}

	// Handle cancel
	if strings.HasPrefix(action.ActionID, "cancel_ticket_") {
		log.Info("User cancelled ticket creation", map[string]interface{}{
			"user": callback.User.ID,
		})

		// Delete confirmation message silently (no reaction needed)
		bugHandler.GetSlackService().DeleteMessage(channel, messageTS)

		// Clean up
		handlers.DeleteConfirmationData(confirmationID)
		return
	}

	// Handle create ticket
	if strings.HasPrefix(action.ActionID, "create_ticket_") {
		log.Info("User confirmed ticket creation", map[string]interface{}{
			"user": callback.User.ID,
		})

		// Update message to show processing
		bugHandler.GetSlackService().GetClient().UpdateMessage(
			channel,
			messageTS,
			slack.MsgOptionText("⏳ Creating ticket...", false),
			slack.MsgOptionBlocks(),
		)

		// Create Notion ticket
		notionPage, err := bugHandler.GetNotionService().CreateBugTicket(&services.BugTicketData{
			Title:          data.Title,
			Description:    data.BugDescription,
			Diagnosis:      data.Diagnosis,
			Reporter:       data.Reporter,
			SlackThreadURL: data.SlackThreadURL,
			ThreadMessages: data.ThreadMessages,
			ThreadSummary:  data.ThreadSummary,
			MediaFiles:     data.MediaFiles,
			CreatedBy:      data.Reporter,
			Assignee:       "",
			Environment:    "Production",
			Reproducible:   true,
			Impact:         data.Diagnosis.Severity,
			Urgency:        data.Diagnosis.Severity,
			DueDate:        time.Time{},
		}, bugHandler.GetSlackService())

		if err != nil {
			log.Error("Failed to create Notion ticket", err, nil)
			bugHandler.GetSlackService().GetClient().UpdateMessage(
				channel,
				messageTS,
				slack.MsgOptionText(fmt.Sprintf("❌ Error creating ticket: %v", err), false),
				slack.MsgOptionBlocks(),
			)
			bugHandler.GetSlackService().AddReaction(data.Channel, data.OriginalTS, "x")
			handlers.DeleteConfirmationData(confirmationID)
			return
		}

		notionURL := bugHandler.GetNotionService().GetNotionPageURL(notionPage.ID.String())

		// Delete confirmation message to avoid spam
		bugHandler.GetSlackService().DeleteMessage(channel, messageTS)

		// Send success message with bug title and link
		blocks := []slack.Block{
			slack.NewSectionBlock(
				&slack.TextBlockObject{
					Type: slack.MarkdownType,
					Text: fmt.Sprintf("✅ *Ticket Created:* %s", data.Title),
				},
				nil,
				nil,
			),
			slack.NewActionBlock(
				"",
				slack.NewButtonBlockElement("view_notion_single", "", &slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "📝 View in Notion",
				}).WithURL(notionURL).WithStyle(slack.StylePrimary),
			),
		}

		bugHandler.GetSlackService().SendThreadReply(data.Channel, data.ThreadTS, "", blocks)

		// Send notification to bug tracking channel
		if bugHandler.GetBugTrackingChannel() != "" {
			bugHandler.SendBugNotification(
				bugHandler.GetBugTrackingChannel(),
				data.Title,
				data.Reporter,
				notionURL,
				data.SlackThreadURL,
				data.Diagnosis,
				data.AppName,
			)
		}

		// Clean up
		handlers.DeleteConfirmationData(confirmationID)
	}
}
