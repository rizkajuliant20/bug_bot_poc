package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// FileUploadResponse represents Notion's file upload API response
type FileUploadResponse struct {
	ID        string `json:"id"`
	UploadURL string `json:"upload_url"`
	Status    string `json:"status"`
}

// UploadFileToNotion uploads a file to Notion and returns the file upload ID
func (s *NotionService) UploadFileToNotion(fileData []byte, filename string) (string, error) {
	// Step 1: Create file upload object with empty body
	createURL := "https://api.notion.com/v1/file_uploads"

	req, err := http.NewRequest("POST", createURL, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create file upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create file upload: status %d, body: %s", resp.StatusCode, string(body))
	}

	var uploadResp FileUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Step 2: Upload file contents using multipart/form-data
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = part.Write(fileData)
	if err != nil {
		return "", fmt.Errorf("failed to write file data: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	uploadReq, err := http.NewRequest("POST", uploadResp.UploadURL, &requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to create upload request: %w", err)
	}

	uploadReq.Header.Set("Authorization", "Bearer "+s.apiToken)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("Notion-Version", "2022-06-28")

	uploadResp2, err := client.Do(uploadReq)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	defer uploadResp2.Body.Close()

	if uploadResp2.StatusCode != http.StatusOK && uploadResp2.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(uploadResp2.Body)
		return "", fmt.Errorf("failed to upload file: status %d, body: %s", uploadResp2.StatusCode, string(body))
	}

	s.logger.Info("File uploaded to Notion successfully", map[string]interface{}{
		"filename":     filename,
		"fileUploadId": uploadResp.ID,
	})

	return uploadResp.ID, nil
}

// AppendMediaBlocksToPage uploads media files and appends them to a Notion page
func (s *NotionService) AppendMediaBlocksToPage(pageID string, mediaFiles []MediaFile, slackService *SlackService) error {
	if len(mediaFiles) == 0 {
		return nil
	}

	var blockChildren []map[string]interface{}

	for _, media := range mediaFiles {
		// Download file from Slack
		fileData, err := slackService.DownloadMediaFile(media.URL)
		if err != nil {
			s.logger.Error("Failed to download media from Slack", err, map[string]interface{}{
				"url": media.URL,
			})
			continue
		}

		// Upload to Notion
		fileUploadID, err := s.UploadFileToNotion(fileData, media.Name)
		if err != nil {
			s.logger.Error("Failed to upload media to Notion", err, map[string]interface{}{
				"filename": media.Name,
			})
			continue
		}

		// Create block JSON
		var blockType string
		if media.IsImage {
			blockType = "image"
		} else if media.IsVideo {
			blockType = "video"
		} else {
			continue
		}

		block := map[string]interface{}{
			"object": "block",
			"type":   blockType,
			blockType: map[string]interface{}{
				"type": "file_upload",
				"file_upload": map[string]interface{}{
					"id": fileUploadID,
				},
			},
		}

		blockChildren = append(blockChildren, block)
	}

	if len(blockChildren) == 0 {
		return nil
	}

	// Append blocks to page using raw API
	url := fmt.Sprintf("https://api.notion.com/v1/blocks/%s/children", pageID)
	payload := map[string]interface{}{
		"children": blockChildren,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to append blocks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to append blocks: status %d, body: %s", resp.StatusCode, string(body))
	}

	s.logger.Success("Media blocks appended to Notion page", map[string]interface{}{
		"pageID":     pageID,
		"mediaCount": len(blockChildren),
	})

	return nil
}
