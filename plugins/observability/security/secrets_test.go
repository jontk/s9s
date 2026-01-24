package security

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSecretsManager(t *testing.T) {
	ctx := context.Background()

	// Use temporary directory for testing
	tempDir := t.TempDir()

	config := SecretConfig{
		StorageDir:         tempDir,
		EncryptAtRest:      true,
		MasterKeySource:    SecretSourceEnvironment,
		MasterKeyEnv:       "TEST_MASTER_KEY",
		EnableRotation:     false,
		RequireEncryption:  true,
		AllowInlineSecrets: true,
	}

	// Set test master key
	_ = os.Setenv("TEST_MASTER_KEY", "dGVzdC1tYXN0ZXIta2V5LTEyMzQ1Njc4OTBhYmNkZWY=") // base64 encoded 32-byte key
	defer func() { _ = os.Unsetenv("TEST_MASTER_KEY") }()

	sm, err := NewSecretsManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}
	defer func() { _ = sm.Stop() }()

	if sm == nil {
		t.Fatal("Expected secrets manager instance, got nil")
	}

	// Check that storage directory was created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Expected storage directory to be created")
	}

	// Verify master key was initialized
	if len(sm.masterKey) == 0 {
		t.Error("Expected master key to be initialized")
	}
}

func TestStoreAndRetrieveSecret(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := SecretConfig{
		StorageDir:         tempDir,
		EncryptAtRest:      true,
		MasterKeySource:    SecretSourceEnvironment,
		MasterKeyEnv:       "TEST_MASTER_KEY",
		RequireEncryption:  true,
		AllowInlineSecrets: true,
	}

	_ = os.Setenv("TEST_MASTER_KEY", "dGVzdC1tYXN0ZXIta2V5LTEyMzQ1Njc4OTBhYmNkZWY=")
	defer func() { _ = os.Unsetenv("TEST_MASTER_KEY") }()

	sm, err := NewSecretsManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}
	defer func() { _ = sm.Stop() }()

	// Store a secret
	err = sm.StoreSecret("test-token", SecretTypeAPIToken, "super-secret-api-key-12345", SecretSourceInline, map[string]string{
		"service": "prometheus",
		"env":     "test",
	})
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

	// Retrieve the secret
	secret, err := sm.RetrieveSecret("test-token")
	if err != nil {
		t.Fatalf("Failed to retrieve secret: %v", err)
	}

	if secret.Name != "test-token" {
		t.Errorf("Expected secret name 'test-token', got '%s'", secret.Name)
	}

	if secret.Type != SecretTypeAPIToken {
		t.Errorf("Expected secret type '%s', got '%s'", SecretTypeAPIToken, secret.Type)
	}

	if secret.Value != "super-secret-api-key-12345" {
		t.Errorf("Expected secret value to be preserved")
	}

	if secret.Source != SecretSourceInline {
		t.Errorf("Expected secret source '%s', got '%s'", SecretSourceInline, secret.Source)
	}

	if secret.Metadata["service"] != "prometheus" {
		t.Errorf("Expected metadata service 'prometheus', got '%s'", secret.Metadata["service"])
	}

	if !secret.Encrypted {
		t.Error("Expected secret to be marked as encrypted")
	}
}

func TestGetSecretValue(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := SecretConfig{
		StorageDir:         tempDir,
		EncryptAtRest:      false, // Test without encryption for simplicity
		MasterKeySource:    SecretSourceEnvironment,
		AllowInlineSecrets: true,
	}

	sm, err := NewSecretsManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}
	defer func() { _ = sm.Stop() }()

	// Store a secret
	err = sm.StoreSecret("test-token", SecretTypeAPIToken, "my-secret-value", SecretSourceInline, nil)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

	// Get just the value
	value, err := sm.GetSecretValue("test-token")
	if err != nil {
		t.Fatalf("Failed to get secret value: %v", err)
	}

	if value != "my-secret-value" {
		t.Errorf("Expected secret value 'my-secret-value', got '%s'", value)
	}
}

