// +build !windows

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/jontk/s9s/internal/debug"
)

// KeyringSecureStore implements secure storage using the system keyring
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
	store := &KeyringSecureStore{
		serviceName: serviceName,
	}

	// Select appropriate backend based on platform
	switch runtime.GOOS {
	case "linux":
		store.backend = NewLinuxKeyringBackend()
	case "darwin":
		store.backend = NewMacOSKeyringBackend()
	default:
		// Fall back to file-based storage for unsupported platforms
		debug.Logger.Printf("Keyring not supported on %s, falling back to file storage", runtime.GOOS)
		return NewFileSecureStore(fmt.Sprintf("/tmp/s9s-%s.keyring", serviceName))
	}

	return store
}

// Initialize initializes the keyring secure store
func (k *KeyringSecureStore) Initialize() error {
	debug.Logger.Printf("Initialized keyring secure store for service: %s", k.serviceName)
	return nil
}

// Store saves data securely in the keyring
func (k *KeyringSecureStore) Store(key string, data []byte) error {
	// Encode data as JSON string for keyring storage
	value := string(data)
	
	if err := k.backend.Set(k.serviceName, key, value); err != nil {
		return fmt.Errorf("failed to store in keyring: %w", err)
	}

	debug.Logger.Printf("Stored data in keyring for key: %s", key)
	return nil
}

// Retrieve gets data securely from the keyring
func (k *KeyringSecureStore) Retrieve(key string) ([]byte, error) {
	value, err := k.backend.Get(k.serviceName, key)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve from keyring: %w", err)
	}

	debug.Logger.Printf("Retrieved data from keyring for key: %s", key)
	return []byte(value), nil
}

// Delete removes data securely from the keyring
func (k *KeyringSecureStore) Delete(key string) error {
	if err := k.backend.Delete(k.serviceName, key); err != nil {
		return fmt.Errorf("failed to delete from keyring: %w", err)
	}

	debug.Logger.Printf("Deleted data from keyring for key: %s", key)
	return nil
}

// List returns all keys in the keyring
func (k *KeyringSecureStore) List() ([]string, error) {
	keys, err := k.backend.List(k.serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to list keyring keys: %w", err)
	}

	return keys, nil
}

// Cleanup performs any necessary cleanup
func (k *KeyringSecureStore) Cleanup() error {
	debug.Logger.Printf("Keyring secure store cleanup completed")
	return nil
}

// Linux implementation using Secret Service API
type LinuxKeyringBackend struct {
	// Linux keyring implementation would use libsecret or similar
	// For now, we'll use a simplified implementation
	fallback SecureStore
}

// NewLinuxKeyringBackend creates a Linux keyring backend
func NewLinuxKeyringBackend() KeyringBackend {
	// In a real implementation, this would detect and use:
	// - GNOME Keyring
	// - KDE KWallet  
	// - Secret Service API
	// For now, fall back to encrypted file storage
	return &LinuxKeyringBackend{
		fallback: NewFileSecureStore("/tmp/s9s-linux-keyring.encrypted"),
	}
}

// Set stores a password in the Linux keyring
func (l *LinuxKeyringBackend) Set(service, user, password string) error {
	// In a real implementation, this would use libsecret:
	// secret_password_store_sync()
	
	debug.Logger.Printf("Linux keyring: storing password for %s@%s", user, service)
	
	// Use fallback for now
	key := fmt.Sprintf("%s:%s", service, user)
	return l.fallback.Store(key, []byte(password))
}

