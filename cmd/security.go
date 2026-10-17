package cmd

import (
	"context"
	"fmt"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/store"
	"github.com/spf13/cobra"
)

var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Token encryption key: status and migration",
}

var securityStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show which key encrypts your stored tokens",
	Run: func(cmd *cobra.Command, args []string) {
		src := store.KeySource()
		fmt.Println("Token key source:", src)
		switch src {
		case "legacy":
			fmt.Println("⚠ Tokens still use the built-in default key (anyone with the database file can read them).")
			fmt.Println("  Run: postctl security migrate")
		case "keyfile":
			fmt.Println("Key file:", store.KeyFilePath(), "(keep it private; copy it to your other devices — never into a synced folder)")
		case "env":
			fmt.Println("Using POSTCTL_ENCRYPTION_KEY from the environment.")
		}
	},
}

var securityMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Create a private per-installation key and re-encrypt all stored tokens with it",
	Long: `Creates ~/.config/postctl/encryption.key (random, 0600) and re-encrypts every stored token with it.

If you use postctl on several devices with a synced database, copy the key file to the other devices
(outside the synced folder) BEFORE they next run postctl, otherwise they cannot read the tokens.
Run this once per profile (--profile NAME) — all profiles share the same key file.`,
	Run: func(cmd *cobra.Command, args []string) {
		s, err := store.NewSQLiteStore(config.GetDBPath(), config.Shared())
		if err != nil {
			fmt.Println("Cannot open database:", err)
			return
		}
		defer s.Close()
		n, err := s.MigrateTokenKey(context.Background())
		if err != nil {
			fmt.Println("Migration failed (nothing was changed):", err)
			return
		}
		fmt.Printf("✔ %d token(s) re-encrypted. Key file: %s\n", n, store.KeyFilePath())
		fmt.Println("  Copy this file to your other devices (outside any synced folder); keep a backup.")
	},
}

func init() {
	securityCmd.AddCommand(securityStatusCmd, securityMigrateCmd)
	rootCmd.AddCommand(securityCmd)
}
