package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/khanalsaroj/gitreport/internal/config"
	"github.com/spf13/cobra"
)

// defaultSettings is the starter settings.json scaffold written by `init`.
// The API key is intentionally left blank for the user to fill in.
const defaultSettings = `{
  "OPENAI_API_KEY": "",
  "OPENAI_BASE_URL": "https://openrouter.ai/api/v1/chat/completions",
  "OPENAI_MODEL": "nvidia/nemotron-3-super-120b-a12b:free"
}
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gitreport configuration and settings",
	Long: `Create the default settings and configuration files in the user's home directory:

  ~/.gitreport/setting.json            API key, base URL, and model
  ~/.gitreport/config/gitreport.yaml   AI prompt templates

Existing files are never overwritten.`,
	Args: cobra.NoArgs,
	RunE: runInit,
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
	// 0700: the directory holds the API key, so restrict it to the owner.
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("could not create %q: %w", configDir, err)
	}

	out := cmd.OutOrStdout()

	// setting.json holds the API key, so it is owner-read/write only (0600).
	settingPath := filepath.Join(gitreportDir, "setting.json")
	if err := writeIfAbsent(out, settingPath, []byte(defaultSettings), 0o600); err != nil {
		return err
	}

	// The prompt config is sourced from the embedded default, so `init` works
	// regardless of the current working directory.
	configPath := filepath.Join(configDir, "gitreport.yaml")
	if err := writeIfAbsent(out, configPath, config.Default, 0o644); err != nil {
		return err
	}

	fmt.Fprintf(out, "\ngitreport initialized successfully.\nNext: add your API key to %s\n", settingPath)
	return nil
}

// writeIfAbsent writes data to path with the given permissions unless the file
// already exists, in which case it is left untouched.
func writeIfAbsent(out io.Writer, path string, data []byte, perm os.FileMode) error {
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(out, "%s already exists, skipping.\n", path)
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("could not stat %q: %w", path, err)
	}

	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("could not write %q: %w", path, err)
	}
	fmt.Fprintf(out, "Created %s\n", path)
	return nil
}
