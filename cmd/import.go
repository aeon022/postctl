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
	"github.com/aeon022/postctl/internal/scheduler"
	"github.com/aeon022/postctl/internal/store"
	"github.com/spf13/cobra"
)

var isInteractiveImport bool

// importCmd repräsentiert den Import-Befehl
var importCmd = &cobra.Command{
	Use:   "import [path]",
	Short: "Import social media posts from Markdown files",
	Long:  `Scan a file or directory for Markdown posts, validate them, and import them into the database.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var targetPath string
		isInteractiveImport = len(args) == 0

		if !isInteractiveImport {
			targetPath = args[0]
		} else {
			// Terminal komplett leeren (wie clear)
			fmt.Print("\033[H\033[2J\033[3J")

			// Hilfe-Header ausgeben
			fmt.Println("=== postctl BEITRAGS-IMPORT / POST IMPORT ===")
			fmt.Println("Beschreibung: Importiert Social-Media-Beiträge aus Markdown-Dateien oder ganzen Ordnern.")
			fmt.Println("Hilfe: Du kannst den Pfad zu einer Datei oder einem Ordner einfach per Drag & Drop")
			fmt.Println("       aus dem Finder direkt in dieses Terminalfenster schieben/ziehen!")
			fmt.Println("       Drücke Enter ohne Eingabe, um den Vorgang abzubrechen.")
			fmt.Println("==========================================================================")
			fmt.Println()
			fmt.Print("➔ Pfad oder Ordner: ")

			fmt.Scanln(&targetPath)
			targetPath = strings.TrimSpace(targetPath)
			
			// Anführungszeichen entfernen, die Terminals bei Drag & Drop um Pfade mit Leerzeichen setzen
			targetPath = strings.ReplaceAll(targetPath, "\"", "")
			targetPath = strings.ReplaceAll(targetPath, "'", "")
			targetPath = strings.TrimSpace(targetPath)
		}

		if targetPath == "" {
			fmt.Println("\nImport abgebrochen.")
			if isInteractiveImport {
				fmt.Println("Drücke Enter, um zur TUI zurückzukehren...")
				var wait string
				fmt.Scanln(&wait)
			}
			os.Exit(0)
		}

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

			for j := range filePosts {
				post := &filePosts[j]
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

				// Validierung: Bluesky Zeichenlänge
				if post.Platform == models.PlatformBluesky {
					if len([]rune(post.Body)) > 300 {
						validationErrors = append(validationErrors, fmt.Sprintf(
							"file %s: Bluesky post %s is too long (%d chars, max 300)",
							file, post.ID, len([]rune(post.Body)),
						))
					}
				}

				// Validierung: Threads Zeichenlänge
				if post.Platform == models.PlatformThreads {
					if len([]rune(post.Body)) > 500 {
						validationErrors = append(validationErrors, fmt.Sprintf(
							"file %s: Threads post %s is too long (%d chars, max 500)",
							file, post.ID, len([]rune(post.Body)),
						))
					}
				}

				// Validierung: Mastodon Zeichenlänge
				if post.Platform == models.PlatformMastodon {
					if len([]rune(post.Body)) > 500 {
						validationErrors = append(validationErrors, fmt.Sprintf(
							"file %s: Mastodon post %s is too long (%d chars, max 500)",
							file, post.ID, len([]rune(post.Body)),
						))
					}
				}

				// Validierung: Bilder existieren und Pfad auflösen
				for idx, img := range post.Images {
					resolved, ok := resolveImagePath(postDir, img, config.ActiveConfig.Defaults.ImageDir)
					if !ok {
						validationErrors = append(validationErrors, fmt.Sprintf(
							"file %s: image %q in post %s does not exist",
							file, img, post.ID,
						))
					} else {
						post.Images[idx] = resolved
					}
				}

				// Validierung: Inline-Bilder der Tweets existieren und Pfad auflösen
				for idx, tweet := range post.Tweets {
					if tweet.Image != "" {
						resolved, ok := resolveImagePath(postDir, tweet.Image, config.ActiveConfig.Defaults.ImageDir)
						if !ok {
							validationErrors = append(validationErrors, fmt.Sprintf(
								"file %s: inline image %q in tweet %d of post %s does not exist",
								file, tweet.Image, tweet.Index, post.ID,
							))
						} else {
							post.Tweets[idx].Image = resolved
						}
					}
				}

				posts = append(posts, *post)
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
				if post.Status == "queue" {
					slot, err := scheduler.GetNextQueueSlot(ctx, s, post.Platform)
					if err != nil {
						reportError(fmt.Errorf("failed to get queue slot for post %s: %w", post.ID, err), 2)
						return
					}
					post.ScheduledAt = &slot
					post.Status = models.StatusScheduled
				}
				if err := s.SavePost(ctx, &post); err != nil {
					reportError(fmt.Errorf("save post %s: %w", post.ID, err), 2)
					return
				}
			}
		}

		// 5. Erfolgsmeldung ausgeben
		reportSuccess(len(files), len(posts))

		if isInteractiveImport {
			fmt.Println()
			fmt.Println("Drücke Enter, um zur TUI zurückzukehren...")
			var wait string
			fmt.Scanln(&wait)
		}
	},
}

// resolveImagePath prüft, ob das Bild an einem der erwarteten Orte existiert, und gibt den absoluten Pfad zurück
func resolveImagePath(postDir, imagePath, configImageDir string) (string, bool) {
	var resolved string
	if filepath.IsAbs(imagePath) {
		resolved = imagePath
	} else if _, err := os.Stat(filepath.Join(postDir, imagePath)); err == nil {
		resolved = filepath.Join(postDir, imagePath)
	} else if _, err := os.Stat(imagePath); err == nil {
		abs, err := filepath.Abs(imagePath)
		if err == nil {
			resolved = abs
		} else {
			resolved = imagePath
		}
	} else if configImageDir != "" {
		fullPath := filepath.Join(configImageDir, imagePath)
		if _, err := os.Stat(fullPath); err == nil {
			resolved = fullPath
		}
	}

	if resolved != "" {
		abs, err := filepath.Abs(resolved)
		if err == nil {
			return abs, true
		}
		return resolved, true
	}
	return "", false
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
	if isInteractiveImport {
		fmt.Println("\nDrücke Enter, um zur TUI zurückzukehren...")
		var wait string
		fmt.Scanln(&wait)
	}
	os.Exit(1)
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
	if isInteractiveImport {
		fmt.Println("\nDrücke Enter, um zur TUI zurückzukehren...")
		var wait string
		fmt.Scanln(&wait)
	}
	exitFunc(exitCode)
}

func init() {
	rootCmd.AddCommand(importCmd)
}
