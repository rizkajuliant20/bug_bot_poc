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
	Severity          string   `json:"severity"`
	Category          string   `json:"category"`
	Priority          string   `json:"priority"`
	Platform          []string `json:"platform"`
	Team              string   `json:"team"`
	Precondition      string   `json:"precondition"`
	StepsToReproduce  string   `json:"stepsToReproduce"`
	ActualResult      string   `json:"actualResult"`
	ExpectedResult    string   `json:"expectedResult"`
	RootCause         string   `json:"rootCause"`
	SuggestedFix      string   `json:"suggestedFix"`
	AffectedComponents []string `json:"affectedComponents"`
	Tags              []string `json:"tags"`
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

Provide your analysis in this exact JSON structure:
{
  "severity": "critical|high|medium|low",
  "category": "Backend|Frontend|Database|API|UI/UX|Performance|Security|Other",
  "priority": "P0|P1|P2|P3",
  "platform": ["iOS", "Android", "Web", "Backend"],
  "team": "Eng|QA|Product|Design",
  "precondition": "Conditions before the bug occurs",
  "stepsToReproduce": "1. Step one\n2. Step two\n3. Step three",
  "actualResult": "What actually happens",
  "expectedResult": "What should happen",
  "rootCause": "Brief analysis of the likely root cause",
  "suggestedFix": "Recommended solution or next steps",
  "affectedComponents": ["component1", "component2"],
  "tags": ["bug", "feature", "enhancement", etc]
}

Extract as much structured information as possible from the bug report. If information is missing, use reasonable defaults.`, bugDescription, conversation)

	startTime := time.Now()
	s.logger.Info("Calling OpenAI API for diagnosis", nil)

	resp, err := s.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
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

	s.logger.Success("Bug diagnosis completed", map[string]interface{}{
		"duration": duration.Milliseconds(),
		"severity": diagnosis.Severity,
		"category": diagnosis.Category,
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
			Model: openai.GPT3Dot5Turbo,
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
		"descriptionLength": len(bugDescription),
		"threadMessageCount": len(threadMessages),
	})

	appName := "App"
	var detectedAppName string

	// Combine all text for app detection
	allText := strings.ToLower(bugDescription)
	for _, msg := range threadMessages {
		allText += " " + strings.ToLower(msg.Text)
	}

	// Detect app name
	if strings.Contains(allText, "jago app") || strings.Contains(allText, "jagoapp") {
		appName = "Jago App"
		detectedAppName = "Jago App"
		s.logger.Info("Detected app name from text", map[string]interface{}{"appName": "Jago App"})
	} else if strings.Contains(allText, "jagoan app") || strings.Contains(allText, "jagoanapp") {
		appName = "Jagoan App"
		detectedAppName = "Jagoan App"
		s.logger.Info("Detected app name from text", map[string]interface{}{"appName": "Jagoan App"})
	} else if strings.Contains(allText, "depot portal") || strings.Contains(allText, "depot") {
		appName = "Depot Portal"
		detectedAppName = "Depot Portal"
	} else if strings.Contains(allText, "service") || strings.Contains(allText, "backend") || strings.Contains(allText, "api") {
		appName = "Service"
		detectedAppName = "Service"
	} else if len(diagnosis.Platform) > 0 {
		for _, platform := range diagnosis.Platform {
			if strings.Contains(strings.ToLower(platform), "android") || strings.Contains(strings.ToLower(platform), "ios") {
				appName = "Jagoan App"
				detectedAppName = "Jagoan App"
				break
			}
		}
	}

	prompt := fmt.Sprintf(`Create a concise bug ticket title in this exact format:
[Bug][%s] Brief description of the issue

Bug: %s
Category: %s

Keep the description brief and clear (max 8 words after app name). Focus on the core issue.`, appName, bugDescription, diagnosis.Category)

	startTime := time.Now()
	s.logger.Info("Calling OpenAI API for bug title generation", map[string]interface{}{"appName": appName})

	resp, err := s.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
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
			Temperature: 0.3,
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
	for i, msg := range messages {
		parts = append(parts, fmt.Sprintf("Message %d: %s", i+1, msg.Text))
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
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
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
	
	prompt := fmt.Sprintf(`Analyze this bug report and thread discussion to detect if there are MULTIPLE separate issues mentioned.

Bug Description: %s

Thread Discussion:
%s

Carefully read through all messages and identify:
1. How many DISTINCT issues/bugs are mentioned?
2. For each issue, provide a brief title and description

Respond in JSON format:
{
  "issueCount": <number of distinct issues found>,
  "issues": [
    {
      "title": "Brief title of issue 1",
      "description": "Short description of what the issue is",
      "severity": "critical|high|medium|low",
      "category": "Backend|Frontend|Database|API|UI/UX|Performance|Security|Other"
    }
  ]
}

IMPORTANT: 
- If only ONE issue is discussed, return issueCount: 1 with that single issue
- If multiple SEPARATE issues are mentioned, list all of them
- Don't split one issue into multiple parts
- Focus on truly distinct problems`, bugDescription, conversation)

	startTime := time.Now()
	s.logger.Info("Calling OpenAI API for multi-issue detection", nil)

	resp, err := s.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a technical analyst. Identify distinct issues from bug reports. Always respond with valid JSON only.",
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
		s.logger.Error("Multi-issue detection failed", err, nil)
		return nil, fmt.Errorf("failed to detect multiple issues: %w", err)
	}

	duration := time.Since(startTime)
	
	var analysis MultiIssueAnalysis
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &analysis); err != nil {
		s.logger.Error("Failed to parse multi-issue JSON", err, nil)
		return nil, fmt.Errorf("failed to parse multi-issue analysis: %w", err)
	}

	s.logger.Success("Multi-issue detection completed", map[string]interface{}{
		"duration":   duration.Milliseconds(),
		"issueCount": analysis.IssueCount,
	})

	return &analysis, nil
}
