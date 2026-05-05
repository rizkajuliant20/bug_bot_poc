package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jomei/notionapi"
	"github.com/rizkajuliant20/bug-bot/pkg/logger"
	"github.com/slack-go/slack"
)

type WeeklyReportService struct {
	notionService *NotionService
	slackService  *SlackService
	logger        *logger.Logger
	channel       string
}

func NewWeeklyReportService(notionService *NotionService, slackService *SlackService, logger *logger.Logger, channel string) *WeeklyReportService {
	return &WeeklyReportService{
		notionService: notionService,
		slackService:  slackService,
		logger:        logger,
		channel:       channel,
	}
}

type WeeklyStats struct {
	TotalBugs   int
	JagoApp     int
	JagoanApp   int
	DepotPortal int
	Service     int
	StartDate   time.Time
	EndDate     time.Time
}

// StartWeeklyReports starts the weekly report scheduler
func (s *WeeklyReportService) StartWeeklyReports() {
	s.logger.Info("Weekly report scheduler started", map[string]interface{}{
		"channel": s.channel,
	})

	// Run every Friday at 5 PM
	go func() {
		for {
			now := time.Now()

			// Calculate next Friday 5 PM
			daysUntilFriday := (5 - int(now.Weekday()) + 7) % 7
			if daysUntilFriday == 0 && now.Hour() >= 17 {
				daysUntilFriday = 7 // Next week if already past 5 PM Friday
			}

			nextFriday := now.AddDate(0, 0, daysUntilFriday)
			nextReport := time.Date(nextFriday.Year(), nextFriday.Month(), nextFriday.Day(), 17, 0, 0, 0, now.Location())

			if nextReport.Before(now) {
				nextReport = nextReport.AddDate(0, 0, 7)
			}

			duration := nextReport.Sub(now)
			s.logger.Info("Next weekly report scheduled", map[string]interface{}{
				"nextReport": nextReport.Format("2006-01-02 15:04:05"),
				"duration":   duration.String(),
			})

			time.Sleep(duration)

			// Generate and send report
			s.GenerateAndSendReport()
		}
	}()
}

// GenerateAndSendReport generates weekly bug statistics and sends to Slack
func (s *WeeklyReportService) GenerateAndSendReport() {
	s.logger.Flow("WEEKLY_REPORT", "Generating weekly bug report", nil)

	// Get this week's date range (Monday to Friday)
	now := time.Now()
	weekday := int(now.Weekday())

	// Calculate Monday of this week
	daysFromMonday := weekday - 1
	if weekday == 0 { // Sunday
		daysFromMonday = 6
	}
	monday := now.AddDate(0, 0, -daysFromMonday)
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())

	// Friday is 4 days after Monday
	friday := monday.AddDate(0, 0, 4)
	friday = time.Date(friday.Year(), friday.Month(), friday.Day(), 23, 59, 59, 0, friday.Location())

	// Query Notion for bugs created this week
	stats, err := s.getWeeklyStats(monday, friday)
	if err != nil {
		s.logger.Error("Failed to get weekly stats", err, nil)
		return
	}

	// Send report to Slack
	s.sendWeeklyReport(stats)
}

func (s *WeeklyReportService) getWeeklyStats(startDate, endDate time.Time) (*WeeklyStats, error) {
	s.logger.Info("Querying Notion for weekly bugs", map[string]interface{}{
		"startDate": startDate.Format("2006-01-02"),
		"endDate":   endDate.Format("2006-01-02"),
	})

	// Query Notion database for bugs created this week
	query, err := s.notionService.client.Database.Query(context.Background(), s.notionService.databaseID, &notionapi.DatabaseQueryRequest{
		Filter: &notionapi.PropertyFilter{
			Property: "Created time",
			Date: &notionapi.DateFilterCondition{
				OnOrAfter: (*notionapi.Date)(&startDate),
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query Notion: %w", err)
	}

	stats := &WeeklyStats{
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Count bugs by app
	for _, page := range query.Results {
		// Check if created this week
		createdTime := page.CreatedTime
		if createdTime.Before(startDate) || createdTime.After(endDate) {
			continue
		}

		// Get tags
		tags := extractMultiSelect(page.Properties, "Tags")

		// Only count if has "Bug" tag
		if !containsIgnoreCase(tags, "bug") {
			continue
		}

		stats.TotalBugs++

		// Count by app
		for _, tag := range tags {
			switch tag {
			case "Jago App":
				stats.JagoApp++
			case "Jagoan App":
				stats.JagoanApp++
			case "Depot Portal":
				stats.DepotPortal++
			case "Service":
				stats.Service++
			}
		}
	}

	s.logger.Success("Weekly stats calculated", map[string]interface{}{
		"totalBugs":   stats.TotalBugs,
		"jagoApp":     stats.JagoApp,
		"jagoanApp":   stats.JagoanApp,
		"depotPortal": stats.DepotPortal,
		"service":     stats.Service,
	})

	return stats, nil
}

func (s *WeeklyReportService) sendWeeklyReport(stats *WeeklyStats) {
	s.logger.Flow("WEEKLY_REPORT", "Sending weekly report to Slack", map[string]interface{}{
		"channel": s.channel,
	})

	dateRange := fmt.Sprintf("%s - %s",
		stats.StartDate.Format("Mon, Jan 2"),
		stats.EndDate.Format("Mon, Jan 2, 2006"))

	blocks := []slack.Block{
		slack.NewHeaderBlock(&slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "📊 Weekly Bug Report",
		}),
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*Week:* %s\n*Total Bugs Found:* *%d*", dateRange, stats.TotalBugs),
			},
			nil,
			nil,
		),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: "*Breakdown by App:*",
			},
			nil,
			nil,
		),
		slack.NewSectionBlock(
			nil,
			[]*slack.TextBlockObject{
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Jago App:*\n%d bugs", stats.JagoApp)},
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Jagoan App:*\n%d bugs", stats.JagoanApp)},
			},
			nil,
		),
		slack.NewSectionBlock(
			nil,
			[]*slack.TextBlockObject{
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Depot Portal:*\n%d bugs", stats.DepotPortal)},
				{Type: slack.MarkdownType, Text: fmt.Sprintf("*Service/Backend:*\n%d bugs", stats.Service)},
			},
			nil,
		),
	}

	if err := s.slackService.PostMessage(s.channel, fmt.Sprintf("📊 Weekly Bug Report: %d bugs found this week", stats.TotalBugs), blocks); err != nil {
		s.logger.Error("Failed to send weekly report", err, nil)
	} else {
		s.logger.Success("Weekly report sent successfully", nil)
	}
}
