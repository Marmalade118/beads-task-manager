package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steveyegge/beads/internal/config"
	"github.com/steveyegge/beads/internal/storage/sqlite"
)

func TestMigrateCommand(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0750); err != nil {
		t.Fatalf("Failed to create .beads directory: %v", err)
	}

	t.Run("no databases", func(t *testing.T) {
		databases, err := detectDatabases(beadsDir)
		if err != nil {
			t.Fatalf("detectDatabases failed: %v", err)
		}
		if len(databases) != 0 {
			t.Errorf("Expected 0 databases, got %d", len(databases))
		}
	})

	t.Run("single old database", func(t *testing.T) {
		// Create old database
		oldDBPath := filepath.Join(beadsDir, "vc.db")
		store, err := sqlite.New(oldDBPath)
		if err != nil {
			t.Fatalf("Failed to create old database: %v", err)
		}

		// Set old version
		ctx := context.Background()
		if err := store.SetMetadata(ctx, "bd_version", "0.16.0"); err != nil {
			t.Fatalf("Failed to set old version: %v", err)
		}
		_ = store.Close()

		// Detect databases
		databases, err := detectDatabases(beadsDir)
		if err != nil {
			t.Fatalf("detectDatabases failed: %v", err)
		}
		if len(databases) != 1 {
			t.Fatalf("Expected 1 database, got %d", len(databases))
		}
		if databases[0].version != "0.16.0" {
			t.Errorf("Expected version 0.16.0, got %s", databases[0].version)
		}

		// Migrate to beads.db
		targetPath := filepath.Join(beadsDir, "beads.db")
		if err := os.Rename(oldDBPath, targetPath); err != nil {
			t.Fatalf("Failed to migrate database: %v", err)
		}

		// Verify migration
		databases, err = detectDatabases(beadsDir)
		if err != nil {
			t.Fatalf("detectDatabases failed after migration: %v", err)
		}
		if len(databases) != 1 {
			t.Fatalf("Expected 1 database after migration, got %d", len(databases))
		}
		if filepath.Base(databases[0].path) != "beads.db" {
			t.Errorf("Expected beads.db, got %s", filepath.Base(databases[0].path))
		}
	})

	t.Run("version detection", func(t *testing.T) {
		// Test getDBVersion with beads.db from previous test
		dbPath := filepath.Join(beadsDir, "beads.db")
		version := getDBVersion(dbPath)
		if version != "0.16.0" {
			t.Errorf("Expected version 0.16.0, got %s", version)
		}

		// Update version
		store, err := sqlite.New(dbPath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		ctx := context.Background()
		if err := store.SetMetadata(ctx, "bd_version", Version); err != nil {
			t.Fatalf("Failed to update version: %v", err)
		}
		_ = store.Close()

		// Verify updated version
		version = getDBVersion(dbPath)
		if version != Version {
			t.Errorf("Expected version %s, got %s", Version, version)
		}
	})
}

func TestFormatDBList(t *testing.T) {
	dbs := []*dbInfo{
		{path: "/tmp/.beads/beads.db", version: "0.17.5"},
		{path: "/tmp/.beads/old.db", version: "0.16.0"},
	}

	result := formatDBList(dbs)
	if len(result) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(result))
	}

	if result[0]["name"] != "beads.db" {
		t.Errorf("Expected name beads.db, got %s", result[0]["name"])
	}
	if result[0]["version"] != "0.17.5" {
		t.Errorf("Expected version 0.17.5, got %s", result[0]["version"])
	}

	if result[1]["name"] != "old.db" {
		t.Errorf("Expected name old.db, got %s", result[1]["name"])
	}
	if result[1]["version"] != "0.16.0" {
		t.Errorf("Expected version 0.16.0, got %s", result[1]["version"])
	}
}

func TestMigrateRespectsConfigJSON(t *testing.T) {
	// Test that migrate respects custom database name from metadata.json
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0750); err != nil {
		t.Fatalf("Failed to create .beads directory: %v", err)
	}

	// Create metadata.json with custom database name
	configPath := filepath.Join(beadsDir, "metadata.json")
	configData := `{"database": "beady.db", "version": "0.21.1", "jsonl_export": "beady.jsonl"}`
	if err := os.WriteFile(configPath, []byte(configData), 0600); err != nil {
		t.Fatalf("Failed to create metadata.json: %v", err)
	}

	// Create old database with custom name
	oldDBPath := filepath.Join(beadsDir, "beady.db")
	store, err := sqlite.New(oldDBPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	ctx := context.Background()
	if err := store.SetMetadata(ctx, "bd_version", "0.21.1"); err != nil {
		t.Fatalf("Failed to set version: %v", err)
	}
	_ = store.Close()

	// Load config
	cfg, err := loadOrCreateConfig(beadsDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify config respects custom database name
	if cfg.Database != "beady.db" {
		t.Errorf("Expected database name 'beady.db', got %s", cfg.Database)
	}

	expectedPath := filepath.Join(beadsDir, "beady.db")
	actualPath := cfg.DatabasePath(beadsDir)
	if actualPath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, actualPath)
	}

	// Verify database exists at custom path
	if _, err := os.Stat(actualPath); os.IsNotExist(err) {
		t.Errorf("Database does not exist at custom path: %s", actualPath)
	}
}

// TestMigrateConfigToYAMLIntegration tests that bd migrate migrates issue_prefix to config.yaml
func TestMigrateConfigToYAMLIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads directory: %v", err)
	}

	// Change to temp directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create empty config.yaml FIRST (simulating upgrade scenario)
	configPath := filepath.Join(beadsDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create config.yaml: %v", err)
	}

	// Initialize config to pick up the empty config.yaml
	// (This must happen before database open to ensure migration can write to it)
	if err := config.Initialize(); err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	// Create v0.21 style database with issue_prefix in DB
	dbPath := filepath.Join(beadsDir, "beads.db")
	store, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	ctx := context.Background()
	
	// Set issue_prefix in DB (old way)
	if err := store.SetConfig(ctx, "issue_prefix", "old-prefix"); err != nil {
		t.Fatalf("Failed to set issue_prefix in DB: %v", err)
	}

	// Set version to simulate old database
	if err := store.SetMetadata(ctx, "bd_version", "0.21.0"); err != nil {
		t.Fatalf("Failed to set version: %v", err)
	}

	// Close and re-open to trigger migration
	_ = store.Close()

	store, err = sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to re-open database: %v", err)
	}
	defer store.Close()

	// The MigrateConfigToYAML should be called automatically on DB open
	// Give it a moment to complete
	
	// Verify config.yaml now has issue-prefix
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config.yaml: %v", err)
	}

	configStr := string(configContent)

	// The migration should have written issue-prefix to config.yaml
	if !strings.Contains(configStr, `issue-prefix: "old-prefix"`) {
		t.Errorf("config.yaml should contain migrated prefix, got: %s", configStr)
	}
}
