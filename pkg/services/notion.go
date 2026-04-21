package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jomei/notionapi"
	"github.com/rizkajuliant20/bug-bot/pkg/logger"
	"github.com/slack-go/slack"
)

type NotionService struct {
	client     *notionapi.Client
	databaseID notionapi.DatabaseID
	logger     *logger.Logger
}

type BugTicketData struct {
	Title          string
	Description    string
	Diagnosis      *BugDiagnosis
	Reporter       string
	SlackThreadURL string
	ThreadMessages []slack.Message
	ThreadSummary  string
}

func NewNotionService(apiKey, databaseID string, log *logger.Logger) *NotionService {
	client := notionapi.NewClient(notionapi.Token(apiKey))
	return &NotionService{
		client:     client,
		databaseID: notionapi.DatabaseID(databaseID),
		logger:     log,
	}
}

func (s *NotionService) CreateBugTicket(data *BugTicketData) (*notionapi.Page, error) {
	s.logger.Flow("NOTION_SERVICE", "Creating bug ticket", map[string]interface{}{
		"title": data.Title,
	})

	// Build properties
	properties := notionapi.Properties{
		"Title": notionapi.TitleProperty{
			Type: notionapi.PropertyTypeTitle,
			Title: []notionapi.RichText{
				{
					Type: notionapi.ObjectTypeText,
					Text: &notionapi.Text{Content: data.Title},
				},
			},
		},
		"Status": notionapi.StatusProperty{
			Type: notionapi.PropertyTypeStatus,
			Status: notionapi.Status{Name: "Not started"},
		},
		"Reporter": notionapi.RichTextProperty{
			Type: notionapi.PropertyTypeRichText,
			RichText: []notionapi.RichText{
				{
					Type: notionapi.ObjectTypeText,
					Text: &notionapi.Text{Content: data.Reporter},
				},
			},
		},
	}

	// Add diagnosis properties
	if data.Diagnosis != nil {
		properties["Severity"] = notionapi.SelectProperty{
			Type:   notionapi.PropertyTypeSelect,
			Select: notionapi.Option{Name: strings.Title(data.Diagnosis.Severity)},
		}
		properties["Priority"] = notionapi.SelectProperty{
			Type:   notionapi.PropertyTypeSelect,
			Select: notionapi.Option{Name: data.Diagnosis.Priority},
		}
		properties["Category"] = notionapi.SelectProperty{
			Type:   notionapi.PropertyTypeSelect,
			Select: notionapi.Option{Name: data.Diagnosis.Category},
		}
		properties["Team"] = notionapi.SelectProperty{
			Type:   notionapi.PropertyTypeSelect,
			Select: notionapi.Option{Name: data.Diagnosis.Team},
		}

		// Platform multi-select
		if len(data.Diagnosis.Platform) > 0 {
			var platformOptions []notionapi.Option
			for _, p := range data.Diagnosis.Platform {
				platformOptions = append(platformOptions, notionapi.Option{Name: p})
			}
			properties["Platform"] = notionapi.MultiSelectProperty{
				Type:        notionapi.PropertyTypeMultiSelect,
				MultiSelect: platformOptions,
			}
		}

		// Tags multi-select
		if len(data.Diagnosis.Tags) > 0 {
			var tagOptions []notionapi.Option
			for _, t := range data.Diagnosis.Tags {
				tagOptions = append(tagOptions, notionapi.Option{Name: t})
			}
			properties["Tags"] = notionapi.MultiSelectProperty{
				Type:        notionapi.PropertyTypeMultiSelect,
				MultiSelect: tagOptions,
			}
		}
	}

	// Add Slack Thread URL
	if data.SlackThreadURL != "" {
		properties["Slack Thread"] = notionapi.URLProperty{
			Type: notionapi.PropertyTypeURL,
			URL:  data.SlackThreadURL,
		}
	}

	// Build page content
	children := s.buildPageContent(data)

	// Create page
	page, err := s.client.Page.Create(context.Background(), &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:       notionapi.ParentTypeDatabaseID,
			DatabaseID: s.databaseID,
		},
		Properties: properties,
		Children:   children,
	})

	if err != nil {
		s.logger.Error("Failed to create Notion page", err, nil)
		return nil, fmt.Errorf("failed to create notion page: %w", err)
	}

	s.logger.Success("Notion ticket created", map[string]interface{}{
		"pageId": page.ID,
	})

	return page, nil
}

