package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestUserConfig_ClaudeConfigDir(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configContent := `
[claude]
config_dir = "~/.claude-work"

[tools.test]
command = "test"
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test parsing
	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.Claude.ConfigDir != "~/.claude-work" {
		t.Errorf("Claude.ConfigDir = %s, want ~/.claude-work", config.Claude.ConfigDir)
	}
}

func TestUserConfig_ClaudeConfigDirEmpty(t *testing.T) {
	// Test with no Claude section
	tmpDir := t.TempDir()
	configContent := `
[tools.test]
command = "test"
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.Claude.ConfigDir != "" {
		t.Errorf("Claude.ConfigDir = %s, want empty string", config.Claude.ConfigDir)
	}
}

func TestGlobalSearchConfig(t *testing.T) {
	// Create temp config with global search settings
	tmpDir := t.TempDir()
	configContent := `
[global_search]
enabled = true
tier = "auto"
memory_limit_mb = 150
recent_days = 60
index_rate_limit = 30
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test parsing
	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if !config.GlobalSearch.Enabled {
		t.Error("Expected GlobalSearch.Enabled to be true")
	}
	if config.GlobalSearch.Tier != "auto" {
		t.Errorf("Expected tier 'auto', got %q", config.GlobalSearch.Tier)
	}
	if config.GlobalSearch.MemoryLimitMB != 150 {
		t.Errorf("Expected MemoryLimitMB 150, got %d", config.GlobalSearch.MemoryLimitMB)
	}
	if config.GlobalSearch.RecentDays != 60 {
		t.Errorf("Expected RecentDays 60, got %d", config.GlobalSearch.RecentDays)
	}
	if config.GlobalSearch.IndexRateLimit != 30 {
		t.Errorf("Expected IndexRateLimit 30, got %d", config.GlobalSearch.IndexRateLimit)
	}
}

func TestGlobalSearchConfigDefaults(t *testing.T) {
	// Config without global_search section should parse with zero values
	// (defaults are applied by LoadUserConfig, not parsing)
	tmpDir := t.TempDir()
	configContent := `default_tool = "claude"`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// When parsing directly without LoadUserConfig, values should be zero
	if config.GlobalSearch.Enabled {
		t.Error("GlobalSearch.Enabled should be false when not specified (zero value)")
	}
	if config.GlobalSearch.MemoryLimitMB != 0 {
		t.Errorf("Expected default MemoryLimitMB 0 (zero value), got %d", config.GlobalSearch.MemoryLimitMB)
	}
}

func TestGlobalSearchConfigDisabled(t *testing.T) {
	// Test explicitly disabling global search
	tmpDir := t.TempDir()
	configContent := `
[global_search]
enabled = false
tier = "disabled"
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.GlobalSearch.Enabled {
		t.Error("Expected GlobalSearch.Enabled to be false")
	}
	if config.GlobalSearch.Tier != "disabled" {
		t.Errorf("Expected tier 'disabled', got %q", config.GlobalSearch.Tier)
	}
}

func TestSaveUserConfig(t *testing.T) {
	// Setup: use temp directory
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Clear cache
	ClearUserConfigCache()

	// Create agent-deck directory
	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)

	// Create config to save
	dangerousModeBool := true
	config := &UserConfig{
		DefaultTool: "claude",
		Claude: ClaudeSettings{
			DangerousMode: &dangerousModeBool,
			ConfigDir:     "~/.claude-work",
		},
		Logs: LogSettings{
			MaxSizeMB:     20,
			MaxLines:      5000,
			RemoveOrphans: true,
		},
	}

	// Save it
	err := SaveUserConfig(config)
	if err != nil {
		t.Fatalf("SaveUserConfig failed: %v", err)
	}

	// Clear cache and reload
	ClearUserConfigCache()
	loaded, err := LoadUserConfig()
	if err != nil {
		t.Fatalf("LoadUserConfig failed: %v", err)
	}

	// Verify values
	if loaded.DefaultTool != "claude" {
		t.Errorf("DefaultTool: got %q, want %q", loaded.DefaultTool, "claude")
	}
	if !loaded.Claude.GetDangerousMode() {
		t.Error("DangerousMode should be true")
	}
	if loaded.Claude.ConfigDir != "~/.claude-work" {
		t.Errorf("ConfigDir: got %q, want %q", loaded.Claude.ConfigDir, "~/.claude-work")
	}
	if loaded.Logs.MaxSizeMB != 20 {
		t.Errorf("MaxSizeMB: got %d, want %d", loaded.Logs.MaxSizeMB, 20)
	}
}

