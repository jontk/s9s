package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/jontk/s9s/internal/debug"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/term"
)

// FileSecureStore implements secure storage using encrypted files
type FileSecureStore struct {
	storePath string
	masterKey []byte
}

// NewFileSecureStore creates a new file-based secure store
func NewFileSecureStore(storePath string) SecureStore {
	return &FileSecureStore{
		storePath: storePath,
	}
}

// Initialize initializes the secure store
func (f *FileSecureStore) Initialize() error {
	// Create store directory if it doesn't exist
	dir := filepath.Dir(f.storePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create store directory: %w", err)
	}

	// Initialize master key
	if err := f.initializeMasterKey(); err != nil {
		return fmt.Errorf("failed to initialize master key: %w", err)
	}

	debug.Logger.Printf("Initialized file secure store at: %s", f.storePath)
	return nil
}

// initializeMasterKey initializes or loads the master encryption key
func (f *FileSecureStore) initializeMasterKey() error {
	keyFile := f.storePath + ".key"

	// Check if key file exists
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		// Create new master key
		return f.createMasterKey(keyFile)
	}

	// Load existing master key
	return f.loadMasterKey(keyFile)
}

// createMasterKey creates a new master key with user password
func (f *FileSecureStore) createMasterKey(keyFile string) error {
	debug.Logger.Printf("Creating new master key")

	// Get password from user
	password, err := f.promptForPassword("Enter master password for secure storage: ")
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}

	// Confirm password
	confirmPassword, err := f.promptForPassword("Confirm master password: ")
	if err != nil {
		return fmt.Errorf("failed to confirm password: %w", err)
	}

	if password != confirmPassword {
		return fmt.Errorf("passwords do not match")
	}

	// Generate salt
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key from password
	key := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)

	// Store salt and derive master key
	keyData := struct {
		Salt []byte `json:"salt"`
	}{
		Salt: salt,
	}

	keyDataBytes, err := json.Marshal(keyData)
	if err != nil {
		return fmt.Errorf("failed to marshal key data: %w", err)
	}

	// Write key file with restricted permissions
	if err := os.WriteFile(keyFile, keyDataBytes, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	f.masterKey = key
	debug.Logger.Printf("Created new master key")
	return nil
}

// loadMasterKey loads the existing master key
func (f *FileSecureStore) loadMasterKey(keyFile string) error {
	debug.Logger.Printf("Loading existing master key")

	// Read key file
	keyDataBytes, err := os.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	var keyData struct {
		Salt []byte `json:"salt"`
	}

	if err := json.Unmarshal(keyDataBytes, &keyData); err != nil {
		return fmt.Errorf("failed to unmarshal key data: %w", err)
	}

	// Get password from user
	password, err := f.promptForPassword("Enter master password: ")
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}

	// Derive key from password
	key := pbkdf2.Key([]byte(password), keyData.Salt, 100000, 32, sha256.New)
	f.masterKey = key

	// Validate key by trying to decrypt a test value
	if err := f.validateMasterKey(); err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}

	debug.Logger.Printf("Loaded master key successfully")
	return nil
}

// validateMasterKey validates the master key by decrypting existing data
func (f *FileSecureStore) validateMasterKey() error {
	// Check if store file exists
	if _, err := os.Stat(f.storePath); os.IsNotExist(err) {
		// No store file, key is valid
		return nil
	}

	// Try to load and decrypt the store
	_, err := f.loadStore()
	return err
}

// promptForPassword prompts the user for a password securely
func (f *FileSecureStore) promptForPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	
	// Use terminal package for secure password input
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Print newline after password input
	
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	return string(password), nil
}

// Store saves data securely
func (f *FileSecureStore) Store(key string, data []byte) error {
	// Load existing store
	store, err := f.loadStore()
	if err != nil {
		// Create new store if loading fails
		store = make(map[string]string)
	}

	// Encrypt data
	encryptedData, err := f.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Store encrypted data
	store[key] = encryptedData

	// Save store
	if err := f.saveStore(store); err != nil {
		return fmt.Errorf("failed to save store: %w", err)
	}

	debug.Logger.Printf("Stored encrypted data for key: %s", key)
	return nil
}

