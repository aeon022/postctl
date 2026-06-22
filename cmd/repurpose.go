package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/generator"
	"github.com/aeon022/postctl/internal/store"
	"github.com/spf13/cobra"
)

var (
	repurposeTo        string
	repurposeCampaign  string
	repurposeOutputDir string
)

// repurposeCmd represents the repurpose command
var repurposeCmd = &cobra.Command{
	Use:   "repurpose <id>",
	Short: "Repurpose an existing post for other platforms",
	Long:  `Load an existing post from the database and rewrite it using AI for one or more target platforms.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		postID := args[0]
		ctx := context.Background()

		// 1. Validierung: --to Flag darf nicht leer sein
		if repurposeTo == "" {
			reportRepurposeError(fmt.Errorf("missing target platforms. Use --to flag (e.g., --to twitter,linkedin)"), 1)
			return
		}

		// Parsen der Ziel-Plattformen
		var targetPlatforms []string
		for _, part := range strings.Split(repurposeTo, ",") {
			plat := strings.ToLower(strings.TrimSpace(part))
			if plat != "" {
				targetPlatforms = append(targetPlatforms, plat)
			}
		}

		if len(targetPlatforms) == 0 {
			reportRepurposeError(fmt.Errorf("no valid target platforms specified in --to"), 1)
			return
		}

		// 2. Datenbank laden
		dbPath := config.GetDBPath()
		s, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			reportRepurposeError(fmt.Errorf("open store: %w", err), 2)
			return
		}
		defer s.Close()

		// 3. Post abfragen
		post, err := s.GetPost(ctx, postID)
		if err != nil {
			reportRepurposeError(fmt.Errorf("post with ID %q not found: %w", postID, err), 1)
			return
		}

		// 4. Inhalt formatieren
		var srcContent string
		if len(post.Tweets) > 0 {
			var sb strings.Builder
			for _, tweet := range post.Tweets {
				sb.WriteString(fmt.Sprintf("## Tweet %d\n%s\n\n", tweet.Index, tweet.Content))
			}
			srcContent = sb.String()
		} else {
			srcContent = post.Body
		}

		// 5. AI Konfiguration laden und validieren
		aiCfg := generator.GeneratorConfig{
			Provider: config.ActiveConfig.AI.Provider,
			APIKey:   config.ActiveConfig.AI.APIKey,
			Model:    config.ActiveConfig.AI.Model,
			BaseURL:  config.ActiveConfig.AI.BaseURL,
		}

		// Fallback auf Umgebungsvariablen falls leer
		if aiCfg.APIKey == "" {
			if strings.ToLower(aiCfg.Provider) == "claude" {
				aiCfg.APIKey = os.Getenv("ANTHROPIC_API_KEY")
			} else if strings.ToLower(aiCfg.Provider) == "openai" || aiCfg.Provider == "" {
				aiCfg.APIKey = os.Getenv("OPENAI_API_KEY")
			}
		}

		if strings.ToLower(aiCfg.Provider) != "ollama" && aiCfg.APIKey == "" {
			reportRepurposeError(fmt.Errorf("AI API Key is missing. Please set it in config.yaml or as an environment variable (OPENAI_API_KEY/ANTHROPIC_API_KEY)"), 1)
			return
		}

		if FormatFlag != "json" {
			fmt.Fprintf(os.Stderr, "Repurposing post %q to %s using %s...\n", postID, strings.Join(targetPlatforms, ", "), aiCfg.Provider)
		}

		// 6. AI Repurposing ausführen
		result, err := generator.RepurposeContent(ctx, aiCfg, post.Platform, post.Type, post.Title, srcContent, targetPlatforms)
		if err != nil {
			reportRepurposeError(fmt.Errorf("AI repurposing failed: %w", err), 2)
			return
		}

		// 7. Dateien speichern (wenn kein dry-run)
		var writtenFiles []string
		if !DryRunFlag {
			if repurposeOutputDir == "" {
				repurposeOutputDir = "."
			}
			writtenFiles, err = generator.SaveRepurposedToMarkdownFiles(result, repurposeOutputDir, repurposeCampaign)
			if err != nil {
				reportRepurposeError(fmt.Errorf("failed to save repurposed markdown files: %w", err), 2)
				return
			}
		}

		// 8. Erfolg ausgeben
		reportRepurposeSuccess(result, writtenFiles)
	},
}

type repurposeSuccessJSON struct {
	OK       bool                       `json:"ok"`
	DryRun   bool                       `json:"dry_run"`
	Slug     string                     `json:"slug"`
	Campaign string                     `json:"campaign,omitempty"`
	Files    []string                   `json:"files,omitempty"`
	Posts    *generator.RepurposeResult `json:"posts"`
}

type repurposeErrorJSON struct {
	OK     bool     `json:"ok"`
	Code   int      `json:"code"`
	Errors []string `json:"errors"`
}

func reportRepurposeSuccess(result *generator.RepurposeResult, files []string) {
	absFiles := make([]string, len(files))
	for i, f := range files {
		if abs, err := filepath.Abs(f); err == nil {
			absFiles[i] = abs
		} else {
			absFiles[i] = f
		}
	}

	if FormatFlag == "json" {
		out := repurposeSuccessJSON{
			OK:       true,
			DryRun:   DryRunFlag,
			Slug:     result.Slug,
			Campaign: repurposeCampaign,
			Files:    absFiles,
			Posts:    result,
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		prefix := ""
		if DryRunFlag {
			prefix = "[DRY RUN] "
		}
		fmt.Printf("%sSuccessfully repurposed post under slug %q!\n", prefix, result.Slug)
		if !DryRunFlag && len(absFiles) > 0 {
			fmt.Println("Written files:")
			for _, f := range absFiles {
				fmt.Printf(" - %s\n", f)
			}
		} else if DryRunFlag {
			for platform, data := range result.Posts {
				fmt.Printf("\n--- %s ---\n", strings.Title(platform))
				fmt.Printf("Title: %s\n\n%s\n", data.Title, data.Content)
			}
		}
	}
}

func reportRepurposeError(err error, exitCode int) {
	if FormatFlag == "json" {
		out := repurposeErrorJSON{
			OK:     false,
			Code:   exitCode,
			Errors: []string{err.Error()},
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stderr, string(jsonBytes))
	} else {
		fmt.Fprintf(os.Stderr, "Repurpose Error: %v\n", err)
	}
	exitFunc(exitCode)
}

func init() {
	repurposeCmd.Flags().StringVar(&repurposeTo, "to", "", "Target platforms (comma-separated, e.g., twitter,linkedin,threads)")
	repurposeCmd.Flags().StringVar(&repurposeCampaign, "campaign", "", "Campaign slug for the frontmatter")
	repurposeCmd.Flags().StringVar(&repurposeOutputDir, "output-dir", ".", "Directory where markdown files are saved")
	rootCmd.AddCommand(repurposeCmd)
}