func (s *NotionService) buildPageContent(data *BugTicketData) []notionapi.Block {
	var blocks []notionapi.Block

	// Bug Description
	blocks = append(blocks, notionapi.Heading2Block{
		BasicBlock: notionapi.BasicBlock{
			Object: notionapi.ObjectTypeBlock,
			Type:   notionapi.BlockTypeHeading2,
		},
		Heading2: notionapi.Heading{
			RichText: []notionapi.RichText{
				{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: "🐛 Bug Description"}},
			},
		},
	})

	blocks = append(blocks, notionapi.ParagraphBlock{
		BasicBlock: notionapi.BasicBlock{
			Object: notionapi.ObjectTypeBlock,
			Type:   notionapi.BlockTypeParagraph,
		},
		Paragraph: notionapi.Paragraph{
			RichText: []notionapi.RichText{
				{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: data.Description}},
			},
		},
	})

	if data.Diagnosis != nil {
		// Precondition
		if data.Diagnosis.Precondition != "" {
			blocks = append(blocks, notionapi.Heading3Block{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeHeading3},
				Heading3:   notionapi.Heading{RichText: []notionapi.RichText{{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: "Precondition"}}}},
			})
			blocks = append(blocks, notionapi.ParagraphBlock{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeParagraph},
				Paragraph:  notionapi.Paragraph{RichText: []notionapi.RichText{{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: data.Diagnosis.Precondition}}}},
			})
		}

		// Steps to Reproduce
		if data.Diagnosis.StepsToReproduce != "" {
			blocks = append(blocks, notionapi.Heading3Block{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeHeading3},
				Heading3:   notionapi.Heading{RichText: []notionapi.RichText{{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: "Steps to Reproduce"}}}},
			})
			blocks = append(blocks, notionapi.ParagraphBlock{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeParagraph},
				Paragraph:  notionapi.Paragraph{RichText: []notionapi.RichText{{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: data.Diagnosis.StepsToReproduce}}}},
			})
		}

		// Actual Result
		if data.Diagnosis.ActualResult != "" {
			blocks = append(blocks, notionapi.Heading3Block{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeHeading3},
				Heading3:   notionapi.Heading{RichText: []notionapi.RichText{{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: "Actual Result"}}}},
			})
			blocks = append(blocks, notionapi.ParagraphBlock{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeParagraph},
				Paragraph:  notionapi.Paragraph{RichText: []notionapi.RichText{{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: data.Diagnosis.ActualResult}}}},
			})
		}

		// Expected Result
		if data.Diagnosis.ExpectedResult != "" {
			blocks = append(blocks, notionapi.Heading3Block{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeHeading3},
				Heading3:   notionapi.Heading{RichText: []notionapi.RichText{{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: "Expected Result"}}}},
			})
			blocks = append(blocks, notionapi.ParagraphBlock{
				BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeParagraph},
				Paragraph:  notionapi.Paragraph{RichText: []notionapi.RichText{{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: data.Diagnosis.ExpectedResult}}}},
			})
		}

		// QA Diagnosis
		blocks = append(blocks, notionapi.Heading2Block{
			BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeHeading2},
			Heading2:   notionapi.Heading{RichText: []notionapi.RichText{{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: "🔍 QA Diagnosis"}}}},
		})

		diagnosisText := fmt.Sprintf("**Root Cause:** %s\n\n**Suggested Fix:** %s", data.Diagnosis.RootCause, data.Diagnosis.SuggestedFix)
		blocks = append(blocks, notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeParagraph},
			Paragraph:  notionapi.Paragraph{RichText: []notionapi.RichText{{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: diagnosisText}}}},
		})
	}

	// Thread Summary
	if data.ThreadSummary != "" {
		blocks = append(blocks, notionapi.Heading2Block{
			BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeHeading2},
			Heading2:   notionapi.Heading{RichText: []notionapi.RichText{{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: "💬 Thread Summary"}}}},
		})
		blocks = append(blocks, notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{Object: notionapi.ObjectTypeBlock, Type: notionapi.BlockTypeParagraph},
			Paragraph:  notionapi.Paragraph{RichText: []notionapi.RichText{{Type: notionapi.ObjectTypeText, Text: &notionapi.Text{Content: data.ThreadSummary}}}},
		})
	}

	return blocks
}

