package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Slack struct {
		BotToken            string
		SigningSecret       string
		AppToken            string
		BugTrackingChannel  string
	}
	Notion struct {
		APIKey       string
		DatabaseID   string
	}
	OpenAI struct {
		APIKey string
	}
	Port string
}

func Load() (*Config, error) {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		// .env file is optional in production
		fmt.Println("No .env file found, using environment variables")
	}

	cfg := &Config{}

	// Slack configuration
	cfg.Slack.BotToken = os.Getenv("SLACK_BOT_TOKEN")
	cfg.Slack.SigningSecret = os.Getenv("SLACK_SIGNING_SECRET")
	cfg.Slack.AppToken = os.Getenv("SLACK_APP_TOKEN")
	cfg.Slack.BugTrackingChannel = os.Getenv("SLACK_BUG_TRACKING_CHANNEL")

	// Notion configuration
	cfg.Notion.APIKey = os.Getenv("NOTION_API_KEY")
	cfg.Notion.DatabaseID = os.Getenv("NOTION_DATABASE_ID")

	// OpenAI configuration
	cfg.OpenAI.APIKey = os.Getenv("OPENAI_API_KEY")

	// Server configuration
	cfg.Port = os.Getenv("PORT")
	if cfg.Port == "" {
		cfg.Port = "3000"
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Slack.BotToken == "" {
		return fmt.Errorf("SLACK_BOT_TOKEN is required")
	}
	if c.Slack.SigningSecret == "" {
		return fmt.Errorf("SLACK_SIGNING_SECRET is required")
	}
	if c.Slack.AppToken == "" {
		return fmt.Errorf("SLACK_APP_TOKEN is required")
	}
	if c.Notion.APIKey == "" {
		return fmt.Errorf("NOTION_API_KEY is required")
	}
	if c.Notion.DatabaseID == "" {
		return fmt.Errorf("NOTION_DATABASE_ID is required")
	}
	if c.OpenAI.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}
	return nil
}
