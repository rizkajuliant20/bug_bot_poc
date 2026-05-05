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

Provide your analysis in this exact JSON structure:
{
  "severity": "critical|high|medium|low",
  "category": "Backend|Frontend|Database|API|UI/UX|Performance|Security|Other",
  "priority": "Low|Medium|High",
  "platform": ["Android", "iOS", "Website", "Backend"],
  "team": "Eng|Data|Design|Product",
  "precondition": "Conditions before the bug occurs",
  "stepsToReproduce": "1. Step one\n2. Step two\n3. Step three",
  "actualResult": "What actually happens",
  "expectedResult": "What should happen",
  "rootCause": "Brief analysis of the likely root cause",
  "suggestedFix": "Recommended solution or next steps",
  "affectedComponents": ["component1", "component2"],
  "tags": ["Bug", "Jago App"]
}

IMPORTANT RULES:
- Priority must be exactly: "Low", "Medium", or "High" (case-sensitive)
- Platform options: "Android", "iOS", "Website", "Backend"
- Team must be exactly: "Eng", "Data", "Design", or "Product"
- Tags MUST contain EXACTLY 2 items: "Bug" (mandatory) + ONE app name
- App name options: "Jago App", "Jagoan App", "Depot Portal", or "Service"
- NEVER include other tags like "Tech Debt", "Design System", "UI/UX", "Website", etc.
- Example valid tags: ["Bug", "Jago App"] or ["Bug", "Service"]

APP DETECTION RULES (CRITICAL):
- If text mentions "jagoan", "aplikasi jagoan", "jagoan app" → Use "Jagoan App"
- If text mentions "jago", "aplikasi jago", "jago app" (but NOT jagoan) → Use "Jago App"
- If text mentions "depot", "depot portal" → Use "Depot Portal"
- If text mentions "service", "backend", "api" → Use "Service"
- Check Jagoan FIRST before Jago (Jagoan is more specific)

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
2. For each issue, provide a brief title and description

Respond in JSON format:
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
- Check for "jagoan", "aplikasi jagoan" → Use "Jagoan App"
- Check for "jago", "aplikasi jago" → Use "Jago App"
- Check for "depot" → Use "Depot Portal"
- Check for "service", "backend", "api" → Use "Service"`, bugDescription, conversation)

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
