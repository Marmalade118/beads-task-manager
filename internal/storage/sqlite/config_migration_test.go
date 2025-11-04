package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steveyegge/beads/internal/config"
)

// TestMigrateConfigToYAML_AlreadySet tests that migration is skipped when config.yaml has issue-prefix
func TestMigrateConfigToYAML_AlreadySet(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("failed to create .beads directory: %v", err)
	}

	// Create config.yaml with issue-prefix already set
	configPath := filepath.Join(beadsDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(`issue-prefix: "existing"`), 0644); err != nil {
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
	if err := config.Initialize(); err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}

	// Create test database with prefix in DB
	dbPath := filepath.Join(beadsDir, "test.db")
	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	
	// Set prefix in DB (different from config.yaml)
	if err := store.SetConfig(ctx, "issue_prefix", "db-prefix"); err != nil {
		t.Fatalf("failed to set issue_prefix in DB: %v", err)
	}

	// The migration should have already run during New()
	// Re-initialize config and verify prefix wasn't changed
	if err := config.Initialize(); err != nil {
		t.Fatalf("Re-initialize returned error: %v", err)
	}
	
	if got := config.GetIssuePrefix(); got != "existing" {
		t.Errorf("GetIssuePrefix() = %q, want \"existing\" (should not be overwritten)", got)
	}
}

// TestMigrateConfigToYAML_FromDB tests migration from DB to config.yaml
func TestMigrateConfigToYAML_FromDB(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("failed to create .beads directory: %v", err)
	}

	// Create empty config.yaml
	configPath := filepath.Join(beadsDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
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
	if err := config.Initialize(); err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}

	// Create test database with prefix in DB
	dbPath := filepath.Join(beadsDir, "test.db")
	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	
	// Set prefix in DB
	if err := store.SetConfig(ctx, "issue_prefix", "bd"); err != nil {
		t.Fatalf("failed to set issue_prefix in DB: %v", err)
	}
	
	// Close and re-open to trigger migration
	store.Close()
	
	store, err = New(dbPath)
	if err != nil {
		t.Fatalf("failed to re-open store: %v", err)
	}
	defer store.Close()

	// Re-initialize config and verify prefix was migrated
	if err := config.Initialize(); err != nil {
		t.Fatalf("Re-initialize returned error: %v", err)
	}
	
	if got := config.GetIssuePrefix(); got != "bd" {
		t.Errorf("GetIssuePrefix() = %q, want \"bd\"", got)
	}

	// Verify config file was updated
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	
	if !strings.Contains(string(content), `issue-prefix: "bd"`) {
		t.Errorf("config file doesn't contain migrated prefix: %q", string(content))
	}
}

// TestMigrateConfigToYAML_NoPrefixAnywhere tests migration when no prefix exists anywhere
func TestMigrateConfigToYAML_NoPrefixAnywhere(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("failed to create .beads directory: %v", err)
	}

	// Create empty config.yaml
	configPath := filepath.Join(beadsDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
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
	if err := config.Initialize(); err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}

	// Create test database with no prefix
	dbPath := filepath.Join(beadsDir, "test.db")
	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// The migration runs during New() but should not write anything
	// since there's no prefix anywhere
	
	// Re-initialize config and verify prefix is still empty
	if err := config.Initialize(); err != nil {
		t.Fatalf("Re-initialize returned error: %v", err)
	}
	
	if got := config.GetIssuePrefix(); got != "" {
		t.Errorf("GetIssuePrefix() = %q, want \"\" (no prefix should be set)", got)
	}

	// Verify config file wasn't changed
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	
	if string(content) != "" {
		t.Errorf("config file should be empty but got: %q", string(content))
	}
}

// TestMigrateConfigToYAML_PreservesExistingSettings tests that migration preserves other config settings
func TestMigrateConfigToYAML_PreservesExistingSettings(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("failed to create .beads directory: %v", err)
	}

	// Create config.yaml with existing settings but no issue-prefix
	configPath := filepath.Join(beadsDir, "config.yaml")
	existingConfig := `# My config
json: true
actor: testuser
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
	if err := config.Initialize(); err != nil {
		t.Fatalf("Initialize() returned error: %v", err)
	}

	// Create test database with prefix in DB
	dbPath := filepath.Join(beadsDir, "test.db")
	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	
	// Set prefix in DB
	if err := store.SetConfig(ctx, "issue_prefix", "migrated"); err != nil {
		t.Fatalf("failed to set issue_prefix in DB: %v", err)
	}
	
	// Close and re-open to trigger migration
	store.Close()
	
	store, err = New(dbPath)
	if err != nil {
		t.Fatalf("failed to re-open store: %v", err)
	}
	defer store.Close()

	// Verify config file has migrated prefix AND preserved existing settings
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	
	contentStr := string(content)
	
	if !strings.Contains(contentStr, `issue-prefix: "migrated"`) {
		t.Errorf("config file doesn't contain migrated prefix: %q", contentStr)
	}
	
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