func (s *NotionService) GetNotionPageURL(pageID string) string {
	cleanID := strings.ReplaceAll(pageID, "-", "")
	return fmt.Sprintf("https://notion.so/%s", cleanID)
}

// Notion Polling
type TrackedBugs struct {
	BugIDs []string `json:"bugIds"`
}

const trackingFile = ".notion-tracking.json"

func (s *NotionService) PollForNewBugs(bugTrackingChannel string, slackService *SlackService) error {
	s.logger.Flow("NOTION_POLLING", "Starting poll cycle", nil)

	trackedBugs := s.loadTrackedBugs()

	// Query Notion for recent bugs (last 10 minutes)
	tenMinutesAgo := time.Now().Add(-10 * time.Minute).Format(time.RFC3339)
	s.logger.Info("Querying Notion database", map[string]interface{}{"since": tenMinutesAgo})

	query, err := s.client.Database.Query(context.Background(), s.databaseID, &notionapi.DatabaseQueryRequest{
		Sorts: []notionapi.SortObject{
			{
				Timestamp: "created_time",
				Direction: notionapi.SortOrderDESC,
			},
		},
	})

	if err != nil {
		s.logger.Error("Notion query failed", err, nil)
		return err
	}

	s.logger.Info("Notion query completed", map[string]interface{}{"totalResults": len(query.Results)})

	newBugs := 0
	for _, page := range query.Results {
		pageID := page.ID.String()

		// Skip if already tracked
		if contains(trackedBugs.BugIDs, pageID) {
			continue
		}

		s.logger.Flow("NOTION_POLLING", "Processing new page", map[string]interface{}{"pageId": pageID})

		// Extract properties
		title := extractTitle(page.Properties)
		tags := extractMultiSelect(page.Properties, "Tags")
		slackThread := extractURL(page.Properties, "Slack Thread")

		// Skip if no bug tag
		if !containsIgnoreCase(tags, "bug") {
			s.logger.Info("Skipping page without bug tag", map[string]interface{}{"pageId": pageID, "tags": tags})
			trackedBugs.BugIDs = append(trackedBugs.BugIDs, pageID)
			continue
		}

		// Skip if automation-created (has Slack Thread URL)
		if slackThread != "" {
			s.logger.Info("Skipping automation-created bug", map[string]interface{}{"pageId": pageID, "slackThread": slackThread})
			trackedBugs.BugIDs = append(trackedBugs.BugIDs, pageID)
			continue
		}

		s.logger.Success("Found manually created bug", map[string]interface{}{"pageId": pageID, "title": title})

		// Send notification
		if bugTrackingChannel != "" {
			s.sendBugNotification(slackService, bugTrackingChannel, page)
		}

		trackedBugs.BugIDs = append(trackedBugs.BugIDs, pageID)
		newBugs++
	}

	if newBugs > 0 {
		s.logger.Success("Poll cycle completed", map[string]interface{}{"newBugsFound": newBugs})
		s.saveTrackedBugs(trackedBugs)
	} else {
		s.logger.Info("Poll cycle completed - no new bugs found", nil)
	}

	return nil
}

