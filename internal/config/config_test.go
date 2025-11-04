package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInitialize(t *testing.T) {
	// Test that initialization doesn't error
	err := Initialize()
	if err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}
	
	if v == nil {
		t.Fatal("viper instance is nil after Initialize()")
	}
}

func TestDefaults(t *testing.T) {
	// Reset viper for test isolation
	err := Initialize()
	if err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}
	
	tests := []struct {
		key      string
		expected interface{}
		getter   func(string) interface{}
	}{
		{"json", false, func(k string) interface{} { return GetBool(k) }},
		{"no-daemon", false, func(k string) interface{} { return GetBool(k) }},
		{"no-auto-flush", false, func(k string) interface{} { return GetBool(k) }},
		{"no-auto-import", false, func(k string) interface{} { return GetBool(k) }},
		{"db", "", func(k string) interface{} { return GetString(k) }},
		{"actor", "", func(k string) interface{} { return GetString(k) }},
		{"flush-debounce", 30 * time.Second, func(k string) interface{} { return GetDuration(k) }},
		{"auto-start-daemon", true, func(k string) interface{} { return GetBool(k) }},
	}
	
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := tt.getter(tt.key)
			if got != tt.expected {
				t.Errorf("GetXXX(%q) = %v, want %v", tt.key, got, tt.expected)
			}
		})
	}
}

func TestEnvironmentBinding(t *testing.T) {
	// Test environment variable binding
	tests := []struct {
		envVar   string
		key      string
		value    string
		expected interface{}
		getter   func(string) interface{}
	}{
		{"BD_JSON", "json", "true", true, func(k string) interface{} { return GetBool(k) }},
		{"BD_NO_DAEMON", "no-daemon", "true", true, func(k string) interface{} { return GetBool(k) }},
		{"BD_ACTOR", "actor", "testuser", "testuser", func(k string) interface{} { return GetString(k) }},
		{"BD_DB", "db", "/tmp/test.db", "/tmp/test.db", func(k string) interface{} { return GetString(k) }},
		{"BEADS_FLUSH_DEBOUNCE", "flush-debounce", "10s", 10 * time.Second, func(k string) interface{} { return GetDuration(k) }},
		{"BEADS_AUTO_START_DAEMON", "auto-start-daemon", "false", false, func(k string) interface{} { return GetBool(k) }},
	}
	
	for _, tt := range tests {
		t.Run(tt.envVar, func(t *testing.T) {
			// Set environment variable
			oldValue := os.Getenv(tt.envVar)
			_ = os.Setenv(tt.envVar, tt.value)
			defer os.Setenv(tt.envVar, oldValue)
			
			// Re-initialize viper to pick up env var
			err := Initialize()
			if err != nil {
				t.Fatalf("Initialize() returned error: %v", err)
			}
			
			got := tt.getter(tt.key)
			if got != tt.expected {
				t.Errorf("GetXXX(%q) with %s=%s = %v, want %v", tt.key, tt.envVar, tt.value, got, tt.expected)
			}
		})
	}
}