func TestDeleteSecret(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := SecretConfig{
		StorageDir:         tempDir,
		EncryptAtRest:      false,
		MasterKeySource:    SecretSourceEnvironment,
		AllowInlineSecrets: true,
	}

	sm, err := NewSecretsManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}
	defer func() { _ = sm.Stop() }()

	// Store a secret
	err = sm.StoreSecret("temp-secret", SecretTypeAPIToken, "temporary-value", SecretSourceInline, nil)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

	// Verify it exists
	_, err = sm.RetrieveSecret("temp-secret")
	if err != nil {
		t.Fatalf("Secret should exist before deletion: %v", err)
	}

	// Delete the secret
	err = sm.DeleteSecret("temp-secret")
	if err != nil {
		t.Fatalf("Failed to delete secret: %v", err)
	}

	// Verify it no longer exists
	_, err = sm.RetrieveSecret("temp-secret")
	if err == nil {
		t.Error("Expected secret to be deleted")
	}

	// Check that file was removed
	secretPath := filepath.Join(tempDir, "temp-secret.secret")
	if _, err := os.Stat(secretPath); !os.IsNotExist(err) {
		t.Error("Expected secret file to be removed")
	}
}

func TestListSecrets(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := SecretConfig{
		StorageDir:         tempDir,
		EncryptAtRest:      false,
		MasterKeySource:    SecretSourceEnvironment,
		AllowInlineSecrets: true,
	}

	sm, err := NewSecretsManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}
	defer func() { _ = sm.Stop() }()

	// Store multiple secrets
	secrets := map[string]struct {
		secretType SecretType
		value      string
	}{
		"api-token-1": {SecretTypeAPIToken, "test-api-token-123"},
		"api-token-2": {SecretTypeAPIToken, "another-api-token-456"},
		"basic-auth":  {SecretTypeBasicAuth, "username:password"},
	}

	for name, secret := range secrets {
		err = sm.StoreSecret(name, secret.secretType, secret.value, SecretSourceInline, nil)
		if err != nil {
			t.Fatalf("Failed to store secret '%s': %v", name, err)
		}
	}

	// List secrets
	refs := sm.ListSecrets()

	if len(refs) != 3 {
		t.Errorf("Expected 3 secret references, got %d", len(refs))
	}

	// Verify secret names are present but values are not
	secretNames := make(map[string]bool)
	for _, ref := range refs {
		secretNames[ref.Name] = true

		if ref.Type == "" {
			t.Errorf("Expected secret type to be set for '%s'", ref.Name)
		}

		if ref.Source == "" {
			t.Errorf("Expected secret source to be set for '%s'", ref.Name)
		}
	}

	for name := range secrets {
		if !secretNames[name] {
			t.Errorf("Expected secret '%s' in list", name)
		}
	}
}

func TestSecretRotation(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := SecretConfig{
		StorageDir:         tempDir,
		EncryptAtRest:      false,
		MasterKeySource:    SecretSourceEnvironment,
		EnableRotation:     true,
		RotationInterval:   1 * time.Hour,
		AllowInlineSecrets: true,
	}

	sm, err := NewSecretsManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}
	defer func() { _ = sm.Stop() }()

	// Store a rotatable secret
	err = sm.StoreSecret("rotatable-token", SecretTypeAPIToken, "original-value", SecretSourceInline, nil)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

	// Verify it's marked as rotatable
	secret, err := sm.RetrieveSecret("rotatable-token")
	if err != nil {
		t.Fatalf("Failed to retrieve secret: %v", err)
	}

	if !secret.Rotatable {
		t.Error("Expected API token to be rotatable")
	}

	if secret.ExpiresAt == nil {
		t.Error("Expected expiration time to be set for rotatable secret")
	}

	// Rotate the secret
	err = sm.RotateSecret("rotatable-token", "new-rotated-value")
	if err != nil {
		t.Fatalf("Failed to rotate secret: %v", err)
	}

	// Verify new value
	rotatedSecret, err := sm.RetrieveSecret("rotatable-token")
	if err != nil {
		t.Fatalf("Failed to retrieve rotated secret: %v", err)
	}

	if rotatedSecret.Value != "new-rotated-value" {
		t.Errorf("Expected rotated value 'new-rotated-value', got '%s'", rotatedSecret.Value)
	}

	if !rotatedSecret.UpdatedAt.After(secret.UpdatedAt) {
		t.Error("Expected updated time to be more recent after rotation")
	}
}

