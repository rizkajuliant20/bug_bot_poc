package main

import (
	"fmt"
	"log"
	"os"
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
	bugHandler := handlers.NewBugHandler(slackService, notionService, aiService, logger)

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
	if cfg.Slack.BugTrackingChannel != "" {
		logger.Info("Bug tracking channel configured", map[string]interface{}{
			"channel": cfg.Slack.BugTrackingChannel,
		})
		notionService.StartPolling(2*time.Minute, cfg.Slack.BugTrackingChannel, slackService)
	} else {
		logger.Info("Bug tracking channel not configured - polling disabled", nil)
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
		"reaction":       event.Reaction,
		"channel":        event.Item.Channel,
		"messagePreview": truncate(message.Text, 50),
	})

	// Extract team ID (not critical for functionality)
	teamID := ""

	// Handle bug report
	err = bugHandler.HandleBugReport(
		event.Item.Channel,
		event.Item.Timestamp,
		message.ThreadTimestamp,
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
	messageTS := callback.Container.MessageTs // The message with buttons
	threadTS := callback.Message.ThreadTimestamp // The thread timestamp
	
	log.Flow("SLACK_EVENT", "Button interaction received", map[string]interface{}{
		"actionID":  action.ActionID,
		"user":      callback.User.ID,
		"channel":   channel,
		"messageTS": messageTS,
		"threadTS":  threadTS,
	})

	// Get stored issue data
	storeKey := fmt.Sprintf("%s_%s", channel, threadTS)
	if threadTS == "" {
		storeKey = fmt.Sprintf("%s_%s", channel, messageTS)
	}
	
	issueData, ok := handlers.GetIssueStore().Get(storeKey)
	if !ok {
		log.Error("Issue data not found in store", nil, map[string]interface{}{
			"storeKey": storeKey,
		})
		return
	}

	// Update message to show processing (disable buttons)
	// Use messageTS (the message with buttons), not threadTS
	bugHandler.UpdateMessageToProcessing(channel, messageTS, action.ActionID, issueData.Analysis)

	// Handle cancel
	if action.ActionID == "cancel_issues" {
		log.Info("User cancelled multi-issue creation", map[string]interface{}{
			"user": callback.User.ID,
		})
		
		// Clean up store
		handlers.GetIssueStore().Delete(storeKey)
		
		// Send cancellation message
		bugHandler.SendCancellationMessage(channel, issueData.ThreadTS)
		return
	}

	// Handle create all issues
	if action.ActionID == "create_all_issues" {
		log.Info("User requested to create all issues", map[string]interface{}{
			"user":  callback.User.ID,
			"count": issueData.Analysis.IssueCount,
		})
		
		var createdTickets []string
		var failedIssues []int
		
		for i := range issueData.Analysis.Issues {
			notionURL, err := bugHandler.CreateIssueTicket(i, issueData)
			if err != nil {
				log.Error("Failed to create ticket for issue", err, map[string]interface{}{
					"issueIndex": i,
				})
				failedIssues = append(failedIssues, i+1)
			} else {
				createdTickets = append(createdTickets, notionURL)
			}
		}
		
		// Send summary to Slack
		bugHandler.SendMultiIssueResults(channel, issueData.ThreadTS, issueData.TS, createdTickets, failedIssues, issueData.Analysis.IssueCount)
		
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
		
		notionURL, err := bugHandler.CreateIssueTicket(issueIndex, issueData)
		if err != nil {
			log.Error("Failed to create ticket for issue", err, map[string]interface{}{
				"issueIndex": issueIndex,
			})
			bugHandler.SendSingleIssueError(channel, issueData.ThreadTS, issueIndex+1, err)
		} else {
			bugHandler.SendSingleIssueSuccess(channel, issueData.ThreadTS, issueData.TS, issueIndex+1, notionURL)
		}
		
		// Clean up store
		handlers.GetIssueStore().Delete(storeKey)
		return
	}
}