func TestConfigFile(t *testing.T) {
	// Create a temporary directory for config file
	tmpDir := t.TempDir()
	
	// Create a config file
	configContent := `
json: true
no-daemon: true
actor: configuser
flush-debounce: 15s
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	
	// Change to tmp directory so config file is discovered
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	
	// Create .beads directory
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0750); err != nil {
		t.Fatalf("failed to create .beads directory: %v", err)
	}
	
	// Move config to .beads directory
	beadsConfigPath := filepath.Join(beadsDir, "config.yaml")
	if err := os.Rename(configPath, beadsConfigPath); err != nil {
		t.Fatalf("failed to move config file: %v", err)
	}
	
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	
	// Initialize viper
	err = Initialize()
	if err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}
	
	// Test that config file values are loaded
	if got := GetBool("json"); got != true {
		t.Errorf("GetBool(json) = %v, want true", got)
	}
	
	if got := GetBool("no-daemon"); got != true {
		t.Errorf("GetBool(no-daemon) = %v, want true", got)
	}
	
	if got := GetString("actor"); got != "configuser" {
		t.Errorf("GetString(actor) = %q, want \"configuser\"", got)
	}
	
	if got := GetDuration("flush-debounce"); got != 15*time.Second {
		t.Errorf("GetDuration(flush-debounce) = %v, want 15s", got)
	}
}

func TestConfigPrecedence(t *testing.T) {
	// Create a temporary directory for config file
	tmpDir := t.TempDir()
	
	// Create a config file with json: false
	configContent := `json: false`
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0750); err != nil {
		t.Fatalf("failed to create .beads directory: %v", err)
	}
	
	configPath := filepath.Join(beadsDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	
	// Change to tmp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	
	// Test 1: Config file value (json: false)
	err = Initialize()
	if err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}
	
	if got := GetBool("json"); got != false {
		t.Errorf("GetBool(json) from config file = %v, want false", got)
	}
	
	// Test 2: Environment variable overrides config file
	_ = os.Setenv("BD_JSON", "true")
	defer func() { _ = os.Unsetenv("BD_JSON") }()
	
	err = Initialize()
	if err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}
	
	if got := GetBool("json"); got != true {
		t.Errorf("GetBool(json) with env var = %v, want true (env should override config)", got)
	}
}

func TestSetAndGet(t *testing.T) {
	err := Initialize()
	if err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}
	
	// Test Set and Get
	Set("test-key", "test-value")
	if got := GetString("test-key"); got != "test-value" {
		t.Errorf("GetString(test-key) = %q, want \"test-value\"", got)
	}
	
	Set("test-bool", true)
	if got := GetBool("test-bool"); got != true {
		t.Errorf("GetBool(test-bool) = %v, want true", got)
	}
	
	Set("test-int", 42)
	if got := GetInt("test-int"); got != 42 {
		t.Errorf("GetInt(test-int) = %d, want 42", got)
	}
}

func TestAllSettings(t *testing.T) {
	err := Initialize()
	if err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}
	
	Set("custom-key", "custom-value")
	
	settings := AllSettings()
	if settings == nil {
		t.Fatal("AllSettings() returned nil")
	}
	
	// Check that our custom key is in the settings
	if val, ok := settings["custom-key"]; !ok || val != "custom-value" {
		t.Errorf("AllSettings() missing or incorrect custom-key: got %v", val)
	}
}

// TestSetIssuePrefixEmptyConfig tests SetIssuePrefix with an empty config.yaml file
func TestSetIssuePrefixEmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("failed to create .beads directory: %v", err)
	}

	// Create empty config.yaml
	configPath := filepath.Join(beadsDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create empty config file: %v", err)
	}

	// Change to tmp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Initialize config
	if err := Initialize(); err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}

	// Set issue prefix
	if err := SetIssuePrefix("test"); err != nil {
		t.Fatalf("SetIssuePrefix() returned error: %v", err)
	}

	// Verify file was updated
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	expected := `issue-prefix: "test"`
	contentStr := string(content)
	if !strings.Contains(contentStr, expected) {
		t.Errorf("config file content = %q, want to contain %q", contentStr, expected)
	}

	// Verify in-memory config was updated
	if err := Initialize(); err != nil {
		t.Fatalf("Re-initialize returned error: %v", err)
	}
	
	if got := GetIssuePrefix(); got != "test" {
		t.Errorf("GetIssuePrefix() = %q, want \"test\"", got)
	}
}

// TestSetIssuePrefixExistingConfig tests SetIssuePrefix with existing config.yaml (preserves other settings)
func TestSetIssuePrefixExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("failed to create .beads directory: %v", err)
	}

	// Create config.yaml with existing settings
	configPath := filepath.Join(beadsDir, "config.yaml")
	existingConfig := `# My config
json: true
actor: testuser
# Some comment
`
	if err := os.WriteFile(configPath, []byte(existingConfig), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Change to tmp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Initialize config
	if err := Initialize(); err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}

	// Set issue prefix
	if err := SetIssuePrefix("bd"); err != nil {
		t.Fatalf("SetIssuePrefix() returned error: %v", err)
	}

	// Verify file was updated and other settings preserved
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	contentStr := string(content)
	
	// Check that issue-prefix was added
	if !strings.Contains(contentStr, `issue-prefix: "bd"`) {
		t.Errorf("config file doesn't contain issue-prefix: %q", contentStr)
	}
	
	// Check that existing settings are preserved
	if !strings.Contains(contentStr, "json: true") {
		t.Errorf("config file lost json setting: %q", contentStr)
	}
	if !strings.Contains(contentStr, "actor: testuser") {
		t.Errorf("config file lost actor setting: %q", contentStr)
	}
	if !strings.Contains(contentStr, "# My config") {
		t.Errorf("config file lost comment: %q", contentStr)
	}
}

// TestSetIssuePrefixUpdateExisting tests SetIssuePrefix when issue-prefix already exists
func TestSetIssuePrefixUpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("failed to create .beads directory: %v", err)
	}

	// Create config.yaml with existing issue-prefix
	configPath := filepath.Join(beadsDir, "config.yaml")
	existingConfig := `issue-prefix: "old"
json: true
`
	if err := os.WriteFile(configPath, []byte(existingConfig), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Change to tmp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Initialize config
	if err := Initialize(); err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}

	// Update issue prefix
	if err := SetIssuePrefix("new"); err != nil {
		t.Fatalf("SetIssuePrefix() returned error: %v", err)
	}

	// Verify file was updated
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	contentStr := string(content)
	
	// Check that issue-prefix was updated
	if !strings.Contains(contentStr, `issue-prefix: "new"`) {
		t.Errorf("config file doesn't contain updated issue-prefix: %q", contentStr)
	}
	
	// Check that old prefix is gone
	if strings.Contains(contentStr, `issue-prefix: "old"`) {
		t.Errorf("config file still contains old issue-prefix: %q", contentStr)
	}
	
	// Check that other settings are preserved
	if !strings.Contains(contentStr, "json: true") {
		t.Errorf("config file lost json setting: %q", contentStr)
	}
}

// TestSetIssuePrefixCommented tests SetIssuePrefix with commented issue-prefix line
func TestSetIssuePrefixCommented(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("failed to create .beads directory: %v", err)
	}

	// Create config.yaml with commented issue-prefix
	configPath := filepath.Join(beadsDir, "config.yaml")
	existingConfig := `# issue-prefix: "commented"
json: true
`
	if err := os.WriteFile(configPath, []byte(existingConfig), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Change to tmp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Initialize config
	if err := Initialize(); err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}

	// Set issue prefix
	if err := SetIssuePrefix("active"); err != nil {
		t.Fatalf("SetIssuePrefix() returned error: %v", err)
	}

	// Verify file was updated
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	contentStr := string(content)
	
	// Check that active issue-prefix was added (not updating comment)
	if !strings.Contains(contentStr, `issue-prefix: "active"`) {
		t.Errorf("config file doesn't contain active issue-prefix: %q", contentStr)
	}
	
	// Check that comment is preserved
	if !strings.Contains(contentStr, `# issue-prefix: "commented"`) {
		t.Errorf("config file lost commented line: %q", contentStr)
	}
}