func TestSecretEncryption(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := SecretConfig{
		StorageDir:         tempDir,
		EncryptAtRest:      true,
		MasterKeySource:    SecretSourceEnvironment,
		MasterKeyEnv:       "TEST_MASTER_KEY",
		RequireEncryption:  true,
		AllowInlineSecrets: true,
	}

	_ = os.Setenv("TEST_MASTER_KEY", "dGVzdC1tYXN0ZXIta2V5LTEyMzQ1Njc4OTBhYmNkZWY=")
	defer func() { _ = os.Unsetenv("TEST_MASTER_KEY") }()

	sm, err := NewSecretsManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}
	defer func() { _ = sm.Stop() }()

	originalValue := "super-secret-value-to-encrypt"

	// Store encrypted secret
	err = sm.StoreSecret("encrypted-secret", SecretTypeAPIToken, originalValue, SecretSourceInline, nil)
	if err != nil {
		t.Fatalf("Failed to store encrypted secret: %v", err)
	}

	// Retrieve and verify
	secret, err := sm.RetrieveSecret("encrypted-secret")
	if err != nil {
		t.Fatalf("Failed to retrieve encrypted secret: %v", err)
	}

	if secret.Value != originalValue {
		t.Errorf("Expected decrypted value to match original")
	}

	if !secret.Encrypted {
		t.Error("Expected secret to be marked as encrypted")
	}

	// Verify the stored file is actually encrypted
	secretPath := filepath.Join(tempDir, "encrypted-secret.secret")
	// nolint:gosec // G304: secretPath is test fixture path in temp directory
	data, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("Failed to read secret file: %v", err)
	}

	// The file should not contain the original value in plaintext
	if contains(string(data), originalValue) {
		t.Error("Secret file should not contain plaintext value")
	}
}

func TestSecretValidation(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := SecretConfig{
		StorageDir:         tempDir,
		EncryptAtRest:      false,
		MasterKeySource:    SecretSourceEnvironment,
		AllowInlineSecrets: true,
	}

	sm, err := NewSecretsManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}
	defer func() { _ = sm.Stop() }()

	// Test API token validation
	testCases := []struct {
		name        string
		secretType  SecretType
		value       string
		shouldFail  bool
		description string
	}{
		{"valid-api-token", SecretTypeAPIToken, "valid-api-token-123", false, "valid API token"},
		{"short-api-token", SecretTypeAPIToken, "short", true, "API token too short"},
		{"valid-basic-auth", SecretTypeBasicAuth, "user:password", false, "valid basic auth"},
		{"invalid-basic-auth", SecretTypeBasicAuth, "invalid-format", true, "invalid basic auth format"},
		{"empty-username", SecretTypeBasicAuth, ":password", true, "empty username"},
		{"empty-password", SecretTypeBasicAuth, "user:", true, "empty password"},
		{"valid-encryption-key", SecretTypeEncryptionKey, "YWJjZGVmZ2hpams6bG1ub3BxcnN0dXZ3eHl6MTIzNDU2", false, "valid encryption key"},
		{"invalid-encryption-key", SecretTypeEncryptionKey, "not-base64!", true, "invalid base64 encryption key"},
		{"short-encryption-key", SecretTypeEncryptionKey, "c2hvcnQ=", true, "encryption key too short"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := sm.StoreSecret(tc.name, tc.secretType, tc.value, SecretSourceInline, nil)

			if tc.shouldFail && err == nil {
				t.Errorf("Expected validation to fail for %s", tc.description)
			}

			if !tc.shouldFail && err != nil {
				t.Errorf("Expected validation to pass for %s, got error: %v", tc.description, err)
			}
		})
	}
}

