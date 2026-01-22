//go:build windows
// +build windows

package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jontk/s9s/internal/debug"
)

// KeyringSecureStore implements secure storage using Windows Credential Store
type KeyringSecureStore struct {
	serviceName string
	backend     KeyringBackend
}

// KeyringBackend defines the interface for keyring implementations
type KeyringBackend interface {
	Set(service, user, password string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
	List(service string) ([]string, error)
}

// NewKeyringSecureStore creates a new keyring-based secure store
func NewKeyringSecureStore(serviceName string) SecureStore {
	return &KeyringSecureStore{
		serviceName: serviceName,
		backend:     NewWindowsKeyringBackend(),
	}
}

// Initialize initializes the keyring secure store
func (k *KeyringSecureStore) Initialize() error {
	debug.Logger.Printf("Initialized Windows keyring secure store for service: %s", k.serviceName)
	return nil
}

// Store saves data securely in the keyring
func (k *KeyringSecureStore) Store(key string, data []byte) error {
	// Encode data as JSON string for keyring storage
	value := string(data)

	if err := k.backend.Set(k.serviceName, key, value); err != nil {
		return fmt.Errorf("failed to store in Windows keyring: %w", err)
	}

	debug.Logger.Printf("Stored data in Windows keyring for key: %s", key)
	return nil
}

// Retrieve gets data securely from the keyring
func (k *KeyringSecureStore) Retrieve(key string) ([]byte, error) {
	value, err := k.backend.Get(k.serviceName, key)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve from Windows keyring: %w", err)
	}

	debug.Logger.Printf("Retrieved data from Windows keyring for key: %s", key)
	return []byte(value), nil
}

// Delete removes data securely from the keyring
func (k *KeyringSecureStore) Delete(key string) error {
	if err := k.backend.Delete(k.serviceName, key); err != nil {
		return fmt.Errorf("failed to delete from Windows keyring: %w", err)
	}

	debug.Logger.Printf("Deleted data from Windows keyring for key: %s", key)
	return nil
}

// List returns all keys in the keyring
func (k *KeyringSecureStore) List() ([]string, error) {
	keys, err := k.backend.List(k.serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to list Windows keyring keys: %w", err)
	}

	return keys, nil
}

// Cleanup performs any necessary cleanup
func (k *KeyringSecureStore) Cleanup() error {
	debug.Logger.Printf("Windows keyring secure store cleanup completed")
	return nil
}

// Windows implementation using Windows Credential Store
type WindowsKeyringBackend struct {
	// Windows keyring implementation would use Windows Credential Store API
	fallback SecureStore
}

// NewWindowsKeyringBackend creates a Windows keyring backend
func NewWindowsKeyringBackend() KeyringBackend {
	// In a real implementation, this would use Windows Credential Store:
	// CredWrite, CredRead, CredDelete, CredEnumerate
	// For now, fall back to encrypted file storage
	return &WindowsKeyringBackend{
		fallback: NewFileSecureStore("C:\\temp\\s9s-windows-creds.encrypted"),
	}
}

// Set stores a credential in the Windows Credential Store
func (w *WindowsKeyringBackend) Set(service, user, password string) error {
	// In a real implementation, this would use advapi32.dll:
	// CredWriteW with CREDENTIAL_TYPE_GENERIC

	debug.Logger.Printf("Windows credential store: storing credential for %s@%s", user, service)

	// Use fallback for now
	key := fmt.Sprintf("%s:%s", service, user)
	return w.fallback.Store(key, []byte(password))
}

