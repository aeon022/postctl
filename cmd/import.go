package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/markdown"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
	"github.com/spf13/cobra"
)

// importCmd repräsentiert den Import-Befehl
var importCmd = &cobra.Command{
	Use:   "import <path>",
	Short: "Import social media posts from Markdown files",
	Long:  `Scan a file or directory for Markdown posts, validate them, and import them into the database.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetPath := args[0]
		ctx := context.Background()

		// 1. Dateien sammeln
		var files []string
		info, err := os.Stat(targetPath)
		if err != nil {
			reportError(fmt.Errorf("stat target path: %w", err), 1)
			return
		}

		if info.IsDir() {
			err = filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && (strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".markdown")) {
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				reportError(fmt.Errorf("walk directory: %w", err), 1)
				return
			}
		} else {
			files = append(files, targetPath)
		}

		// 2. Parsen und Validieren
		var posts []models.Post
		var validationErrors []string

		for _, file := range files {
			filePosts, err := markdown.ParseFile(file)
			if err != nil {
				validationErrors = append(validationErrors, fmt.Errorf("file %s: %w", file, err).Error())
				continue
			}

			postDir := filepath.Dir(file)

			for _, post := range filePosts {
				// Validierung: Twitter Zeichenlänge
				if post.Platform == models.PlatformTwitter {
					for _, tweet := range post.Tweets {
						if !tweet.IsValid() {
							validationErrors = append(validationErrors, fmt.Sprintf(
								"file %s: tweet %d in post %s is too long (%d chars, max 280)",
								file, tweet.Index, post.ID, tweet.CharCount(),
							))
						}
					}
				}

				// Validierung: Bilder existieren
				for _, img := range post.Images {
					if !checkImageExists(postDir, img, config.ActiveConfig.Defaults.ImageDir) {
						validationErrors = append(validationErrors, fmt.Sprintf(
							"file %s: image %q in post %s does not exist",
							file, img, post.ID,
						))
					}
				}

				// Validierung: Inline-Bilder der Tweets existieren
				if post.Platform == models.PlatformTwitter {
					for _, tweet := range post.Tweets {
						if tweet.Image != "" && !checkImageExists(postDir, tweet.Image, config.ActiveConfig.Defaults.ImageDir) {
							validationErrors = append(validationErrors, fmt.Sprintf(
								"file %s: inline image %q in tweet %d of post %s does not exist",
								file, tweet.Image, tweet.Index, post.ID,
							))
						}
					}
				}

				posts = append(posts, post)
			}
		}

		// 3. Fehlerbehandlung bei Validierungsfehlern
		if len(validationErrors) > 0 {
			reportValidationErrors(validationErrors, len(files))
			os.Exit(1)
		}

		// 4. In die DB schreiben (wenn kein dry-run)
		if !DryRunFlag {
			dbPath := config.GetDBPath()
			s, err := store.NewSQLiteStore(dbPath)
			if err != nil {
				reportError(fmt.Errorf("open store: %w", err), 2)
				return
			}
			defer s.Close()

			for _, post := range posts {
				if err := s.SavePost(ctx, &post); err != nil {
					reportError(fmt.Errorf("save post %s: %w", post.ID, err), 2)
					return
				}
			}
		}

		// 5. Erfolgsmeldung ausgeben
		reportSuccess(len(files), len(posts))
	},
}

// checkImageExists prüft, ob das Bild an einem der erwarteten Orte existiert
func checkImageExists(postDir, imagePath, configImageDir string) bool {
	if filepath.IsAbs(imagePath) {
		_, err := os.Stat(imagePath)
		return err == nil
	}
	// Relativ zum Verzeichnis der Markdown-Datei
	if _, err := os.Stat(filepath.Join(postDir, imagePath)); err == nil {
		return true
	}
	// Relativ zum aktuellen Arbeitsverzeichnis
	if _, err := os.Stat(imagePath); err == nil {
		return true
	}
	// Relativ zum konfigurierten Image-Verzeichnis
	if configImageDir != "" {
		if _, err := os.Stat(filepath.Join(configImageDir, imagePath)); err == nil {
			return true
		}
	}
	return false
}

// JSON-Strukturen für CLI Output

type importSuccessJSON struct {
	OK            bool `json:"ok"`
	DryRun        bool `json:"dry_run"`
	FilesScanned  int  `json:"files_scanned"`
	PostsImported int  `json:"posts_imported"`
}

type importErrorJSON struct {
	OK     bool     `json:"ok"`
	Code   int      `json:"code"`
	Errors []string `json:"errors"`
}

func reportSuccess(filesScanned, postsImported int) {
	if FormatFlag == "json" {
		out := importSuccessJSON{
			OK:            true,
			DryRun:        DryRunFlag,
			FilesScanned:  filesScanned,
			PostsImported: postsImported,
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		prefix := ""
		if DryRunFlag {
			prefix = "[DRY RUN] "
		}
		action := "imported"
		if DryRunFlag {
			action = "validated"
		}
		fmt.Printf("%sSuccessfully %s %d posts from %d files.\n", prefix, action, postsImported, filesScanned)
	}
}

func reportValidationErrors(errs []string, filesScanned int) {
	if FormatFlag == "json" {
		out := importErrorJSON{
			OK:     false,
			Code:   1,
			Errors: errs,
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stderr, string(jsonBytes))
	} else {
		fmt.Fprintln(os.Stderr, "Validation failed:")
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, " - %s\n", err)
		}
		fmt.Fprintf(os.Stderr, "\nScanned %d files. No posts were saved.\n", filesScanned)
	}
}

func reportError(err error, exitCode int) {
	if FormatFlag == "json" {
		out := importErrorJSON{
			OK:     false,
			Code:   exitCode,
			Errors: []string{err.Error()},
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stderr, string(jsonBytes))
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	os.Exit(exitCode)
}

func init() {
	rootCmd.AddCommand(importCmd)
}