func TestSecretExpiration(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := SecretConfig{
		StorageDir:         tempDir,
		EncryptAtRest:      false,
		MasterKeySource:    SecretSourceEnvironment,
		EnableRotation:     true,
		RotationInterval:   100 * time.Millisecond, // Very short for testing
		AllowInlineSecrets: true,
	}

	sm, err := NewSecretsManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}
	defer func() { _ = sm.Stop() }()

	// Store a secret that will expire quickly
	err = sm.StoreSecret("expiring-secret", SecretTypeAPIToken, "will-expire-soon", SecretSourceInline, nil)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

	// Verify it exists initially
	_, err = sm.RetrieveSecret("expiring-secret")
	if err != nil {
		t.Fatalf("Secret should exist initially: %v", err)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Try to retrieve expired secret
	_, err = sm.RetrieveSecret("expiring-secret")
	if err == nil {
		t.Error("Expected secret to be expired")
	}

	if !contains(err.Error(), "expired") {
		t.Errorf("Expected expiration error, got: %v", err)
	}
}

func TestAuditLogging(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := SecretConfig{
		StorageDir:         tempDir,
		EncryptAtRest:      false,
		MasterKeySource:    SecretSourceEnvironment,
		AllowInlineSecrets: true,
	}

	sm, err := NewSecretsManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}
	defer func() { _ = sm.Stop() }()

	// Perform various operations
	_ = sm.StoreSecret("audit-test", SecretTypeAPIToken, "test-value", SecretSourceInline, nil)
	_, _ = sm.RetrieveSecret("audit-test")
	_, _ = sm.RetrieveSecret("non-existent") // This should fail
	_ = sm.DeleteSecret("audit-test")

	// Check audit log
	auditLog := sm.GetAuditLog()

	if len(auditLog) < 4 {
		t.Errorf("Expected at least 4 audit entries, got %d", len(auditLog))
	}

	// Check for expected operations
	operations := make(map[string]bool)
	for _, entry := range auditLog {
		operations[entry.Operation] = true

		if entry.Timestamp.IsZero() {
			t.Error("Expected timestamp to be set for audit entry")
		}
	}

	expectedOps := []string{"store", "retrieve", "delete"}
	for _, op := range expectedOps {
		if !operations[op] {
			t.Errorf("Expected operation '%s' in audit log", op)
		}
	}

	// Check that failed operations are logged correctly
	failedRetrieve := false
	for _, entry := range auditLog {
		if entry.Operation == "retrieve" && !entry.Success {
			failedRetrieve = true
			if entry.Error == "" {
				t.Error("Expected error message for failed retrieve operation")
			}
		}
	}

	if !failedRetrieve {
		t.Error("Expected to find failed retrieve operation in audit log")
	}
}

func TestSecretsManagerHealth(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := SecretConfig{
		StorageDir:         tempDir,
		EncryptAtRest:      true,
		MasterKeySource:    SecretSourceEnvironment,
		MasterKeyEnv:       "TEST_MASTER_KEY",
		EnableRotation:     true,
		RotationInterval:   1 * time.Hour,
		AllowInlineSecrets: true,
	}

	_ = os.Setenv("TEST_MASTER_KEY", "dGVzdC1tYXN0ZXIta2V5LTEyMzQ1Njc4OTBhYmNkZWY=")
	defer func() { _ = os.Unsetenv("TEST_MASTER_KEY") }()

	sm, err := NewSecretsManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}
	defer func() { _ = sm.Stop() }()

	// Store some secrets, including an expired one
	_ = sm.StoreSecret("active-secret", SecretTypeAPIToken, "active-value", SecretSourceInline, nil)

	// Create an expired secret by manipulating the expiration time
	_ = sm.StoreSecret("expired-secret", SecretTypeAPIToken, "expired-value", SecretSourceInline, nil)
	sm.mutex.Lock()
	if secret, exists := sm.secrets["expired-secret"]; exists {
		pastTime := time.Now().Add(-2 * time.Hour)
		secret.ExpiresAt = &pastTime
	}
	sm.mutex.Unlock()

	// Get health status
	health := sm.Health()

	totalSecrets, ok := health["total_secrets"].(int)
	if !ok || totalSecrets != 2 {
		t.Errorf("Expected 2 total secrets, got %v", health["total_secrets"])
	}

	expiredSecrets, ok := health["expired_secrets"].(int)
	if !ok || expiredSecrets != 1 {
		t.Errorf("Expected 1 expired secret, got %v", health["expired_secrets"])
	}

	encryptionEnabled, ok := health["encryption_enabled"].(bool)
	if !ok || !encryptionEnabled {
		t.Errorf("Expected encryption to be enabled, got %v", health["encryption_enabled"])
	}

	rotationEnabled, ok := health["rotation_enabled"].(bool)
	if !ok || !rotationEnabled {
		t.Errorf("Expected rotation to be enabled, got %v", health["rotation_enabled"])
	}

	storageDir, ok := health["storage_directory"].(string)
	if !ok || storageDir != tempDir {
		t.Errorf("Expected storage directory '%s', got %v", tempDir, health["storage_directory"])
	}
}

