package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/plugin"
)

// PersistentSubscription represents a subscription that can be persisted
type PersistentSubscription struct {
	ID             string                 `json:"id"`
	ProviderID     string                 `json:"provider_id"`
	Params         map[string]interface{} `json:"params"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdateInterval time.Duration          `json:"update_interval"`
	Active         bool                   `json:"active"`
	AutoRestore    bool                   `json:"auto_restore"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// SubscriptionPersistence handles subscription persistence and recovery
type SubscriptionPersistence struct {
	dataDir         string
	subscriptionMgr *SubscriptionManager
	mu              sync.RWMutex
	autoSave        bool
	saveInterval    time.Duration
	stopChan        chan struct{}
}

// PersistenceConfig configuration for subscription persistence
type PersistenceConfig struct {
	DataDir      string        `json:"data_dir"`
	AutoSave     bool          `json:"auto_save"`
	SaveInterval time.Duration `json:"save_interval"`
}

// NewSubscriptionPersistence creates a new subscription persistence manager
func NewSubscriptionPersistence(config PersistenceConfig, subscriptionMgr *SubscriptionManager) (*SubscriptionPersistence, error) {
	if config.DataDir == "" {
		config.DataDir = "./data/subscriptions"
	}

	if config.SaveInterval == 0 {
		config.SaveInterval = 5 * time.Minute
	}

	// Ensure data directory exists
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	sp := &SubscriptionPersistence{
		dataDir:         config.DataDir,
		subscriptionMgr: subscriptionMgr,
		autoSave:        config.AutoSave,
		saveInterval:    config.SaveInterval,
		stopChan:        make(chan struct{}),
	}

	if config.AutoSave {
		go sp.autoSaveLoop()
	}

	return sp, nil
}

// SaveSubscriptions saves all current subscriptions to disk
func (sp *SubscriptionPersistence) SaveSubscriptions() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	subscriptions := sp.subscriptionMgr.ListSubscriptions()
	persistentSubs := make([]PersistentSubscription, 0, len(subscriptions))

	for _, sub := range subscriptions {
		persistentSub := PersistentSubscription{
			ID:             sub.ID,
			ProviderID:     sub.ProviderID,
			Params:         sub.Params,
			CreatedAt:      sub.CreatedAt,
			UpdateInterval: sub.UpdateInterval,
			Active:         sub.Active,
			AutoRestore:    true, // Mark for auto-restore
			Metadata: map[string]interface{}{
				"update_count": sub.UpdateCount,
				"error_count":  sub.ErrorCount,
				"last_error":   sub.LastError,
			},
		}
		persistentSubs = append(persistentSubs, persistentSub)
	}

	// Save to JSON file
	filename := filepath.Join(sp.dataDir, "subscriptions.json")
	data, err := json.MarshalIndent(persistentSubs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal subscriptions: %w", err)
	}

	// Write to temporary file first, then rename for atomicity
	tempFile := filename + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := os.Rename(tempFile, filename); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// LoadSubscriptions loads subscriptions from disk
func (sp *SubscriptionPersistence) LoadSubscriptions() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	filename := filepath.Join(sp.dataDir, "subscriptions.json")
	
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// No saved subscriptions, that's okay
		return nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read subscriptions file: %w", err)
	}

	var persistentSubs []PersistentSubscription
	if err := json.Unmarshal(data, &persistentSubs); err != nil {
		return fmt.Errorf("failed to unmarshal subscriptions: %w", err)
	}

	// Restore subscriptions that are marked for auto-restore
	restoredCount := 0
	for _, persistentSub := range persistentSubs {
		if persistentSub.AutoRestore && persistentSub.Active {
			// Create a placeholder callback for restored subscriptions
			callback := func(data interface{}, err error) {
				// This is a placeholder callback for restored subscriptions
				// The actual callback will need to be re-registered by the client
			}

			_, err := sp.subscriptionMgr.Subscribe(
				persistentSub.ProviderID,
				persistentSub.Params,
				callback,
			)

			if err != nil {
				// Log error but continue with other subscriptions
				continue
			}

			restoredCount++
		}
	}

	return nil
}

// SaveSubscription saves a single subscription
func (sp *SubscriptionPersistence) SaveSubscription(subscriptionID string) error {
	subscription, err := sp.subscriptionMgr.GetSubscription(plugin.SubscriptionID(subscriptionID))
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	persistentSub := PersistentSubscription{
		ID:             subscription.ID,
		ProviderID:     subscription.ProviderID,
		Params:         subscription.Params,
		CreatedAt:      subscription.CreatedAt,
		UpdateInterval: subscription.UpdateInterval,
		Active:         subscription.Active,
		AutoRestore:    true,
		Metadata: map[string]interface{}{
			"update_count": subscription.UpdateCount,
			"error_count":  subscription.ErrorCount,
			"last_error":   subscription.LastError,
		},
	}

	// Save individual subscription file
	filename := filepath.Join(sp.dataDir, fmt.Sprintf("subscription_%s.json", subscriptionID))
	data, err := json.MarshalIndent(persistentSub, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal subscription: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write subscription file: %w", err)
	}

	return nil
}

// DeleteSubscription removes a persisted subscription
func (sp *SubscriptionPersistence) DeleteSubscription(subscriptionID string) error {
	filename := filepath.Join(sp.dataDir, fmt.Sprintf("subscription_%s.json", subscriptionID))
	
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete subscription file: %w", err)
	}

	return nil
}

