package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsAlwaysProtected(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "empty path",
			path: "",
			want: false,
		},
		{
			name: "claude directory",
			path: filepath.Join(home, ".claude", "settings.json"),
			want: true,
		},
		{
			name: "ssh directory",
			path: filepath.Join(home, ".ssh", "id_rsa"),
			want: true,
		},
		{
			name: "ssh directory root",
			path: filepath.Join(home, ".ssh"),
			want: true,
		},
		{
			name: "aws directory",
			path: filepath.Join(home, ".aws", "credentials"),
			want: true,
		},
		{
			name: "gnupg directory",
			path: filepath.Join(home, ".gnupg", "pubring.gpg"),
			want: true,
		},
		{
			name: "gpg directory",
			path: filepath.Join(home, ".gpg", "key"),
			want: true,
		},
		{
			name: "gh config",
			path: filepath.Join(home, ".config", "gh", "hosts.yml"),
			want: true,
		},
		{
			name: "watchman global config",
			path: filepath.Join(home, ".config", "watchman", "config.yml"),
			want: true,
		},
		{
			name: "netrc file",
			path: filepath.Join(home, ".netrc"),
			want: true,
		},
		{
			name: "git-credentials file",
			path: filepath.Join(home, ".git-credentials"),
			want: true,
		},
		{
			name: "watchman binary",
			path: filepath.Join(home, "go", "bin", "watchman"),
			want: true,
		},
		{
			name: "watchman.yml in any directory",
			path: "/some/project/.watchman.yml",
			want: true,
		},
		{
			name: "watchman.yml relative",
			path: ".watchman.yml",
			want: true,
		},
		{
			name: "regular file in home",
			path: filepath.Join(home, "Documents", "file.txt"),
			want: false,
		},
		{
			name: "regular project file",
			path: "src/main.go",
			want: false,
		},
		{
			name: "similar but not protected",
			path: filepath.Join(home, ".sshkeys"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAlwaysProtected(tt.path)
			if got != tt.want {
				t.Errorf("IsAlwaysProtected(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestResolvePath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "absolute path",
			path: "/etc/passwd",
			want: "/etc/passwd",
		},
		{
			name: "absolute path with dots",
			path: "/etc/../etc/passwd",
			want: "/etc/passwd",
		},
		{
			name: "relative path",
			path: "src/main.go",
			want: filepath.Join(cwd, "src/main.go"),
		},
		{
			name: "relative path with dots",
			path: "./src/../src/main.go",
			want: filepath.Join(cwd, "src/main.go"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolvePath(tt.path)
			if got != tt.want {
				t.Errorf("resolvePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		pattern string
		want    bool
	}{
		{
			name:    "exact match",
			path:    "/etc/passwd",
			pattern: "/etc/passwd",
			want:    true,
		},
		{
			name:    "no match",
			path:    "/etc/passwd",
			pattern: "/etc/shadow",
			want:    false,
		},
		{
			name:    "directory prefix match",
			path:    "/etc/ssh/sshd_config",
			pattern: "/etc/ssh/",
			want:    true,
		},
		{
			name:    "directory exact match",
			path:    "/etc/ssh",
			pattern: "/etc/ssh/",
			want:    true,
		},
		{
			name:    "path prefix without trailing slash",
			path:    "/etc/ssh/sshd_config",
			pattern: "/etc/ssh",
			want:    true,
		},
		{
			name:    "no partial match",
			path:    "/etc/sshd",
			pattern: "/etc/ssh",
			want:    false,
		},
		{
			name:    "tilde expansion",
			path:    filepath.Join(home, ".ssh", "id_rsa"),
			pattern: "~/.ssh/",
			want:    true,
		},
		{
			name:    "tilde expansion exact",
			path:    filepath.Join(home, ".netrc"),
			pattern: "~/.netrc",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPath(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("matchPath(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}