func TestInlineSecretsDisabled(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := SecretConfig{
		StorageDir:         tempDir,
		EncryptAtRest:      false,
		MasterKeySource:    SecretSourceEnvironment,
		AllowInlineSecrets: false, // Disable inline secrets
	}

	sm, err := NewSecretsManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}
	defer func() { _ = sm.Stop() }()

	// Try to store inline secret - should fail
	err = sm.StoreSecret("inline-secret", SecretTypeAPIToken, "inline-value", SecretSourceInline, nil)
	if err == nil {
		t.Error("Expected inline secret storage to fail when disabled")
	}

	if !contains(err.Error(), "inline secrets are not allowed") {
		t.Errorf("Expected inline secrets error, got: %v", err)
	}

	// Store with different source - should succeed
	err = sm.StoreSecret("file-secret", SecretTypeAPIToken, "file-value", SecretSourceFile, nil)
	if err != nil {
		t.Fatalf("Expected file secret to be allowed: %v", err)
	}
}

func TestPersistenceAndReload(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := SecretConfig{
		StorageDir:         tempDir,
		EncryptAtRest:      true,
		MasterKeySource:    SecretSourceEnvironment,
		MasterKeyEnv:       "TEST_MASTER_KEY",
		RequireEncryption:  true,
		AllowInlineSecrets: true,
	}

	_ = os.Setenv("TEST_MASTER_KEY", "dGVzdC1tYXN0ZXIta2V5LTEyMzQ1Njc4OTBhYmNkZWY=")
	defer func() { _ = os.Unsetenv("TEST_MASTER_KEY") }()

	// Create first manager and store secrets
	sm1, err := NewSecretsManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create first secrets manager: %v", err)
	}

	err = sm1.StoreSecret("persistent-secret", SecretTypeAPIToken, "persistent-value", SecretSourceInline, map[string]string{
		"env": "test",
	})
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

	_ = sm1.Stop()

	// Create second manager with same config - should reload secrets
	sm2, err := NewSecretsManager(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create second secrets manager: %v", err)
	}
	defer func() { _ = sm2.Stop() }()

	// Check what secrets are available
	refs := sm2.ListSecrets()
	t.Logf("Reloaded secrets: %d found", len(refs))
	for _, ref := range refs {
		t.Logf("  - %s (type: %s, source: %s)", ref.Name, ref.Type, ref.Source)
	}

	// Verify secret was reloaded
	secret, err := sm2.RetrieveSecret("persistent-secret")
	if err != nil {
		// Check if file exists
		secretPath := filepath.Join(tempDir, "persistent-secret.secret")
		if _, statErr := os.Stat(secretPath); statErr != nil {
			t.Logf("Secret file does not exist: %v", statErr)
		} else {
			t.Logf("Secret file exists but failed to load")
		}
		t.Fatalf("Failed to retrieve reloaded secret: %v", err)
	}

	if secret.Value != "persistent-value" {
		t.Errorf("Expected reloaded value 'persistent-value', got '%s'", secret.Value)
	}

	if secret.Metadata["env"] != "test" {
		t.Errorf("Expected metadata to be preserved, got %v", secret.Metadata)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