// BackupSubscriptions creates a backup of all subscriptions
func (sp *SubscriptionPersistence) BackupSubscriptions() (string, error) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	timestamp := time.Now().Format("20060102_150405")
	backupDir := filepath.Join(sp.dataDir, "backups")
	
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	backupFile := filepath.Join(backupDir, fmt.Sprintf("subscriptions_backup_%s.json", timestamp))

	subscriptions := sp.subscriptionMgr.ListSubscriptions()
	data, err := json.MarshalIndent(subscriptions, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal backup data: %w", err)
	}

	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return backupFile, nil
}

// RestoreFromBackup restores subscriptions from a backup file
func (sp *SubscriptionPersistence) RestoreFromBackup(backupFile string) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	data, err := os.ReadFile(backupFile)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	var persistentSubs []PersistentSubscription
	if err := json.Unmarshal(data, &persistentSubs); err != nil {
		return fmt.Errorf("failed to unmarshal backup data: %w", err)
	}

	// Clear existing subscriptions first
	for id := range sp.subscriptionMgr.ListSubscriptions() {
		sp.subscriptionMgr.Unsubscribe(plugin.SubscriptionID(id))
	}

	// Restore subscriptions from backup
	restoredCount := 0
	for _, persistentSub := range persistentSubs {
		if persistentSub.Active {
			callback := func(data interface{}, err error) {
				// Placeholder callback for restored subscriptions
			}

			_, err := sp.subscriptionMgr.Subscribe(
				persistentSub.ProviderID,
				persistentSub.Params,
				callback,
			)

			if err != nil {
				continue
			}

			restoredCount++
		}
	}

	return nil
}

// GetBackupList returns a list of available backup files
func (sp *SubscriptionPersistence) GetBackupList() ([]string, error) {
	backupDir := filepath.Join(sp.dataDir, "backups")
	
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := filepath.Glob(filepath.Join(backupDir, "subscriptions_backup_*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list backup files: %w", err)
	}

	return files, nil
}

// CleanupOldBackups removes backup files older than the specified duration
func (sp *SubscriptionPersistence) CleanupOldBackups(maxAge time.Duration) error {
	backupFiles, err := sp.GetBackupList()
	if err != nil {
		return err
	}

	cutoffTime := time.Now().Add(-maxAge)
	removedCount := 0

	for _, file := range backupFiles {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoffTime) {
			if err := os.Remove(file); err != nil {
				// Log error but continue
				continue
			}
			removedCount++
		}
	}

	return nil
}

// GetPersistenceStats returns statistics about persistence operations
func (sp *SubscriptionPersistence) GetPersistenceStats() map[string]interface{} {
	backupFiles, _ := sp.GetBackupList()
	
	stats := map[string]interface{}{
		"data_directory":      sp.dataDir,
		"auto_save_enabled":   sp.autoSave,
		"save_interval":       sp.saveInterval.String(),
		"backup_count":        len(backupFiles),
	}

	// Add file size information
	if info, err := os.Stat(filepath.Join(sp.dataDir, "subscriptions.json")); err == nil {
		stats["subscriptions_file_size"] = info.Size()
		stats["last_save_time"] = info.ModTime()
	}

	return stats
}

// Stop stops the auto-save loop
func (sp *SubscriptionPersistence) Stop() {
	close(sp.stopChan)
}

// autoSaveLoop runs the automatic save process
func (sp *SubscriptionPersistence) autoSaveLoop() {
	ticker := time.NewTicker(sp.saveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sp.stopChan:
			return
		case <-ticker.C:
			if err := sp.SaveSubscriptions(); err != nil {
				// Log error but continue
			}
		}
	}
}

// SubscriptionRecovery handles recovery of failed subscriptions
type SubscriptionRecovery struct {
	subscriptionMgr *SubscriptionManager
	persistence     *SubscriptionPersistence
	maxRetries      int
	retryDelay      time.Duration
}

// NewSubscriptionRecovery creates a new subscription recovery manager
func NewSubscriptionRecovery(subscriptionMgr *SubscriptionManager, persistence *SubscriptionPersistence) *SubscriptionRecovery {
	return &SubscriptionRecovery{
		subscriptionMgr: subscriptionMgr,
		persistence:     persistence,
		maxRetries:      3,
		retryDelay:      30 * time.Second,
	}
}

// RecoverFailedSubscriptions attempts to recover failed subscriptions
func (sr *SubscriptionRecovery) RecoverFailedSubscriptions(ctx context.Context) error {
	subscriptions := sr.subscriptionMgr.ListSubscriptions()
	
	for _, sub := range subscriptions {
		if !sub.Active && sub.ErrorCount > 0 {
			// Try to recover the subscription
			if err := sr.recoverSubscription(ctx, sub.ID); err != nil {
				// Log error but continue with other subscriptions
				continue
			}
		}
	}

	return nil
}

// recoverSubscription attempts to recover a single subscription
func (sr *SubscriptionRecovery) recoverSubscription(ctx context.Context, subscriptionID string) error {
	subscription, err := sr.subscriptionMgr.GetSubscription(plugin.SubscriptionID(subscriptionID))
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Reset error count and reactivate
	subscription.ErrorCount = 0
	subscription.LastError = ""
	subscription.Active = true

	// Try to test the subscription with a sample query
	switch subscription.ProviderID {
	case "prometheus-metrics", "node-metrics", "job-metrics":
		// Test with a simple query
		testParams := make(map[string]interface{})
		for k, v := range subscription.Params {
			testParams[k] = v
		}
		testParams["query"] = "up"
		
		_, err := sr.subscriptionMgr.getData(ctx, subscription.ProviderID, testParams)
		if err != nil {
			return fmt.Errorf("subscription test failed: %w", err)
		}
	}

	return nil
}