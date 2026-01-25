// Package security provides security mechanisms including secrets management,
// authentication, authorization, and audit logging for the observability plugin.
package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// SecretType represents different types of secrets
type SecretType string

const (
	// SecretTypeAPIToken is the secret type for API tokens.
	SecretTypeAPIToken SecretType = "api_token"
	// SecretTypeBasicAuth is the secret type for basic authentication.
	SecretTypeBasicAuth SecretType = "basic_auth"
	// SecretTypeBearerToken is the secret type for bearer tokens.
	SecretTypeBearerToken SecretType = "bearer_token"
	// SecretTypeTLSCert is the secret type for TLS certificates.
	SecretTypeTLSCert SecretType = "tls_cert"
	// SecretTypeTLSKey is the secret type for TLS private keys.
	SecretTypeTLSKey SecretType = "tls_key"
	// SecretTypeDatabase is the secret type for database credentials.
	SecretTypeDatabase SecretType = "database"
	// SecretTypeEncryptionKey is the secret type for encryption keys.
	SecretTypeEncryptionKey SecretType = "encryption_key"
)

// SecretSource represents where secrets can be loaded from
type SecretSource string

const (
	// SecretSourceEnvironment is the secret source for environment variables.
	SecretSourceEnvironment SecretSource = "environment"
	// SecretSourceFile is the secret source for files.
	SecretSourceFile SecretSource = "file"
	// SecretSourceVault is the secret source for HashiCorp Vault.
	SecretSourceVault SecretSource = "vault"
	// SecretSourceK8sSecret is the secret source for Kubernetes secrets.
	SecretSourceK8sSecret SecretSource = "kubernetes"
	// SecretSourceInline is the secret source for inline values.
	SecretSourceInline SecretSource = "inline"
)

