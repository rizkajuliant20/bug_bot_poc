package services

import (
	"fmt"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

type SlackService struct {
	client   *slack.Client
	socket   *socketmode.Client
	botToken string
}

func NewSlackService(botToken, appToken string) *SlackService {
	client := slack.New(
		botToken,
		slack.OptionDebug(false),
		slack.OptionAppLevelToken(appToken),
	)

	socket := socketmode.New(
		client,
		socketmode.OptionDebug(false),
	)

	return &SlackService{
		client:   client,
		socket:   socket,
		botToken: botToken,
	}
}

func (s *SlackService) GetClient() *slack.Client {
	return s.client
}

func (s *SlackService) GetSocket() *socketmode.Client {
	return s.socket
}

// GetThreadMessages retrieves all messages in a thread
func (s *SlackService) GetThreadMessages(channel, threadTS string) ([]slack.Message, error) {
	params := &slack.GetConversationRepliesParameters{
		ChannelID: channel,
		Timestamp: threadTS,
	}

	messages, _, _, err := s.client.GetConversationReplies(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread messages: %w", err)
	}

	// Debug: Log first and last message preview
	if len(messages) > 0 {
		firstMsg := truncateString(messages[0].Text, 50)
		lastMsg := truncateString(messages[len(messages)-1].Text, 50)
		println("DEBUG GetThreadMessages: channel=", channel, "threadTS=", threadTS, "count=", len(messages))
		println("  First:", firstMsg)
		println("  Last:", lastMsg)
	}

	return messages, nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// GetUserInfo retrieves user information
func (s *SlackService) GetUserInfo(userID string) (string, error) {
	user, err := s.client.GetUserInfo(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}

	if user.RealName != "" {
		return user.RealName, nil
	}
	return user.Name, nil
}

// GetSlackThreadURL generates a Slack thread URL
func (s *SlackService) GetSlackThreadURL(teamID, channel, threadTS string) string {
	messageID := "p" + strings.ReplaceAll(threadTS, ".", "")
	return fmt.Sprintf("https://slack.com/archives/%s/%s", channel, messageID)
}

// SendThreadReply sends a reply to a thread and returns the message timestamp
func (s *SlackService) SendThreadReply(channel, threadTS, text string, blocks []slack.Block) string {
	options := []slack.MsgOption{
		slack.MsgOptionText(text, false),
		slack.MsgOptionTS(threadTS),
	}

	if blocks != nil {
		options = append(options, slack.MsgOptionBlocks(blocks...))
	}

	_, msgTS, err := s.client.PostMessage(channel, options...)
	if err != nil {
		return ""
	}

	return msgTS
}

// DeleteMessage deletes a message from a channel
func (s *SlackService) DeleteMessage(channel, timestamp string) error {
	_, _, err := s.client.DeleteMessage(channel, timestamp)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}
	return nil
}

// AddReaction adds a reaction to a message
func (s *SlackService) AddReaction(channel, timestamp, reaction string) error {
	ref := slack.ItemRef{
		Channel:   channel,
		Timestamp: timestamp,
	}

	err := s.client.AddReaction(reaction, ref)
	if err != nil && !strings.Contains(err.Error(), "already_reacted") {
		return fmt.Errorf("failed to add reaction: %w", err)
	}

	return nil
}

// RemoveReaction removes a reaction from a message
func (s *SlackService) RemoveReaction(channel, timestamp, reaction string) error {
	ref := slack.ItemRef{
		Channel:   channel,
		Timestamp: timestamp,
	}

	err := s.client.RemoveReaction(reaction, ref)
	if err != nil {
		// Ignore if reaction doesn't exist
		return nil
	}

	return nil
}

// PostMessage posts a message to a channel
func (s *SlackService) PostMessage(channel, text string, blocks []slack.Block) error {
	options := []slack.MsgOption{
		slack.MsgOptionText(text, false),
	}

	if len(blocks) > 0 {
		options = append(options, slack.MsgOptionBlocks(blocks...))
	}

	_, _, err := s.client.PostMessage(channel, options...)
	if err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}

	return nil
}

// GetMessage retrieves a specific message (works for both channel messages and thread replies)
func (s *SlackService) GetMessage(channel, timestamp string) (*slack.Message, error) {
	// First try to get as channel message
	history, err := s.client.GetConversationHistory(&slack.GetConversationHistoryParameters{
		ChannelID: channel,
		Latest:    timestamp,
		Limit:     1,
		Inclusive: true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	if len(history.Messages) > 0 && history.Messages[0].Timestamp == timestamp {
		return &history.Messages[0], nil
	}

	// If not found, it might be a thread reply - we need to find its parent thread
	// Try getting recent messages and search for the timestamp
	recentHistory, err := s.client.GetConversationHistory(&slack.GetConversationHistoryParameters{
		ChannelID: channel,
		Limit:     100, // Get recent messages to find thread parents
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get conversation history: %w", err)
	}

	// Search through messages to find one with replies
	for _, msg := range recentHistory.Messages {
		if msg.ReplyCount > 0 {
			// This message has replies, check if our timestamp is in there
			replies, _, _, err := s.client.GetConversationReplies(&slack.GetConversationRepliesParameters{
				ChannelID: channel,
				Timestamp: msg.Timestamp,
			})

			if err != nil {
				continue
			}

			for _, reply := range replies {
				if reply.Timestamp == timestamp {
					return &reply, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("message not found")
}

// ExtractTeamID extracts team ID from event
func ExtractTeamID(event interface{}) string {
	// Team ID is typically available in the outer event wrapper
	// For now, return empty string as it's not critical for functionality
	return ""
}
