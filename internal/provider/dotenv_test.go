package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	if err := os.WriteFile(envFile, []byte(`
# comment
KEY1=value1
KEY2 = "quoted value"
export KEY3=exported
EMPTY=
`), 0644); err != nil {
		t.Fatal(err)
	}

	// Clear env vars that might conflict
	for _, key := range []string{"KEY1", "KEY2", "KEY3"} {
		os.Unsetenv(key)
	}

	loadDotEnvFile(envFile)

	tests := []struct {
		key, want string
	}{
		{"KEY1", "value1"},
		{"KEY2", "quoted value"},
		{"KEY3", "exported"},
	}
	for _, tt := range tests {
		if got := os.Getenv(tt.key); got != tt.want {
			t.Errorf("%s = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestLoadDotEnvFileDoesNotOverrideExisting(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	if err := os.WriteFile(envFile, []byte("EXISTING=from_file\n"), 0644); err != nil {
		t.Fatal(err)
	}
	os.Setenv("EXISTING", "from_env")
	defer os.Unsetenv("EXISTING")

	loadDotEnvFile(envFile)
	if got := os.Getenv("EXISTING"); got != "from_env" {
		t.Errorf("EXISTING = %q, want 'from_env' (env should win)", got)
	}
}

func TestUserCredentialsPath(t *testing.T) {
	path := UserCredentialsPath()
	if path == "" {
		t.Fatal("UserCredentialsPath() should not be empty")
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("path should be absolute: %s", path)
	}
	if !strings.HasSuffix(path, ".pair\\credentials.env") && !strings.HasSuffix(path, ".pair/credentials.env") {
		t.Fatalf("path should end with .pair/credentials.env: %s", path)
	}
}
