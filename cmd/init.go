package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gitreport configuration and settings",
	Long:  `Create the default settings and configuration files in the user's home directory.`,
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine user home directory: %w", err)
	}

	gitreportDir := filepath.Join(home, ".gitreport")
	configDir := filepath.Join(gitreportDir, "config")

	// Create directories if they don't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("could not create directories: %w", err)
	}

	// 1. Create ~/.gitreport/setting.json
	settingPath := filepath.Join(gitreportDir, "setting.json")
	settingContent := `{
    "OPENAI_API_KEY": "sk-or-v1-",
	"OPENAI_BASE_URL" : "https://openrouter.ai/api/v1/chat/completions",
	"OPENAI_MODEL": "nvidia/nemotron-3-super-120b-a12b:free"
}`
	if _, err := os.Stat(settingPath); os.IsNotExist(err) {
		if err := os.WriteFile(settingPath, []byte(settingContent), 0644); err != nil {
			return fmt.Errorf("could not write setting.json: %w", err)
		}
		fmt.Printf("Created %s\n", settingPath)
	} else {
		// Even if it exists, let's inform the user we are NOT overwriting it,
		// but since the user specifically asked for this content in the issue,
		// maybe they want it to be created if missing with this specific content.
		fmt.Printf("%s already exists, skipping.\n", settingPath)
	}

	// 2. Create ~/.gitreport/config/gitreport.yaml
	configPath := filepath.Join(configDir, "gitreport.yaml")

	// Try to find the default config file from the project or embedded
	// Since we are in the project root, we can try to read from "config/gitreport.yaml"
	defaultConfigPath := filepath.Join("config", "gitreport.yaml")
	configData, err := os.ReadFile(defaultConfigPath)
	if err != nil {
		// Fallback if local file is missing, we could hardcode a minimal one or error out
		return fmt.Errorf("could not find default config file at %s: %w", defaultConfigPath, err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, configData, 0644); err != nil {
			return fmt.Errorf("could not write gitreport.yaml: %w", configPath, err)
		}
		fmt.Printf("Created %s\n", configPath)
	} else {
		fmt.Printf("%s already exists, skipping.\n", configPath)
	}

	fmt.Println("git report initialized successfully.")
	return nil
}
