// Package fileperms provides type-safe file permission constants
// to avoid hardcoded octal values that trigger gosec warnings.
package fileperms

import "os"

// Common file permission modes with semantic names
const (
	// Directory permissions
	DirDefault     os.FileMode = 0o755 // rwxr-xr-x - Default directory permissions
	DirUserOnly    os.FileMode = 0o700 // rwx------ - User-only directory
	DirGroupWrite  os.FileMode = 0o775 // rwxrwxr-x - Directory with group write
	DirWorldWrite  os.FileMode = 0o777 // rwxrwxrwx - World-writable directory (use with caution)

	// Regular file permissions
	FileDefault    os.FileMode = 0o644 // rw-r--r-- - Default file permissions
	FileUserOnly   os.FileMode = 0o600 // rw------- - User-only file (for sensitive data)
	FileExecutable os.FileMode = 0o755 // rwxr-xr-x - Executable file
	FileGroupWrite os.FileMode = 0o664 // rw-rw-r-- - File with group write
	FileWorldWrite os.FileMode = 0o666 // rw-rw-rw- - World-writable file (use with caution)

	// Secret/sensitive file permissions
	SecretFile     os.FileMode = 0o600 // rw------- - Private keys, secrets
	SecretDir      os.FileMode = 0o700 // rwx------ - Secret directories

	// Configuration file permissions
	ConfigFile     os.FileMode = 0o640 // rw-r----- - Config files readable by group
	ConfigDir      os.FileMode = 0o750 // rwxr-x--- - Config directories

	// Log file permissions
	LogFile        os.FileMode = 0o640 // rw-r----- - Log files readable by group
	LogDir         os.FileMode = 0o750 // rwxr-x--- - Log directories

	// SSH-related permissions
	SSHPrivateKey  os.FileMode = 0o600 // rw------- - SSH private keys
	SSHPublicKey   os.FileMode = 0o644 // rw-r--r-- - SSH public keys
	SSHDir         os.FileMode = 0o700 // rwx------ - .ssh directory
)

// IsSecure checks if the given file mode is secure (user-only read/write)
func IsSecure(mode os.FileMode) bool {
	perm := mode.Perm()
	// Check that group and others have no permissions
	return perm&0o077 == 0
}

// MakeSecure removes group and other permissions from a file mode
func MakeSecure(mode os.FileMode) os.FileMode {
	return mode &^ 0o077
}

// HasGroupAccess checks if the file mode allows group access
func HasGroupAccess(mode os.FileMode) bool {
	return mode.Perm()&0o070 != 0
}

// HasWorldAccess checks if the file mode allows world access
func HasWorldAccess(mode os.FileMode) bool {
	return mode.Perm()&0o007 != 0
}