// TestSetIssuePrefixNoBeadsDir tests SetIssuePrefix when .beads directory doesn't exist
func TestSetIssuePrefixNoBeadsDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to tmp directory (no .beads subdirectory)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Initialize config
	if err := Initialize(); err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}

	// Set issue prefix should succeed but only update in-memory
	if err := SetIssuePrefix("test"); err != nil {
		t.Fatalf("SetIssuePrefix() returned error: %v", err)
	}

	// Verify in-memory config was updated
	if got := GetIssuePrefix(); got != "test" {
		t.Errorf("GetIssuePrefix() = %q, want \"test\"", got)
	}

	// Verify no config file was created
	configPath := filepath.Join(tmpDir, ".beads", "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		t.Errorf("config file was created when it shouldn't exist")
	}
}

// TestSetIssuePrefixEmptyPrefix tests SetIssuePrefix with empty prefix (should fail)
func TestSetIssuePrefixEmptyPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("failed to create .beads directory: %v", err)
	}

	// Change to tmp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Initialize config
	if err := Initialize(); err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}

	// Set empty issue prefix should fail
	err = SetIssuePrefix("")
	if err == nil {
		t.Error("SetIssuePrefix(\"\") should return error")
	}
}

// TestGetIssuePrefixFallback tests GetIssuePrefix with various config states
func TestGetIssuePrefixFallback(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		expectedPrefix string
	}{
		{
			name:           "prefix set",
			configContent:  `issue-prefix: "bd"`,
			expectedPrefix: "bd",
		},
		{
			name:           "prefix not set",
			configContent:  `json: true`,
			expectedPrefix: "",
		},
		{
			name:           "empty config",
			configContent:  ``,
			expectedPrefix: "",
		},
		{
			name:           "prefix with quotes",
			configContent:  `issue-prefix: "test-prefix"`,
			expectedPrefix: "test-prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			beadsDir := filepath.Join(tmpDir, ".beads")
			if err := os.MkdirAll(beadsDir, 0755); err != nil {
				t.Fatalf("failed to create .beads directory: %v", err)
			}

			// Create config.yaml
			configPath := filepath.Join(beadsDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("failed to create config file: %v", err)
			}

			// Change to tmp directory
			origDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get working directory: %v", err)
			}
			defer os.Chdir(origDir)
			
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("failed to change directory: %v", err)
			}

			// Initialize config
			if err := Initialize(); err != nil {
				t.Fatalf("Initialize() returned error: %v", err)
			}

			// Get issue prefix
			got := GetIssuePrefix()
			if got != tt.expectedPrefix {
				t.Errorf("GetIssuePrefix() = %q, want %q", got, tt.expectedPrefix)
			}
		})
	}
}
