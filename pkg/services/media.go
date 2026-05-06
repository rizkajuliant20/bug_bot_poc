package services

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/slack-go/slack"
)

// ExtractMediaFiles extracts all media files (images, videos, screenshots) from Slack messages
func (s *SlackService) ExtractMediaFiles(messages []slack.Message) []MediaFile {
	var mediaFiles []MediaFile

	for _, msg := range messages {
		// Check for file attachments
		for _, file := range msg.Files {
			if isMediaFile(file.Mimetype) {
				mediaFiles = append(mediaFiles, MediaFile{
					URL:      file.URLPrivateDownload,
					Name:     file.Name,
					MimeType: file.Mimetype,
					IsImage:  isImage(file.Mimetype),
					IsVideo:  isVideo(file.Mimetype),
				})
			}
		}
	}

	return mediaFiles
}

// DownloadMediaFile downloads a media file from Slack
func (s *SlackService) DownloadMediaFile(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add Slack authorization header
	req.Header.Add("Authorization", "Bearer "+s.botToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	return data, nil
}

func isMediaFile(mimeType string) bool {
	return isImage(mimeType) || isVideo(mimeType)
}

func isImage(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

func isVideo(mimeType string) bool {
	return strings.HasPrefix(mimeType, "video/")
}