func TestGetTheme_Default(t *testing.T) {
	// Setup: use temp directory with no config
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	theme := GetTheme()
	if theme != "dark" {
		t.Errorf("GetTheme: got %q, want %q", theme, "dark")
	}
}

func TestGetTheme_Light(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	// Create config with light theme
	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)
	config := &UserConfig{Theme: "light"}
	_ = SaveUserConfig(config)
	ClearUserConfigCache()

	theme := GetTheme()
	if theme != "light" {
		t.Errorf("GetTheme: got %q, want %q", theme, "light")
	}
}

func TestWorktreeConfig(t *testing.T) {
	// Create temp config with worktree settings
	tmpDir := t.TempDir()
	configContent := `
[worktree]
default_location = "subdirectory"
auto_cleanup = false
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test parsing
	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.Worktree.DefaultLocation != "subdirectory" {
		t.Errorf("Expected DefaultLocation 'subdirectory', got %q", config.Worktree.DefaultLocation)
	}
	if config.Worktree.AutoCleanup {
		t.Error("Expected AutoCleanup to be false")
	}
}

func TestWorktreeConfigDefaults(t *testing.T) {
	// Config without worktree section should parse with zero values
	// (defaults are applied by GetWorktreeSettings, not parsing)
	tmpDir := t.TempDir()
	configContent := `default_tool = "claude"`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// When parsing directly without GetWorktreeSettings, values should be zero
	if config.Worktree.DefaultLocation != "" {
		t.Errorf("Expected empty DefaultLocation (zero value), got %q", config.Worktree.DefaultLocation)
	}
	if config.Worktree.AutoCleanup {
		t.Error("AutoCleanup should be false when not specified (zero value)")
	}
}

func TestGetWorktreeSettings(t *testing.T) {
	// Setup: use temp directory with no config
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	settings := GetWorktreeSettings()
	if settings.DefaultLocation != "subdirectory" {
		t.Errorf("GetWorktreeSettings DefaultLocation: got %q, want %q", settings.DefaultLocation, "subdirectory")
	}
	if !settings.AutoCleanup {
		t.Error("GetWorktreeSettings AutoCleanup: should default to true")
	}
}

func TestGetWorktreeSettings_FromConfig(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	// Create config with custom worktree settings
	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)
	config := &UserConfig{
		Worktree: WorktreeSettings{
			DefaultLocation: "subdirectory",
			AutoCleanup:     false,
		},
	}
	_ = SaveUserConfig(config)
	ClearUserConfigCache()

	settings := GetWorktreeSettings()
	if settings.DefaultLocation != "subdirectory" {
		t.Errorf("GetWorktreeSettings DefaultLocation: got %q, want %q", settings.DefaultLocation, "subdirectory")
	}
	if settings.AutoCleanup {
		t.Error("GetWorktreeSettings AutoCleanup: should be false from config")
	}
}

// ============================================================================
// Preview Settings Tests
// ============================================================================

func TestPreviewSettings(t *testing.T) {
	// Create temp config
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	// Write config with preview settings
	content := `
[preview]
show_output = true
show_analytics = false
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.Preview.ShowOutput == nil || !*config.Preview.ShowOutput {
		t.Error("Expected Preview.ShowOutput to be true")
	}
	if config.Preview.ShowAnalytics == nil {
		t.Error("Expected Preview.ShowAnalytics to be set")
	} else if *config.Preview.ShowAnalytics {
		t.Error("Expected Preview.ShowAnalytics to be false")
	}
}

func TestPreviewSettingsDefaults(t *testing.T) {
	cfg := &UserConfig{}

	// Default: output ON, analytics OFF
	if !cfg.GetShowOutput() {
		t.Error("GetShowOutput should default to true")
	}
	if cfg.GetShowAnalytics() {
		t.Error("GetShowAnalytics should default to false")
	}
}

func TestPreviewSettingsExplicitTrue(t *testing.T) {
	// Test when analytics is explicitly set to true
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `
[preview]
show_output = false
show_analytics = true
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.GetShowOutput() {
		t.Error("GetShowOutput should be false")
	}
	if !config.GetShowAnalytics() {
		t.Error("GetShowAnalytics should be true when explicitly set")
	}
}

func TestPreviewSettingsNotSet(t *testing.T) {
	// Test when preview section exists but analytics is not set
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `
[preview]
show_output = true
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if !config.GetShowOutput() {
		t.Error("GetShowOutput should be true")
	}
	// When not set, ShowAnalytics should default to false
	if config.GetShowAnalytics() {
		t.Error("GetShowAnalytics should default to false when not set")
	}
}

