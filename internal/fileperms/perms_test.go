package fileperms

import (
	"os"
	"testing"
)

func TestIsSecure(t *testing.T) {
	tests := []struct {
		name string
		mode os.FileMode
		want bool
	}{
		{"User only read/write", 0o600, true},
		{"User only rwx", 0o700, true},
		{"World readable", 0o644, false},
		{"Group readable", 0o640, false},
		{"World writable", 0o666, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSecure(tt.mode); got != tt.want {
				t.Errorf("IsSecure(%o) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}

func TestMakeSecure(t *testing.T) {
	tests := []struct {
		name string
		mode os.FileMode
		want os.FileMode
	}{
		{"Already secure", 0o600, 0o600},
		{"Remove group read", 0o640, 0o600},
		{"Remove world read", 0o644, 0o600},
		{"Remove all group/world", 0o777, 0o700},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MakeSecure(tt.mode); got != tt.want {
				t.Errorf("MakeSecure(%o) = %o, want %o", tt.mode, got, tt.want)
			}
		})
	}
}

func TestHasGroupAccess(t *testing.T) {
	tests := []struct {
		name string
		mode os.FileMode
		want bool
	}{
		{"No group access", 0o600, false},
		{"Group read", 0o640, true},
		{"Group write", 0o620, true},
		{"Group exec", 0o610, true},
		{"Group rwx", 0o670, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasGroupAccess(tt.mode); got != tt.want {
				t.Errorf("HasGroupAccess(%o) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}

func TestHasWorldAccess(t *testing.T) {
	tests := []struct {
		name string
		mode os.FileMode
		want bool
	}{
		{"No world access", 0o600, false},
		{"No world access (group only)", 0o640, false},
		{"World read", 0o604, true},
		{"World write", 0o602, true},
		{"World exec", 0o601, true},
		{"World readable", 0o644, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasWorldAccess(tt.mode); got != tt.want {
				t.Errorf("HasWorldAccess(%o) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}

func TestPermissionConstants(t *testing.T) {
	// Verify that constants have expected values
	tests := []struct {
		name string
		got  os.FileMode
		want os.FileMode
	}{
		{"DirDefault", DirDefault, 0o755},
		{"DirUserOnly", DirUserOnly, 0o700},
		{"FileDefault", FileDefault, 0o644},
		{"FileUserOnly", FileUserOnly, 0o600},
		{"FileExecutable", FileExecutable, 0o755},
		{"SecretFile", SecretFile, 0o600},
		{"SecretDir", SecretDir, 0o700},
		{"SSHPrivateKey", SSHPrivateKey, 0o600},
		{"SSHPublicKey", SSHPublicKey, 0o644},
		{"SSHDir", SSHDir, 0o700},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %o, want %o", tt.name, tt.got, tt.want)
			}
		})
	}
}