// Retrieve gets data securely
func (f *FileSecureStore) Retrieve(key string) ([]byte, error) {
	// Load store
	store, err := f.loadStore()
	if err != nil {
		return nil, fmt.Errorf("failed to load store: %w", err)
	}

	// Get encrypted data
	encryptedData, exists := store[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// Decrypt data
	data, err := f.decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	debug.Logger.Printf("Retrieved encrypted data for key: %s", key)
	return data, nil
}

// Delete removes data securely
func (f *FileSecureStore) Delete(key string) error {
	// Load store
	store, err := f.loadStore()
	if err != nil {
		return fmt.Errorf("failed to load store: %w", err)
	}

	// Delete key
	delete(store, key)

	// Save store
	if err := f.saveStore(store); err != nil {
		return fmt.Errorf("failed to save store: %w", err)
	}

	debug.Logger.Printf("Deleted data for key: %s", key)
	return nil
}

// List returns all keys in the store
func (f *FileSecureStore) List() ([]string, error) {
	// Load store
	store, err := f.loadStore()
	if err != nil {
		return nil, fmt.Errorf("failed to load store: %w", err)
	}

	// Extract keys
	keys := make([]string, 0, len(store))
	for key := range store {
		keys = append(keys, key)
	}

	return keys, nil
}

// loadStore loads the encrypted store from file
func (f *FileSecureStore) loadStore() (map[string]string, error) {
	// Check if file exists
	if _, err := os.Stat(f.storePath); os.IsNotExist(err) {
		return make(map[string]string), nil
	}

	// Read file
	encryptedBytes, err := os.ReadFile(f.storePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read store file: %w", err)
	}

	// Decrypt store
	decryptedBytes, err := f.decrypt(string(encryptedBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt store: %w", err)
	}

	// Unmarshal store
	var store map[string]string
	if err := json.Unmarshal(decryptedBytes, &store); err != nil {
		return nil, fmt.Errorf("failed to unmarshal store: %w", err)
	}

	return store, nil
}

// saveStore saves the store to file with encryption
func (f *FileSecureStore) saveStore(store map[string]string) error {
	// Marshal store
	storeBytes, err := json.Marshal(store)
	if err != nil {
		return fmt.Errorf("failed to marshal store: %w", err)
	}

	// Encrypt store
	encryptedStore, err := f.encrypt(storeBytes)
	if err != nil {
		return fmt.Errorf("failed to encrypt store: %w", err)
	}

	// Write to file with restricted permissions
	if err := os.WriteFile(f.storePath, []byte(encryptedStore), 0600); err != nil {
		return fmt.Errorf("failed to write store file: %w", err)
	}

	return nil
}

// encrypt encrypts data using AES-256-GCM
func (f *FileSecureStore) encrypt(data []byte) (string, error) {
	// Create cipher
	block, err := aes.NewCipher(f.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	// Encode to base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts data using AES-256-GCM
func (f *FileSecureStore) decrypt(encryptedData string) ([]byte, error) {
	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create cipher
	block, err := aes.NewCipher(f.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// Cleanup performs any necessary cleanup
func (f *FileSecureStore) Cleanup() error {
	// Clear master key from memory
	if f.masterKey != nil {
		for i := range f.masterKey {
			f.masterKey[i] = 0
		}
		f.masterKey = nil
	}

	debug.Logger.Printf("File secure store cleanup completed")
	return nil
}

// FileTokenStore implements TokenStore using FileSecureStore
type FileTokenStore struct {
	secureStore SecureStore
}

// NewFileTokenStore creates a new file-based token store
func NewFileTokenStore(storePath string) TokenStore {
	return &FileTokenStore{
		secureStore: NewFileSecureStore(storePath),
	}
}

// Store saves a token securely
func (f *FileTokenStore) Store(ctx context.Context, clusterID string, token *Token) error {
	// Serialize token
	tokenBytes, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to serialize token: %w", err)
	}

	// Store encrypted token
	key := fmt.Sprintf("token:%s", clusterID)
	if err := f.secureStore.Store(key, tokenBytes); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	debug.Logger.Printf("Stored token for cluster: %s", clusterID)
	return nil
}

// Retrieve gets a token from storage
func (f *FileTokenStore) Retrieve(ctx context.Context, clusterID string) (*Token, error) {
	key := fmt.Sprintf("token:%s", clusterID)
	
	// Retrieve encrypted token
	tokenBytes, err := f.secureStore.Retrieve(key)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve token: %w", err)
	}

	// Deserialize token
	var token Token
	if err := json.Unmarshal(tokenBytes, &token); err != nil {
		return nil, fmt.Errorf("failed to deserialize token: %w", err)
	}

	debug.Logger.Printf("Retrieved token for cluster: %s", clusterID)
	return &token, nil
}

// Delete removes a token from storage
func (f *FileTokenStore) Delete(ctx context.Context, clusterID string) error {
	key := fmt.Sprintf("token:%s", clusterID)
	
	if err := f.secureStore.Delete(key); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	debug.Logger.Printf("Deleted token for cluster: %s", clusterID)
	return nil
}

// List returns all stored cluster IDs
func (f *FileTokenStore) List(ctx context.Context) ([]string, error) {
	keys, err := f.secureStore.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	// Filter token keys and extract cluster IDs
	var clusterIDs []string
	for _, key := range keys {
		if strings.HasPrefix(key, "token:") {
			clusterID := strings.TrimPrefix(key, "token:")
			clusterIDs = append(clusterIDs, clusterID)
		}
	}

	return clusterIDs, nil
}

// Clear removes all tokens from storage
func (f *FileTokenStore) Clear(ctx context.Context) error {
	keys, err := f.secureStore.List()
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	// Delete all token keys
	for _, key := range keys {
		if strings.HasPrefix(key, "token:") {
			if err := f.secureStore.Delete(key); err != nil {
				debug.Logger.Printf("Failed to delete key %s: %v", key, err)
			}
		}
	}

	debug.Logger.Printf("Cleared all tokens from storage")
	return nil
}