package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/generator"
	"github.com/spf13/cobra"
)

var (
	generateCampaign  string
	generateOutputDir string
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate <url>",
	Short: "AI-generierte Posts aus URL/Artikel",
	Long:  `Scrapes the content of the given URL and uses the configured AI model to generate social media posts for Twitter, LinkedIn, and Threads.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetURL := args[0]
		ctx := context.Background()

		// 1. URL validieren
		parsedURL, err := url.ParseRequestURI(targetURL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			reportGenerateError(fmt.Errorf("invalid URL: %s", targetURL), 1)
			return
		}

		// 2. AI Konfiguration laden und validieren
		aiCfg := generator.GeneratorConfig{
			Provider: config.ActiveConfig.AI.Provider,
			APIKey:   config.ActiveConfig.AI.APIKey,
			Model:    config.ActiveConfig.AI.Model,
			BaseURL:  config.ActiveConfig.AI.BaseURL,
		}

		// Fallback auf Umgebungsvariablen falls in config.yaml leer
		if aiCfg.APIKey == "" {
			if strings.ToLower(aiCfg.Provider) == "claude" {
				aiCfg.APIKey = os.Getenv("ANTHROPIC_API_KEY")
			} else if strings.ToLower(aiCfg.Provider) == "openai" || aiCfg.Provider == "" {
				aiCfg.APIKey = os.Getenv("OPENAI_API_KEY")
			}
		}

		// Ollama benötigt keinen Key, aber OpenAI und Claude schon
		if strings.ToLower(aiCfg.Provider) != "ollama" && aiCfg.APIKey == "" {
			reportGenerateError(fmt.Errorf("AI API Key is missing. Please set it in config.yaml or as an environment variable (OPENAI_API_KEY/ANTHROPIC_API_KEY)"), 1)
			return
		}

		// 3. Webpage scrapen (fortschritt an stderr senden)
		if FormatFlag != "json" {
			fmt.Fprintf(os.Stderr, "Scraping page: %s...\n", targetURL)
		}
		scraped, err := generator.ScrapeURL(targetURL)
		if err != nil {
			reportGenerateError(fmt.Errorf("failed to scrape URL: %w", err), 2)
			return
		}

		if FormatFlag != "json" {
			fmt.Fprintf(os.Stderr, "Generating social media posts using %s (%s)...\n", aiCfg.Provider, aiCfg.Model)
		}

		// 4. LLM aufrufen
		posts, err := generator.GenerateContent(ctx, aiCfg, targetURL, scraped.Title, scraped.Content)
		if err != nil {
			reportGenerateError(fmt.Errorf("AI generation failed: %w", err), 2)
			return
		}

		// 5. Dateien schreiben (falls nicht dry-run)
		var writtenFiles []string
		if !DryRunFlag {
			if generateOutputDir == "" {
				generateOutputDir = "."
			}
			writtenFiles, err = generator.SaveToMarkdownFiles(posts, generateOutputDir, generateCampaign)
			if err != nil {
				reportGenerateError(fmt.Errorf("failed to write markdown files: %w", err), 2)
				return
			}
		}

		// 6. Erfolg ausgeben
		reportGenerateSuccess(posts, writtenFiles)
	},
}

type generateSuccessJSON struct {
	OK       bool                      `json:"ok"`
	DryRun   bool                      `json:"dry_run"`
	Slug     string                    `json:"slug"`
	Campaign string                    `json:"campaign,omitempty"`
	Files    []string                  `json:"files,omitempty"`
	Posts    *generator.GeneratedPosts `json:"posts"`
}

type generateErrorJSON struct {
	OK     bool     `json:"ok"`
	Code   int      `json:"code"`
	Errors []string `json:"errors"`
}

func reportGenerateSuccess(posts *generator.GeneratedPosts, files []string) {
	// Falls absolute Pfade gewünscht, konvertieren wir sie
	absFiles := make([]string, len(files))
	for i, f := range files {
		if abs, err := filepath.Abs(f); err == nil {
			absFiles[i] = abs
		} else {
			absFiles[i] = f
		}
	}

	if FormatFlag == "json" {
		out := generateSuccessJSON{
			OK:       true,
			DryRun:   DryRunFlag,
			Slug:     posts.Slug,
			Campaign: generateCampaign,
			Files:    absFiles,
			Posts:    posts,
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		prefix := ""
		if DryRunFlag {
			prefix = "[DRY RUN] "
		}
		fmt.Printf("%sSuccessfully generated social media posts under slug %q!\n", prefix, posts.Slug)
		if !DryRunFlag && len(absFiles) > 0 {
			fmt.Println("Written files:")
			for _, f := range absFiles {
				fmt.Printf(" - %s\n", f)
			}
		} else if DryRunFlag {
			// Preview ausgeben
			fmt.Println("\n--- Twitter ---")
			fmt.Printf("Title: %s\n\n%s\n", posts.Twitter.Title, posts.Twitter.Content)
			fmt.Println("\n--- LinkedIn ---")
			fmt.Printf("Title: %s\n\n%s\n", posts.LinkedIn.Title, posts.LinkedIn.Content)
			fmt.Println("\n--- Threads ---")
			fmt.Printf("Title: %s\n\n%s\n", posts.Threads.Title, posts.Threads.Content)
		}
	}
}

var exitFunc = os.Exit

func reportGenerateError(err error, exitCode int) {
	if FormatFlag == "json" {
		out := generateErrorJSON{
			OK:     false,
			Code:   exitCode,
			Errors: []string{err.Error()},
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stderr, string(jsonBytes))
	} else {
		fmt.Fprintf(os.Stderr, "Generate Error: %v\n", err)
	}
	exitFunc(exitCode)
}

func init() {
	generateCmd.Flags().StringVar(&generateCampaign, "campaign", "", "Campaign slug for the frontmatter")
	generateCmd.Flags().StringVar(&generateOutputDir, "output-dir", ".", "Directory where markdown files are saved")
	rootCmd.AddCommand(generateCmd)
}