func (s *NotionService) sendBugNotification(slackService *SlackService, channel string, page notionapi.Page) {
	title := extractTitle(page.Properties)
	severity := extractSelect(page.Properties, "Severity")
	priority := extractSelect(page.Properties, "Priority")
	category := extractSelect(page.Properties, "Category")
	reporter := extractRichText(page.Properties, "Reporter")
	platform := extractMultiSelect(page.Properties, "Platform")

	notionURL := s.GetNotionPageURL(page.ID.String())

	s.logger.Flow("NOTION_POLLING", "Sending bug notification", map[string]interface{}{"bugId": page.ID, "title": title})

	blocks := []slack.Block{
		slack.NewHeaderBlock(&slack.TextBlockObject{Type: slack.PlainTextType, Text: "🐛 New Bug Ticket"}),
		slack.NewSectionBlock(
			nil,
			[]*slack.TextBlockObject{
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Title:*\n%s", title)},
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Reporter:*\n%s", reporter)},
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Severity:*\n%s", severity)},
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Priority:*\n%s", priority)},
			},
			nil,
		),
		slack.NewSectionBlock(
			&slack.TextBlockObject{Type: slack.MarkdownType, Text: fmt.Sprintf("*Category:* %s\n*Platform:* %s", category, strings.Join(platform, ", "))},
			nil,
			nil,
		),
		slack.NewActionBlock(
			"",
			slack.NewButtonBlockElement("", "", &slack.TextBlockObject{Type: slack.PlainTextType, Text: "📝 View in Notion"}).WithURL(notionURL).WithStyle(slack.StylePrimary),
		),
	}

	if err := slackService.PostMessage(channel, fmt.Sprintf("🐛 New bug ticket created manually: %s", title), blocks); err != nil {
		s.logger.Error("Failed to send bug notification", err, map[string]interface{}{"bugId": page.ID, "title": title})
	} else {
		s.logger.Success("Notification sent for manually created bug", map[string]interface{}{"bugId": page.ID, "title": title})
	}
}

func (s *NotionService) loadTrackedBugs() *TrackedBugs {
	data, err := os.ReadFile(trackingFile)
	if err != nil {
		s.logger.Info("Tracked bugs file not found, starting fresh", nil)
		return &TrackedBugs{BugIDs: []string{}}
	}

	var tracked TrackedBugs
	if err := json.Unmarshal(data, &tracked); err != nil {
		s.logger.Error("Failed to parse tracked bugs", err, nil)
		return &TrackedBugs{BugIDs: []string{}}
	}

	s.logger.Info("Tracked bugs loaded", map[string]interface{}{"count": len(tracked.BugIDs)})
	return &tracked
}

func (s *NotionService) saveTrackedBugs(tracked *TrackedBugs) {
	data, err := json.MarshalIndent(tracked, "", "  ")
	if err != nil {
		s.logger.Error("Failed to marshal tracked bugs", err, nil)
		return
	}

	if err := os.WriteFile(trackingFile, data, 0644); err != nil {
		s.logger.Error("Failed to save tracked bugs", err, nil)
		return
	}

	s.logger.Info("Tracked bugs saved", map[string]interface{}{"count": len(tracked.BugIDs)})
}

// Helper functions
func extractTitle(props notionapi.Properties) string {
	if title, ok := props["Title"].(*notionapi.TitleProperty); ok && len(title.Title) > 0 {
		return title.Title[0].PlainText
	}
	return "Untitled"
}

func extractSelect(props notionapi.Properties, name string) string {
	if sel, ok := props[name].(*notionapi.SelectProperty); ok && sel.Select.Name != "" {
		return sel.Select.Name
	}
	return "Unknown"
}

func extractMultiSelect(props notionapi.Properties, name string) []string {
	if ms, ok := props[name].(*notionapi.MultiSelectProperty); ok {
		var values []string
		for _, opt := range ms.MultiSelect {
			values = append(values, opt.Name)
		}
		return values
	}
	return []string{}
}

func extractRichText(props notionapi.Properties, name string) string {
	if rt, ok := props[name].(*notionapi.RichTextProperty); ok && len(rt.RichText) > 0 {
		return rt.RichText[0].PlainText
	}
	return "Unknown"
}

func extractURL(props notionapi.Properties, name string) string {
	if url, ok := props[name].(*notionapi.URLProperty); ok {
		return url.URL
	}
	return ""
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsIgnoreCase(slice []string, item string) bool {
	lowerItem := strings.ToLower(item)
	for _, s := range slice {
		if strings.ToLower(s) == lowerItem {
			return true
		}
	}
	return false
}

func (s *NotionService) StartPolling(interval time.Duration, bugTrackingChannel string, slackService *SlackService) {
	s.logger.Success("Notion polling service started", map[string]interface{}{"intervalMinutes": interval.Minutes()})

	// Initial poll
	s.logger.Info("Running initial poll", nil)
	s.PollForNewBugs(bugTrackingChannel, slackService)

	// Set up ticker
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			s.PollForNewBugs(bugTrackingChannel, slackService)
		}
	}()
}
