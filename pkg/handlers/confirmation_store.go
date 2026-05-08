package handlers

import (
	"sync"
	"time"

	"github.com/rizkajuliant20/bug-bot/pkg/services"
	"github.com/slack-go/slack"
)

// ConfirmationData stores bug analysis data waiting for user confirmation
type ConfirmationData struct {
	BugDescription string
	Diagnosis      *services.BugDiagnosis
	Title          string
	AppName        string
	Reporter       string
	SlackThreadURL string
	ThreadMessages []slack.Message
	ThreadSummary  string
	MediaFiles     []services.MediaFile
	TeamID         string
	Channel        string
	ThreadTS       string
	OriginalTS     string
	CreatedAt      time.Time
}

var (
	confirmationStore = make(map[string]*ConfirmationData)
	confirmationMutex sync.RWMutex
)

// StoreConfirmationData stores confirmation data with expiry
func StoreConfirmationData(id string, data *ConfirmationData) {
	confirmationMutex.Lock()
	defer confirmationMutex.Unlock()

	data.CreatedAt = time.Now()
	confirmationStore[id] = data

	// Debug: log store operation
	println("DEBUG: Stored confirmation data:", id, "Total stored:", len(confirmationStore))

	// No auto-cleanup - data stays until explicitly deleted
}

// GetConfirmationData retrieves confirmation data
func GetConfirmationData(id string) (*ConfirmationData, bool) {
	confirmationMutex.RLock()
	defer confirmationMutex.RUnlock()

	data, exists := confirmationStore[id]

	// Debug: log retrieval attempt
	println("DEBUG: Get confirmation data:", id, "Exists:", exists, "Total stored:", len(confirmationStore))
	if !exists {
		println("DEBUG: Available keys:")
		for key := range confirmationStore {
			println("  -", key)
		}
	}

	return data, exists
}

// DeleteConfirmationData removes confirmation data
func DeleteConfirmationData(id string) {
	confirmationMutex.Lock()
	defer confirmationMutex.Unlock()

	delete(confirmationStore, id)
}

// ClearConfirmationStore clears all confirmation data (used on bot startup)
func ClearConfirmationStore() {
	confirmationMutex.Lock()
	defer confirmationMutex.Unlock()

	confirmationStore = make(map[string]*ConfirmationData)
}