// Secret represents a secret with metadata
type Secret struct {
	Name      string            `json:"name"`
	Type      SecretType        `json:"type"`
	Value     string            `json:"-"` // Never serialize the actual value
	Source    SecretSource      `json:"source"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Encrypted bool              `json:"encrypted"`
	Rotatable bool              `json:"rotatable"`
}

// SecretConfig represents configuration for secret management
type SecretConfig struct {
	// Storage configuration
	StorageDir    string `json:"storage_dir" yaml:"storageDir"`
	EncryptAtRest bool   `json:"encrypt_at_rest" yaml:"encryptAtRest"`

	// Master key configuration
	MasterKeySource SecretSource `json:"master_key_source" yaml:"masterKeySource"`
	MasterKeyPath   string       `json:"master_key_path,omitempty" yaml:"masterKeyPath,omitempty"`
	MasterKeyEnv    string       `json:"master_key_env,omitempty" yaml:"masterKeyEnv,omitempty"`

	// Rotation configuration
	EnableRotation   bool          `json:"enable_rotation" yaml:"enableRotation"`
	RotationInterval time.Duration `json:"rotation_interval" yaml:"rotationInterval"`

	// Security settings
	RequireEncryption  bool `json:"require_encryption" yaml:"requireEncryption"`
	AllowInlineSecrets bool `json:"allow_inline_secrets" yaml:"allowInlineSecrets"`
}

// SecretsManager manages secrets with encryption and rotation capabilities
type SecretsManager struct {
	config    SecretConfig
	secrets   map[string]*Secret
	masterKey []byte
	mutex     sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc

	// Audit and monitoring
	accessLog []SecretAccess
	auditMu   sync.Mutex
}

// SecretAccess represents an audit log entry for secret access
type SecretAccess struct {
	SecretName string    `json:"secret_name"`
	Operation  string    `json:"operation"`
	Success    bool      `json:"success"`
	Timestamp  time.Time `json:"timestamp"`
	Source     string    `json:"source,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// SecretReference represents a reference to a secret without the actual value
type SecretReference struct {
	Name   string       `json:"name"`
	Type   SecretType   `json:"type"`
	Source SecretSource `json:"source"`
}

// NewSecretsManager creates a new secrets manager
func NewSecretsManager(ctx context.Context, config SecretConfig) (*SecretsManager, error) {
	if config.StorageDir == "" {
		config.StorageDir = "./data/secrets"
	}

	if config.MasterKeyEnv == "" {
		config.MasterKeyEnv = "OBSERVABILITY_MASTER_KEY"
	}

	if config.RotationInterval == 0 {
		config.RotationInterval = 24 * time.Hour // Default rotation every 24 hours
	}

	managerCtx, cancel := context.WithCancel(ctx)

	sm := &SecretsManager{
		config:    config,
		secrets:   make(map[string]*Secret),
		accessLog: make([]SecretAccess, 0),
		ctx:       managerCtx,
		cancel:    cancel,
	}

	// Initialize master key
	if err := sm.initializeMasterKey(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize master key: %w", err)
	}

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(config.StorageDir, 0700); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Load existing secrets
	if err := sm.loadSecrets(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load existing secrets: %w", err)
	}

	// Start rotation routine if enabled
	if config.EnableRotation {
		go sm.rotationRoutine()
	}

	return sm, nil
}

// initializeMasterKey initializes the master encryption key
func (sm *SecretsManager) initializeMasterKey() error {
	switch sm.config.MasterKeySource {
	case SecretSourceEnvironment:
		if err := sm.initializeMasterKeyFromEnvironment(); err != nil {
			return err
		}
	case SecretSourceFile:
		if err := sm.initializeMasterKeyFromFile(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported master key source: %s", sm.config.MasterKeySource)
	}

	return sm.validateMasterKeyLength()
}

func (sm *SecretsManager) initializeMasterKeyFromEnvironment() error {
	envKey := os.Getenv(sm.config.MasterKeyEnv)
	if envKey == "" {
		masterKey := make([]byte, 32)
		if _, err := rand.Read(masterKey); err != nil {
			return fmt.Errorf("failed to generate master key: %w", err)
		}
		sm.masterKey = masterKey
		sm.saveKeyDerivationHint()
		return nil
	}

	keyBytes, err := base64.StdEncoding.DecodeString(envKey)
	if err != nil {
		return fmt.Errorf("invalid master key format in environment: %w", err)
	}
	sm.masterKey = keyBytes
	return nil
}

func (sm *SecretsManager) initializeMasterKeyFromFile() error {
	if sm.config.MasterKeyPath == "" {
		return errors.New("master key path is required when using file source")
	}

	keyData, err := os.ReadFile(sm.config.MasterKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read master key file: %w", err)
	}

	if decodedKey, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(keyData))); err == nil {
		sm.masterKey = decodedKey
	} else {
		sm.masterKey = keyData
	}
	return nil
}

func (sm *SecretsManager) validateMasterKeyLength() error {
	if len(sm.masterKey) < 16 {
		return errors.New("master key must be at least 16 bytes long")
	}
	return nil
}

// saveKeyDerivationHint saves a hint about key derivation (not the actual key)
func (sm *SecretsManager) saveKeyDerivationHint() {
	hintPath := filepath.Join(sm.config.StorageDir, ".key_hint")
	hint := map[string]interface{}{
		"source":     sm.config.MasterKeySource,
		"env_var":    sm.config.MasterKeyEnv,
		"created_at": time.Now().Format(time.RFC3339),
		"key_hash":   hex.EncodeToString(sha256.New().Sum(sm.masterKey)[:8]), // First 8 bytes of hash for verification
	}

	// This is just a hint file, not containing sensitive data
	_ = sm.saveJSON(hintPath, hint)
}

