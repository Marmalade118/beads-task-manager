package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var v *viper.Viper

// Initialize sets up the viper configuration singleton
// Should be called once at application startup
func Initialize() error {
	v = viper.New()

	// Set config type to yaml (we only load config.yaml, not config.json)
	v.SetConfigType("yaml")

	// Explicitly locate config.yaml and use SetConfigFile to avoid picking up config.json
	// Precedence: project .beads/config.yaml > ~/.config/bd/config.yaml > ~/.beads/config.yaml
	configFileSet := false

	// 1. Walk up from CWD to find project .beads/config.yaml
	//    This allows commands to work from subdirectories
	cwd, err := os.Getwd()
	if err == nil && !configFileSet {
		// Walk up parent directories to find .beads/config.yaml
		for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			beadsDir := filepath.Join(dir, ".beads")
			configPath := filepath.Join(beadsDir, "config.yaml")
			if _, err := os.Stat(configPath); err == nil {
				// Found .beads/config.yaml - set it explicitly
				v.SetConfigFile(configPath)
				configFileSet = true
				break
			}
		}
	}

	// 2. User config directory (~/.config/bd/config.yaml)
	if !configFileSet {
		if configDir, err := os.UserConfigDir(); err == nil {
			configPath := filepath.Join(configDir, "bd", "config.yaml")
			if _, err := os.Stat(configPath); err == nil {
				v.SetConfigFile(configPath)
				configFileSet = true
			}
		}
	}

	// 3. Home directory (~/.beads/config.yaml)
	if !configFileSet {
		if homeDir, err := os.UserHomeDir(); err == nil {
			configPath := filepath.Join(homeDir, ".beads", "config.yaml")
			if _, err := os.Stat(configPath); err == nil {
				v.SetConfigFile(configPath)
				configFileSet = true
			}
		}
	}

	// Automatic environment variable binding
	// Environment variables take precedence over config file
	// E.g., BD_JSON, BD_NO_DAEMON, BD_ACTOR, BD_DB
	v.SetEnvPrefix("BD")
	
	// Replace hyphens and dots with underscores for env var mapping
	// This allows BD_NO_DAEMON to map to "no-daemon" config key
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// Set defaults for all flags
	v.SetDefault("json", false)
	v.SetDefault("no-daemon", false)
	v.SetDefault("no-auto-flush", false)
	v.SetDefault("no-auto-import", false)
	v.SetDefault("no-db", false)
	v.SetDefault("db", "")
	v.SetDefault("actor", "")
	v.SetDefault("issue-prefix", "")
	
	// Additional environment variables (not prefixed with BD_)
	// These are bound explicitly for backward compatibility
	_ = v.BindEnv("flush-debounce", "BEADS_FLUSH_DEBOUNCE")
	_ = v.BindEnv("auto-start-daemon", "BEADS_AUTO_START_DAEMON")
	
	// Set defaults for additional settings
	v.SetDefault("flush-debounce", "30s")
	v.SetDefault("auto-start-daemon", true)

	// Read config file if it was found
	if configFileSet {
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("error reading config file: %w", err)
		}
		if os.Getenv("BD_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "Debug: loaded config from %s\n", v.ConfigFileUsed())
		}
	} else {
		// No config.yaml found - use defaults and environment variables
		if os.Getenv("BD_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "Debug: no config.yaml found; using defaults and environment variables\n")
		}
	}

	return nil
}

// GetString retrieves a string configuration value
func GetString(key string) string {
	if v == nil {
		return ""
	}
	return v.GetString(key)
}

// GetBool retrieves a boolean configuration value
func GetBool(key string) bool {
	if v == nil {
		return false
	}
	return v.GetBool(key)
}

// GetInt retrieves an integer configuration value
func GetInt(key string) int {
	if v == nil {
		return 0
	}
	return v.GetInt(key)
}

// GetDuration retrieves a duration configuration value
func GetDuration(key string) time.Duration {
	if v == nil {
		return 0
	}
	return v.GetDuration(key)
}

// Set sets a configuration value
func Set(key string, value interface{}) {
	if v != nil {
		v.Set(key, value)
	}
}

// BindPFlag is reserved for future use if we want to bind Cobra flags directly to Viper
// For now, we handle flag precedence manually in PersistentPreRun
// Uncomment and implement if needed:
//
// func BindPFlag(key string, flag *pflag.Flag) error {
// 	if v == nil {
// 		return fmt.Errorf("viper not initialized")
// 	}
// 	return v.BindPFlag(key, flag)
// }

// AllSettings returns all configuration settings as a map
func AllSettings() map[string]interface{} {
	if v == nil {
		return map[string]interface{}{}
	}
	return v.AllSettings()
}

// GetIssuePrefix returns the issue-prefix from config.yaml
// This is the canonical source of truth for the project's issue prefix
func GetIssuePrefix() string {
	return GetString("issue-prefix")
}

// SetIssuePrefix updates the issue-prefix in config.yaml
// This is the source of truth for the project's issue prefix
// In environments without .beads directory, updates viper in-memory only
func SetIssuePrefix(prefix string) error {
	// Validate prefix (basic check - actual validation in types.Issue)
	if prefix == "" {
		return fmt.Errorf("issue prefix cannot be empty")
	}

	// Find the .beads directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	var beadsDir string
	for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, ".beads")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			beadsDir = candidate
			break
		}
	}

	if beadsDir == "" {
		// No .beads directory found - update in-memory only (for tests)
		Set("issue-prefix", prefix)
		return nil
	}

	configPath := filepath.Join(beadsDir, "config.yaml")

	// Read existing config or create new one
	content := ""
	if data, err := os.ReadFile(configPath); err == nil {
		content = string(data)
	}

	// Update or add issue-prefix line
	// Note: This is a simple line-based approach. It preserves comments and formatting
	// but may not handle all YAML edge cases. For most users, this is sufficient.
	lines := []string{}
	if content != "" {
		lines = strings.Split(content, "\n")
	}
	
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Match "issue-prefix:" at start of line (ignoring whitespace and comments)
		if strings.HasPrefix(trimmed, "issue-prefix:") && !strings.HasPrefix(trimmed, "#") {
			lines[i] = fmt.Sprintf("issue-prefix: %q", prefix)
			found = true
			break
		}
	}

	if !found {
		// Add issue-prefix to beginning of file
		if len(lines) > 0 {
			lines = append([]string{fmt.Sprintf("issue-prefix: %q", prefix), ""}, lines...)
		} else {
			lines = []string{fmt.Sprintf("issue-prefix: %q", prefix)}
		}
	}

	content = strings.Join(lines, "\n")

	// Write back to file atomically using temp file + rename
	tmpPath := configPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write temp config file: %w", err)
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		_ = os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("failed to update config.yaml: %w", err)
	}

	// Update in-memory config too
	Set("issue-prefix", prefix)

	return nil
}