func TestGetPreviewSettings(t *testing.T) {
	// Setup: use temp directory with no config
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	// With no config, should return defaults (output true, analytics false)
	settings := GetPreviewSettings()
	if !settings.GetShowOutput() {
		t.Error("GetPreviewSettings ShowOutput: should default to true")
	}
	if settings.GetShowAnalytics() {
		t.Error("GetPreviewSettings ShowAnalytics: should default to false")
	}
}

func TestGetPreviewSettings_FromConfig(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	// Create config with custom preview settings
	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)

	// Write config directly to test explicit false
	configPath := filepath.Join(agentDeckDir, "config.toml")
	content := `
[preview]
show_output = true
show_analytics = false
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	ClearUserConfigCache()

	settings := GetPreviewSettings()
	if !settings.GetShowOutput() {
		t.Error("GetPreviewSettings ShowOutput: should be true from config")
	}
	if settings.GetShowAnalytics() {
		t.Error("GetPreviewSettings ShowAnalytics: should be false from config")
	}
}

// ============================================================================
// Notifications Settings Tests
// ============================================================================

func TestNotificationsConfig_Defaults(t *testing.T) {
	// Test that default values are applied when section not present
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	// With no config file, GetNotificationsSettings should return defaults
	settings := GetNotificationsSettings()
	if !settings.Enabled {
		t.Error("notifications should be enabled by default")
	}
	if settings.MaxShown != 6 {
		t.Errorf("max_shown should default to 6, got %d", settings.MaxShown)
	}
}

func TestNotificationsConfig_FromTOML(t *testing.T) {
	// Test parsing explicit TOML config
	tmpDir := t.TempDir()
	configContent := `
[notifications]
enabled = true
max_shown = 4
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if !config.Notifications.Enabled {
		t.Error("Expected Notifications.Enabled to be true")
	}
	if config.Notifications.MaxShown != 4 {
		t.Errorf("Expected MaxShown 4, got %d", config.Notifications.MaxShown)
	}
}

func TestGetNotificationsSettings(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	// Create config with custom notification settings
	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)

	configPath := filepath.Join(agentDeckDir, "config.toml")
	content := `
[notifications]
enabled = true
max_shown = 8
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	ClearUserConfigCache()

	settings := GetNotificationsSettings()
	if !settings.Enabled {
		t.Error("GetNotificationsSettings Enabled: should be true from config")
	}
	if settings.MaxShown != 8 {
		t.Errorf("GetNotificationsSettings MaxShown: got %d, want 8", settings.MaxShown)
	}
}

func TestGetNotificationsSettings_PartialConfig(t *testing.T) {
	// Test that missing fields get defaults
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	ClearUserConfigCache()

	agentDeckDir := filepath.Join(tempDir, ".agent-deck")
	_ = os.MkdirAll(agentDeckDir, 0700)

	// Config with only enabled set, max_shown should get default
	configPath := filepath.Join(agentDeckDir, "config.toml")
	content := `
[notifications]
enabled = true
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	ClearUserConfigCache()

	settings := GetNotificationsSettings()
	if !settings.Enabled {
		t.Error("GetNotificationsSettings Enabled: should be true")
	}
	if settings.MaxShown != 6 {
		t.Errorf("GetNotificationsSettings MaxShown: should default to 6, got %d", settings.MaxShown)
	}
}

// ============================================================================
// AI Settings Tests
// ============================================================================