// StoreSecret stores a secret with encryption
func (sm *SecretsManager) StoreSecret(name string, secretType SecretType, value string, source SecretSource, metadata map[string]string) error {
	if name == "" {
		return errors.New("secret name cannot be empty")
	}

	if value == "" {
		return errors.New("secret value cannot be empty")
	}

	// Validate secret based on type
	if err := sm.validateSecret(secretType, value); err != nil {
		return fmt.Errorf("secret validation failed: %w", err)
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Check if inline secrets are allowed
	if source == SecretSourceInline && !sm.config.AllowInlineSecrets {
		return errors.New("inline secrets are not allowed by configuration")
	}

	secret := &Secret{
		Name:      name,
		Type:      secretType,
		Value:     value,
		Source:    source,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  metadata,
		Encrypted: sm.config.EncryptAtRest || sm.config.RequireEncryption,
		Rotatable: sm.isRotatable(secretType),
	}

	// Set expiration for rotatable secrets
	if secret.Rotatable && sm.config.EnableRotation {
		expiresAt := time.Now().Add(sm.config.RotationInterval)
		secret.ExpiresAt = &expiresAt
	}

	sm.secrets[name] = secret

	// Persist to storage
	if err := sm.saveSecret(secret); err != nil {
		delete(sm.secrets, name)
		return fmt.Errorf("failed to save secret: %w", err)
	}

	// Audit log
	sm.logAccess(name, "store", true, "")

	return nil
}

// RetrieveSecret retrieves a secret by name
func (sm *SecretsManager) RetrieveSecret(name string) (*Secret, error) {
	if name == "" {
		return nil, errors.New("secret name cannot be empty")
	}

	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	secret, exists := sm.secrets[name]
	if !exists {
		sm.logAccess(name, "retrieve", false, "secret not found")
		return nil, fmt.Errorf("secret '%s' not found", name)
	}

	// Check expiration
	if secret.ExpiresAt != nil && time.Now().After(*secret.ExpiresAt) {
		sm.logAccess(name, "retrieve", false, "secret expired")
		return nil, fmt.Errorf("secret '%s' has expired", name)
	}

	// Return a copy to prevent modification
	secretCopy := *secret
	sm.logAccess(name, "retrieve", true, "")

	return &secretCopy, nil
}

// GetSecretValue retrieves just the secret value
func (sm *SecretsManager) GetSecretValue(name string) (string, error) {
	secret, err := sm.RetrieveSecret(name)
	if err != nil {
		return "", err
	}
	return secret.Value, nil
}

// DeleteSecret removes a secret
func (sm *SecretsManager) DeleteSecret(name string) error {
	if name == "" {
		return errors.New("secret name cannot be empty")
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	secret, exists := sm.secrets[name]
	if !exists {
		sm.logAccess(name, "delete", false, "secret not found")
		return fmt.Errorf("secret '%s' not found", name)
	}

	// Remove from memory
	delete(sm.secrets, name)

	// Remove from storage
	secretPath := filepath.Join(sm.config.StorageDir, name+".secret")
	if err := os.Remove(secretPath); err != nil && !os.IsNotExist(err) {
		// Re-add to memory if storage removal failed
		sm.secrets[name] = secret
		return fmt.Errorf("failed to remove secret file: %w", err)
	}

	sm.logAccess(name, "delete", true, "")
	return nil
}

// ListSecrets returns references to all secrets (without values)
func (sm *SecretsManager) ListSecrets() []SecretReference {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	refs := make([]SecretReference, 0, len(sm.secrets))
	for _, secret := range sm.secrets {
		refs = append(refs, SecretReference{
			Name:   secret.Name,
			Type:   secret.Type,
			Source: secret.Source,
		})
	}

	return refs
}

// RotateSecret manually rotates a secret
func (sm *SecretsManager) RotateSecret(name, newValue string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	secret, exists := sm.secrets[name]
	if !exists {
		return fmt.Errorf("secret '%s' not found", name)
	}

	if !secret.Rotatable {
		return fmt.Errorf("secret '%s' is not rotatable", name)
	}

	// Update secret
	secret.Value = newValue
	secret.UpdatedAt = time.Now()

	if sm.config.EnableRotation {
		expiresAt := time.Now().Add(sm.config.RotationInterval)
		secret.ExpiresAt = &expiresAt
	}

	// Persist to storage
	if err := sm.saveSecret(secret); err != nil {
		return fmt.Errorf("failed to save rotated secret: %w", err)
	}

	sm.logAccess(name, "rotate", true, "")
	return nil
}

// validateSecret validates a secret value based on its type
func (sm *SecretsManager) validateSecret(secretType SecretType, value string) error {
	switch secretType {
	case SecretTypeAPIToken, SecretTypeBearerToken:
		if len(value) < 8 {
			return errors.New("API tokens must be at least 8 characters long")
		}

	case SecretTypeBasicAuth:
		parts := strings.SplitN(value, ":", 2)
		if len(parts) != 2 {
			return errors.New("basic auth must be in format 'username:password'")
		}
		if len(parts[0]) == 0 || len(parts[1]) == 0 {
			return errors.New("username and password cannot be empty")
		}

	case SecretTypeEncryptionKey:
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return errors.New("encryption key must be base64 encoded")
		}
		if len(decoded) < 16 {
			return errors.New("encryption key must be at least 128 bits (16 bytes)")
		}
	}

	return nil
}

// isRotatable determines if a secret type supports rotation
func (sm *SecretsManager) isRotatable(secretType SecretType) bool {
	switch secretType {
	case SecretTypeAPIToken, SecretTypeBearerToken, SecretTypeEncryptionKey:
		return true
	case SecretTypeTLSCert, SecretTypeTLSKey:
		return true // Certificate rotation
	default:
		return false
	}
}

// Encryption/Decryption methods

// encryptValue encrypts a secret value using AES-GCM
func (sm *SecretsManager) encryptValue(plaintext string) (string, error) {
	if !sm.config.EncryptAtRest && !sm.config.RequireEncryption {
		return plaintext, nil
	}

	// Derive key using PBKDF2
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	key := pbkdf2.Key(sm.masterKey, salt, 4096, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	// Prepend salt to ciphertext
	encrypted := make([]byte, 0, len(salt)+len(ciphertext))
	encrypted = append(encrypted, salt...)
	encrypted = append(encrypted, ciphertext...)

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// decryptValue decrypts a secret value using AES-GCM
func (sm *SecretsManager) decryptValue(encryptedValue string) (string, error) {
	if !sm.config.EncryptAtRest && !sm.config.RequireEncryption {
		return encryptedValue, nil
	}

	encrypted, err := base64.StdEncoding.DecodeString(encryptedValue)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted value: %w", err)
	}

	if len(encrypted) < 16 {
		return "", errors.New("encrypted value too short")
	}

	// Extract salt and ciphertext
	salt := encrypted[:16]
	ciphertext := encrypted[16:]

	// Derive key using same parameters
	key := pbkdf2.Key(sm.masterKey, salt, 4096, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(ciphertext) < aesGCM.NonceSize() {
		return "", errors.New("ciphertext too short")
	}

	nonce := ciphertext[:aesGCM.NonceSize()]
	ciphertext = ciphertext[aesGCM.NonceSize():]

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// Persistence methods

// StorageSecret represents a secret for storage (with value included)
type StorageSecret struct {
	Name      string            `json:"name"`
	Type      SecretType        `json:"type"`
	Value     string            `json:"value"` // Include value for storage
	Source    SecretSource      `json:"source"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Encrypted bool              `json:"encrypted"`
	Rotatable bool              `json:"rotatable"`
}

// saveSecret saves a secret to storage
func (sm *SecretsManager) saveSecret(secret *Secret) error {
	secretPath := filepath.Join(sm.config.StorageDir, secret.Name+".secret")

	// Prepare secret for storage
	storageSecret := StorageSecret{
		Name:      secret.Name,
		Type:      secret.Type,
		Source:    secret.Source,
		CreatedAt: secret.CreatedAt,
		UpdatedAt: secret.UpdatedAt,
		ExpiresAt: secret.ExpiresAt,
		Metadata:  secret.Metadata,
		Encrypted: secret.Encrypted,
		Rotatable: secret.Rotatable,
		Value:     secret.Value, // Will be encrypted below if needed
	}

	// Encrypt value if required
	if secret.Encrypted {
		encryptedValue, err := sm.encryptValue(secret.Value)
		if err != nil {
			return fmt.Errorf("failed to encrypt secret: %w", err)
		}
		storageSecret.Value = encryptedValue
	}

	return sm.saveJSON(secretPath, storageSecret)
}

// loadSecrets loads all secrets from storage
func (sm *SecretsManager) loadSecrets() error {
	if _, err := os.Stat(sm.config.StorageDir); os.IsNotExist(err) {
		return nil // No secrets directory yet
	}

	entries, err := os.ReadDir(sm.config.StorageDir)
	if err != nil {
		return fmt.Errorf("failed to read secrets directory: %w", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".secret") {
			continue
		}

		secretPath := filepath.Join(sm.config.StorageDir, entry.Name())
		var storageSecret StorageSecret

		if err := sm.loadJSON(secretPath, &storageSecret); err != nil {
			// Log error but continue loading other secrets
			continue
		}

		// Convert to runtime Secret struct
		secret := Secret(storageSecret)

		// Decrypt value if encrypted
		if secret.Encrypted {
			decryptedValue, err := sm.decryptValue(secret.Value)
			if err != nil {
				// Log error but continue - might be key rotation issue
				continue
			}
			secret.Value = decryptedValue
		}

		sm.secrets[secret.Name] = &secret
	}

	return nil
}

// Utility methods for JSON storage
func (sm *SecretsManager) saveJSON(path string, data interface{}) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (sm *SecretsManager) loadJSON(path string, data interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	decoder := json.NewDecoder(file)
	return decoder.Decode(data)
}

// logAccess logs secret access for auditing
func (sm *SecretsManager) logAccess(secretName, operation string, success bool, errorMsg string) {
	sm.auditMu.Lock()
	defer sm.auditMu.Unlock()

	access := SecretAccess{
		SecretName: secretName,
		Operation:  operation,
		Success:    success,
		Timestamp:  time.Now(),
		Error:      errorMsg,
	}

	sm.accessLog = append(sm.accessLog, access)

	// Keep only last 1000 entries
	if len(sm.accessLog) > 1000 {
		sm.accessLog = sm.accessLog[len(sm.accessLog)-1000:]
	}
}

// GetAuditLog returns the secret access audit log
func (sm *SecretsManager) GetAuditLog() []SecretAccess {
	sm.auditMu.Lock()
	defer sm.auditMu.Unlock()

	// Return a copy
	logCopy := make([]SecretAccess, len(sm.accessLog))
	copy(logCopy, sm.accessLog)
	return logCopy
}

// rotationRoutine runs periodic secret rotation
func (sm *SecretsManager) rotationRoutine() {
	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.checkAndRotateSecrets()
		}
	}
}

// checkAndRotateSecrets checks for expired secrets and initiates rotation
func (sm *SecretsManager) checkAndRotateSecrets() {
	sm.mutex.RLock()
	expiredSecrets := make([]*Secret, 0)

	for _, secret := range sm.secrets {
		if secret.ExpiresAt != nil && time.Now().After(*secret.ExpiresAt) && secret.Rotatable {
			expiredSecrets = append(expiredSecrets, secret)
		}
	}
	sm.mutex.RUnlock()

	// Rotate expired secrets (this would typically involve external systems)
	for _, secret := range expiredSecrets {
		sm.logAccess(secret.Name, "auto_rotation_needed", true, "secret expired and needs rotation")
		// TODO: Implement actual rotation logic based on secret type
	}
}

// Stop gracefully stops the secrets manager
func (sm *SecretsManager) Stop() error {
	if sm == nil {
		return nil
	}
	if sm.cancel != nil {
		sm.cancel()
	}
	return nil
}

// Health returns the health status of the secrets manager
func (sm *SecretsManager) Health() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	totalSecrets := len(sm.secrets)
	expiredSecrets := 0

	for _, secret := range sm.secrets {
		if secret.ExpiresAt != nil && time.Now().After(*secret.ExpiresAt) {
			expiredSecrets++
		}
	}

	return map[string]interface{}{
		"total_secrets":      totalSecrets,
		"expired_secrets":    expiredSecrets,
		"encryption_enabled": sm.config.EncryptAtRest,
		"rotation_enabled":   sm.config.EnableRotation,
		"storage_directory":  sm.config.StorageDir,
	}
}