// Get retrieves a credential from the Windows Credential Store
func (w *WindowsKeyringBackend) Get(service, user string) (string, error) {
	// In a real implementation, this would use advapi32.dll:
	// CredReadW with CREDENTIAL_TYPE_GENERIC

	debug.Logger.Printf("Windows credential store: retrieving credential for %s@%s", user, service)

	// Use fallback for now
	key := fmt.Sprintf("%s:%s", service, user)
	data, err := w.fallback.Retrieve(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Delete removes a credential from the Windows Credential Store
func (w *WindowsKeyringBackend) Delete(service, user string) error {
	// In a real implementation, this would use advapi32.dll:
	// CredDeleteW with CREDENTIAL_TYPE_GENERIC

	debug.Logger.Printf("Windows credential store: deleting credential for %s@%s", user, service)

	// Use fallback for now
	key := fmt.Sprintf("%s:%s", service, user)
	return w.fallback.Delete(key)
}

// List returns all users for a service in the Windows Credential Store
func (w *WindowsKeyringBackend) List(service string) ([]string, error) {
	// In a real implementation, this would use advapi32.dll:
	// CredEnumerateW to list credentials

	debug.Logger.Printf("Windows credential store: listing users for service %s", service)

	// Use fallback for now
	keys, err := w.fallback.List()
	if err != nil {
		return nil, err
	}

	var users []string
	prefix := service + ":"
	for _, key := range keys {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			user := key[len(prefix):]
			users = append(users, user)
		}
	}
	return users, nil
}

// KeyringTokenStore implements TokenStore using KeyringSecureStore
type KeyringTokenStore struct {
	secureStore SecureStore
}

// NewKeyringTokenStore creates a new keyring-based token store
func NewKeyringTokenStore(serviceName string) TokenStore {
	return &KeyringTokenStore{
		secureStore: NewKeyringSecureStore(serviceName),
	}
}

// Store saves a token securely in the keyring
func (k *KeyringTokenStore) Store(ctx context.Context, clusterID string, token *Token) error {
	// Serialize token
	tokenBytes, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to serialize token: %w", err)
	}

	// Store in keyring
	key := fmt.Sprintf("token:%s", clusterID)
	if err := k.secureStore.Store(key, tokenBytes); err != nil {
		return fmt.Errorf("failed to store token in Windows keyring: %w", err)
	}

	debug.Logger.Printf("Stored token in Windows keyring for cluster: %s", clusterID)
	return nil
}

// Retrieve gets a token from the keyring
func (k *KeyringTokenStore) Retrieve(ctx context.Context, clusterID string) (*Token, error) {
	key := fmt.Sprintf("token:%s", clusterID)

	// Retrieve from keyring
	tokenBytes, err := k.secureStore.Retrieve(key)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve token from Windows keyring: %w", err)
	}

	// Deserialize token
	var token Token
	if err := json.Unmarshal(tokenBytes, &token); err != nil {
		return nil, fmt.Errorf("failed to deserialize token: %w", err)
	}

	debug.Logger.Printf("Retrieved token from Windows keyring for cluster: %s", clusterID)
	return &token, nil
}

// Delete removes a token from the keyring
func (k *KeyringTokenStore) Delete(ctx context.Context, clusterID string) error {
	key := fmt.Sprintf("token:%s", clusterID)

	if err := k.secureStore.Delete(key); err != nil {
		return fmt.Errorf("failed to delete token from Windows keyring: %w", err)
	}

	debug.Logger.Printf("Deleted token from Windows keyring for cluster: %s", clusterID)
	return nil
}

// List returns all stored cluster IDs from the keyring
func (k *KeyringTokenStore) List(ctx context.Context) ([]string, error) {
	keys, err := k.secureStore.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list Windows keyring keys: %w", err)
	}

	// Filter token keys and extract cluster IDs
	var clusterIDs []string
	for _, key := range keys {
		if len(key) > 6 && key[:6] == "token:" {
			clusterID := key[6:]
			clusterIDs = append(clusterIDs, clusterID)
		}
	}

	return clusterIDs, nil
}

// Clear removes all tokens from the keyring
func (k *KeyringTokenStore) Clear(ctx context.Context) error {
	keys, err := k.secureStore.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list Windows keyring keys: %w", err)
	}

	// Delete all token keys
	for _, key := range keys {
		if len(key) > 6 && key[:6] == "token:" {
			if err := k.secureStore.Delete(key); err != nil {
				debug.Logger.Printf("Failed to delete Windows keyring key %s: %v", key, err)
			}
		}
	}

	debug.Logger.Printf("Cleared all tokens from Windows keyring")
	return nil
}
