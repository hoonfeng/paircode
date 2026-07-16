package provider

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadDotEnv loads KEY=value files into the process environment without
// overriding variables that are already set (first file to set a key wins).
// Order: project .env → user credentials → ~/.env fallback.
// Existing environment variables always win over all three.
func LoadDotEnv() {
	LoadDotEnvForRoot(".")
}

// LoadDotEnvForRoot loads a root's .env file (if present) before the home .env
// fallback. When root is "." it behaves like LoadDotEnv().
func LoadDotEnvForRoot(root string) {
	dotEnvPath := ".env"
	if root != "" && root != "." {
		dotEnvPath = filepath.Join(root, ".env")
	}
	loadDotEnvFile(dotEnvPath)
	if p := UserCredentialsPath(); p != "" {
		loadDotEnvFile(p)
	}
	if home, err := os.UserHomeDir(); err == nil {
		loadDotEnvFile(filepath.Join(home, ".env"))
	}
}

// UserCredentialsPath returns the path to the user-level credentials file
// (where `setup` writes API keys). Empty if the config dir can't be determined.
func UserCredentialsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pair", "credentials.env")
}

// loadDotEnvFile reads one .env file (if present) and sets any keys not already
// present in the environment. Lenient, zero-dependency parsing.
func loadDotEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}
