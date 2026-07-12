package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var hookDirFlag string

var gitHookCmd = &cobra.Command{
	Use:   "git-hook",
	Short: "Manage Git hooks for automatic post importing",
	Long:  `Install or uninstall git hooks to automate post importing when changes are committed.`,
}

var installHookCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a post-commit git hook",
	Long:  `Create a post-commit hook in the local .git/hooks directory to auto-import posts from a target directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Verify we are in a Git repository
		if _, err := os.Stat(".git"); os.IsNotExist(err) {
			fmt.Println("Fehler: Du befindest dich nicht im Hauptverzeichnis eines Git-Repositorys (kein .git-Ordner gefunden).")
			os.Exit(1)
		}

		hookPath := filepath.Join(".git", "hooks", "post-commit")
		
		scriptContent := fmt.Sprintf(`#!/bin/sh
# postctl auto-import git hook
# Automatisch generiert von postctl

TARGET_DIR="%s"

if [ ! -d "$TARGET_DIR" ]; then
    exit 0
fi

# Führe den Import im Hintergrund aus
if command -v postctl >/dev/null 2>&1; then
    postctl import "$TARGET_DIR" > /dev/null 2>&1 &
else
    # Fallback: Versuche lokales Binary
    if [ -f ./postctl ]; then
        ./postctl import "$TARGET_DIR" > /dev/null 2>&1 &
    fi
fi
`, hookDirFlag)

		// hooks-Verzeichnis erstellen falls nicht vorhanden
		if err := os.MkdirAll(filepath.Dir(hookPath), 0755); err != nil {
			fmt.Printf("Fehler beim Erstellen des hooks-Verzeichnisses: %v\n", err)
			os.Exit(1)
		}

		// Skript schreiben
		if err := os.WriteFile(hookPath, []byte(scriptContent), 0755); err != nil {
			fmt.Printf("Fehler beim Schreiben des Hooks: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Git-Hook erfolgreich unter %s installiert.\n", hookPath)
		fmt.Printf("  Er wird neue/geänderte Beiträge in '%s/' bei jedem Commit automatisch importieren.\n", hookDirFlag)
	},
}

var uninstallHookCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the post-commit git hook",
	Run: func(cmd *cobra.Command, args []string) {
		hookPath := filepath.Join(".git", "hooks", "post-commit")
		if _, err := os.Stat(hookPath); os.IsNotExist(err) {
			fmt.Println("Kein postctl Git-Hook installiert.")
			return
		}

		if err := os.Remove(hookPath); err != nil {
			fmt.Printf("Fehler beim Löschen des Hooks: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✓ Git-Hook erfolgreich deinstalliert.")
	},
}

func init() {
	installHookCmd.Flags().StringVar(&hookDirFlag, "dir", "posts", "Directory to watch and import posts from")
	gitHookCmd.AddCommand(installHookCmd)
	gitHookCmd.AddCommand(uninstallHookCmd)
	rootCmd.AddCommand(gitHookCmd)
}