// Get retrieves a password from the Linux keyring
func (l *LinuxKeyringBackend) Get(service, user string) (string, error) {
	// In a real implementation, this would use libsecret:
	// secret_password_lookup_sync()
	
	debug.Logger.Printf("Linux keyring: retrieving password for %s@%s", user, service)
	
	// Use fallback for now
	key := fmt.Sprintf("%s:%s", service, user)
	data, err := l.fallback.Retrieve(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Delete removes a password from the Linux keyring
func (l *LinuxKeyringBackend) Delete(service, user string) error {
	// In a real implementation, this would use libsecret:
	// secret_password_clear_sync()
	
	debug.Logger.Printf("Linux keyring: deleting password for %s@%s", user, service)
	
	// Use fallback for now
	key := fmt.Sprintf("%s:%s", service, user)
	return l.fallback.Delete(key)
}

// List returns all users for a service in the Linux keyring
func (l *LinuxKeyringBackend) List(service string) ([]string, error) {
	// In a real implementation, this would search the keyring
	debug.Logger.Printf("Linux keyring: listing users for service %s", service)
	
	// Use fallback for now
	keys, err := l.fallback.List()
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

// macOS implementation using Keychain Services
type MacOSKeyringBackend struct {
	// macOS keyring implementation would use Keychain Services API
	fallback SecureStore
}

// NewMacOSKeyringBackend creates a macOS keyring backend
func NewMacOSKeyringBackend() KeyringBackend {
	// In a real implementation, this would use Keychain Services:
	// SecItemAdd, SecItemCopyMatching, SecItemDelete, etc.
	// For now, fall back to encrypted file storage
	return &MacOSKeyringBackend{
		fallback: NewFileSecureStore("/tmp/s9s-macos-keychain.encrypted"),
	}
}

// Set stores a password in the macOS Keychain
func (m *MacOSKeyringBackend) Set(service, user, password string) error {
	// In a real implementation, this would use Security.framework:
	// SecItemAdd with kSecClassGenericPassword
	
	debug.Logger.Printf("macOS keychain: storing password for %s@%s", user, service)
	
	// Use fallback for now
	key := fmt.Sprintf("%s:%s", service, user)
	return m.fallback.Store(key, []byte(password))
}

// Get retrieves a password from the macOS Keychain
func (m *MacOSKeyringBackend) Get(service, user string) (string, error) {
	// In a real implementation, this would use Security.framework:
	// SecItemCopyMatching with kSecClassGenericPassword
	
	debug.Logger.Printf("macOS keychain: retrieving password for %s@%s", user, service)
	
	// Use fallback for now
	key := fmt.Sprintf("%s:%s", service, user)
	data, err := m.fallback.Retrieve(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Delete removes a password from the macOS Keychain
func (m *MacOSKeyringBackend) Delete(service, user string) error {
	// In a real implementation, this would use Security.framework:
	// SecItemDelete with kSecClassGenericPassword
	
	debug.Logger.Printf("macOS keychain: deleting password for %s@%s", user, service)
	
	// Use fallback for now
	key := fmt.Sprintf("%s:%s", service, user)
	return m.fallback.Delete(key)
}

// List returns all users for a service in the macOS Keychain
func (m *MacOSKeyringBackend) List(service string) ([]string, error) {
	// In a real implementation, this would search the keychain
	debug.Logger.Printf("macOS keychain: listing users for service %s", service)
	
	// Use fallback for now
	keys, err := m.fallback.List()
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
		return fmt.Errorf("failed to store token in keyring: %w", err)
	}

	debug.Logger.Printf("Stored token in keyring for cluster: %s", clusterID)
	return nil
}

// Retrieve gets a token from the keyring
func (k *KeyringTokenStore) Retrieve(ctx context.Context, clusterID string) (*Token, error) {
	key := fmt.Sprintf("token:%s", clusterID)
	
	// Retrieve from keyring
	tokenBytes, err := k.secureStore.Retrieve(key)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve token from keyring: %w", err)
	}

	// Deserialize token
	var token Token
	if err := json.Unmarshal(tokenBytes, &token); err != nil {
		return nil, fmt.Errorf("failed to deserialize token: %w", err)
	}

	debug.Logger.Printf("Retrieved token from keyring for cluster: %s", clusterID)
	return &token, nil
}

// Delete removes a token from the keyring
func (k *KeyringTokenStore) Delete(ctx context.Context, clusterID string) error {
	key := fmt.Sprintf("token:%s", clusterID)
	
	if err := k.secureStore.Delete(key); err != nil {
		return fmt.Errorf("failed to delete token from keyring: %w", err)
	}

	debug.Logger.Printf("Deleted token from keyring for cluster: %s", clusterID)
	return nil
}

// List returns all stored cluster IDs from the keyring
func (k *KeyringTokenStore) List(ctx context.Context) ([]string, error) {
	keys, err := k.secureStore.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list keyring keys: %w", err)
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
		return fmt.Errorf("failed to list keyring keys: %w", err)
	}

	// Delete all token keys
	for _, key := range keys {
		if len(key) > 6 && key[:6] == "token:" {
			if err := k.secureStore.Delete(key); err != nil {
				debug.Logger.Printf("Failed to delete keyring key %s: %v", key, err)
			}
		}
	}

	debug.Logger.Printf("Cleared all tokens from keyring")
	return nil
}