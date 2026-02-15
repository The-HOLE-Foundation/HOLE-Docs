package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CLIConfig holds user preferences for the godocs CLI
type CLIConfig struct {
	DefaultOutputDir string `json:"default_output_dir"`
	Version          string `json:"version"`
}

// ConfigPath returns the path to the CLI config file
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".godocs", "config.json"), nil
}

// LoadCLIConfig loads the CLI configuration from disk
func LoadCLIConfig() (*CLIConfig, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	// If config doesn't exist, return empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &CLIConfig{Version: "1.0.0"}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg CLIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// SaveCLIConfig saves the CLI configuration to disk
func SaveCLIConfig(cfg *CLIConfig) error {
	configPath, err := ConfigPath()
	if err != nil {
		return err
	}

	// Create .godocs directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	cfg.Version = "1.0.0"
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// PromptForDefaultOutputDir prompts the user to set a default output directory
func PromptForDefaultOutputDir() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("╔════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     Welcome to GoDocs CLI                                   ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Print("Would you like to set a default output directory for PDF rendering? (y/n): ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		fmt.Println()
		fmt.Println("ℹ️  You can always specify the output directory when running godocs:")
		fmt.Println("   godocs render input.pdf ./my-output")
		fmt.Println()
		return "", nil
	}

	// Prompt for directory path
	fmt.Print("\nEnter default output directory path (or press Enter to skip): ")
	dirPath, _ := reader.ReadString('\n')
	dirPath = strings.TrimSpace(dirPath)

	if dirPath == "" {
		fmt.Println()
		return "", nil
	}

	// Expand ~ to home directory
	if strings.HasPrefix(dirPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dirPath = filepath.Join(home, dirPath[1:])
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		fmt.Printf("❌ Error creating directory: %v\n", err)
		return "", nil
	}

	fmt.Printf("✅ Default output directory set to: %s\n", dirPath)
	fmt.Println()
	return dirPath, nil
}

// InitializeCLIConfig initializes the CLI config on first run
func InitializeCLIConfig() (*CLIConfig, error) {
	cfg, err := LoadCLIConfig()
	if err != nil {
		return nil, err
	}

	// If config already exists and has a default, return it
	if cfg.DefaultOutputDir != "" {
		return cfg, nil
	}

	// First run - prompt user
	defaultDir, err := PromptForDefaultOutputDir()
	if err != nil {
		return nil, err
	}

	cfg.DefaultOutputDir = defaultDir
	if err := SaveCLIConfig(cfg); err != nil {
		// Don't fail if we can't save config, just log it
		fmt.Fprintf(os.Stderr, "Warning: could not save config: %v\n", err)
	}

	return cfg, nil
}

// GetOutputDir returns the effective output directory
// Priority: explicit flag > default config > empty (required)
func GetOutputDir(explicitDir string, cfg *CLIConfig) (string, error) {
	// If explicit directory specified, use it
	if explicitDir != "" {
		return explicitDir, nil
	}

	// If default is set, use it
	if cfg != nil && cfg.DefaultOutputDir != "" {
		return cfg.DefaultOutputDir, nil
	}

	// No output directory available
	return "", fmt.Errorf("no output directory specified and no default set")
}