func TestParseAIConfig(t *testing.T) {
	// Test parsing complete AI config with all fields
	tmpDir := t.TempDir()
	configContent := `
[ai]
enabled = true
provider = "anthropic"
api_key = "${ANTHROPIC_API_KEY}"
model = "claude-opus-4-5-20250514"
max_tokens_per_request = 4096
daily_token_limit = 100000
request_timeout = 30

[ai.observation]
persist = true
retention_count = 100
max_size_bytes = 51200

[ai.watch]
enabled = true
max_concurrent_goals = 10
default_interval = 5
default_timeout = 3600
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test parsing
	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// Verify AI settings
	if config.AI == nil {
		t.Fatal("Expected AI settings to be parsed, got nil")
	}

	if config.AI.Enabled == nil || !*config.AI.Enabled {
		t.Error("Expected AI.Enabled to be true")
	}
	if config.AI.Provider != "anthropic" {
		t.Errorf("Expected Provider 'anthropic', got %q", config.AI.Provider)
	}
	if config.AI.APIKey != "${ANTHROPIC_API_KEY}" {
		t.Errorf("Expected APIKey '${ANTHROPIC_API_KEY}', got %q", config.AI.APIKey)
	}
	if config.AI.Model != "claude-opus-4-5-20250514" {
		t.Errorf("Expected Model 'claude-opus-4-5-20250514', got %q", config.AI.Model)
	}
	if config.AI.MaxTokensPerRequest == nil || *config.AI.MaxTokensPerRequest != 4096 {
		t.Error("Expected MaxTokensPerRequest to be 4096")
	}
	if config.AI.DailyTokenLimit == nil || *config.AI.DailyTokenLimit != 100000 {
		t.Error("Expected DailyTokenLimit to be 100000")
	}
	if config.AI.RequestTimeout == nil || *config.AI.RequestTimeout != 30 {
		t.Error("Expected RequestTimeout to be 30")
	}

	// Verify Observation settings
	if config.AI.Observation == nil {
		t.Fatal("Expected Observation settings to be parsed, got nil")
	}
	if config.AI.Observation.Persist == nil || !*config.AI.Observation.Persist {
		t.Error("Expected Observation.Persist to be true")
	}
	if config.AI.Observation.RetentionCount == nil || *config.AI.Observation.RetentionCount != 100 {
		t.Error("Expected Observation.RetentionCount to be 100")
	}
	if config.AI.Observation.MaxSizeBytes == nil || *config.AI.Observation.MaxSizeBytes != 51200 {
		t.Error("Expected Observation.MaxSizeBytes to be 51200")
	}

	// Verify Watch settings
	if config.AI.Watch == nil {
		t.Fatal("Expected Watch settings to be parsed, got nil")
	}
	if config.AI.Watch.Enabled == nil || !*config.AI.Watch.Enabled {
		t.Error("Expected Watch.Enabled to be true")
	}
	if config.AI.Watch.MaxConcurrentGoals == nil || *config.AI.Watch.MaxConcurrentGoals != 10 {
		t.Error("Expected Watch.MaxConcurrentGoals to be 10")
	}
	if config.AI.Watch.DefaultInterval == nil || *config.AI.Watch.DefaultInterval != 5 {
		t.Error("Expected Watch.DefaultInterval to be 5")
	}
	if config.AI.Watch.DefaultTimeout == nil || *config.AI.Watch.DefaultTimeout != 3600 {
		t.Error("Expected Watch.DefaultTimeout to be 3600")
	}
}

func TestParseAIConfig_OmittedFields(t *testing.T) {
	// Test that omitted pointer fields are nil (not zero values)
	tmpDir := t.TempDir()
	configContent := `
[ai]
provider = "anthropic"
model = "claude-opus-4-5-20250514"
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.AI == nil {
		t.Fatal("Expected AI settings to be parsed, got nil")
	}

	// Pointer fields should be nil when omitted
	if config.AI.Enabled != nil {
		t.Error("Expected AI.Enabled to be nil when omitted")
	}
	if config.AI.MaxTokensPerRequest != nil {
		t.Error("Expected AI.MaxTokensPerRequest to be nil when omitted")
	}
	if config.AI.DailyTokenLimit != nil {
		t.Error("Expected AI.DailyTokenLimit to be nil when omitted")
	}
	if config.AI.RequestTimeout != nil {
		t.Error("Expected AI.RequestTimeout to be nil when omitted")
	}

	// Nested sections should be nil when omitted
	if config.AI.Observation != nil {
		t.Error("Expected AI.Observation to be nil when omitted")
	}
	if config.AI.Watch != nil {
		t.Error("Expected AI.Watch to be nil when omitted")
	}
}

func TestParseAIConfig_PartialObservation(t *testing.T) {
	// Test parsing with only some observation fields set
	tmpDir := t.TempDir()
	configContent := `
[ai]
provider = "anthropic"

[ai.observation]
persist = false
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	var config UserConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if config.AI == nil || config.AI.Observation == nil {
		t.Fatal("Expected Observation settings to be parsed")
	}

	if config.AI.Observation.Persist == nil || *config.AI.Observation.Persist {
		t.Error("Expected Observation.Persist to be false")
	}
	// Omitted fields should be nil
	if config.AI.Observation.RetentionCount != nil {
		t.Error("Expected Observation.RetentionCount to be nil when omitted")
	}
	if config.AI.Observation.MaxSizeBytes != nil {
		t.Error("Expected Observation.MaxSizeBytes to be nil when omitted")
	}
}
